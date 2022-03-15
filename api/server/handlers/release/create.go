package release

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/porter-dev/porter/api/server/authz"
	"github.com/porter-dev/porter/api/server/handlers"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/analytics"
	"github.com/porter-dev/porter/internal/auth/token"
	"github.com/porter-dev/porter/internal/encryption"
	"github.com/porter-dev/porter/internal/helm"
	"github.com/porter-dev/porter/internal/helm/loader"
	"github.com/porter-dev/porter/internal/integrations/ci/actions"
	"github.com/porter-dev/porter/internal/kubernetes/envgroup"
	"github.com/porter-dev/porter/internal/models"
	"github.com/porter-dev/porter/internal/oauth"
	"github.com/porter-dev/porter/internal/registry"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/release"
)

type CreateReleaseHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

type CreateReleaseEnvValues struct {
	Container struct {
		Env struct {
			Normal map[string]string
			Synced []struct {
				Name    string
				Version int
				Keys    []struct {
					Name   string
					Secret bool
				}
			}
		}
	}
}

func NewCreateReleaseHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *CreateReleaseHandler {
	return &CreateReleaseHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *CreateReleaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	namespace := r.Context().Value(types.NamespaceScope).(string)
	operationID := oauth.CreateRandomState()

	c.Config().AnalyticsClient.Track(analytics.ApplicationLaunchStartTrack(
		&analytics.ApplicationLaunchStartTrackOpts{
			ClusterScopedTrackOpts: analytics.GetClusterScopedTrackOpts(user.ID, cluster.ProjectID, cluster.ID),
			FlowID:                 operationID,
		},
	))

	helmAgent, err := c.GetHelmAgent(r, cluster, "")

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	request := &types.CreateReleaseRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	if request.RepoURL == "" {
		request.RepoURL = c.Config().ServerConf.DefaultApplicationHelmRepoURL
	}

	if request.TemplateVersion == "latest" {
		request.TemplateVersion = ""
	}

	chart, err := loader.LoadChartPublic(request.RepoURL, request.TemplateName, request.TemplateVersion)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	registries, err := c.Repo().Registry().ListRegistriesByProjectID(cluster.ProjectID)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	conf := &helm.InstallChartConfig{
		Chart:      chart,
		Name:       request.Name,
		Namespace:  namespace,
		Values:     request.Values,
		Cluster:    cluster,
		Repo:       c.Repo(),
		Registries: registries,
	}

	helmRelease, err := helmAgent.InstallChart(conf, c.Config().DOConf)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			fmt.Errorf("error installing a new chart: %s", err.Error()),
			http.StatusBadRequest,
		))

		return
	}

	release, err := createReleaseFromHelmRelease(c.Config(), cluster.ProjectID, cluster.ID, helmRelease)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	buildSecrets := make(map[string]string)

	// Marshal and unmarshal into a nice struct so we don't need to type assert maps of interfaces
	raw, _ := json.Marshal(request.CreateReleaseBaseRequest.Values)
	envVals := CreateReleaseEnvValues{}
	json.Unmarshal(raw, &envVals)

	// Get all normal secrets
	for key, val := range envVals.Container.Env.Normal {
		if strings.HasPrefix(val, "PORTERSECRET_") {
			// TODO: convert the PORTERSECRET_group_v1 format into the actual value
			buildSecrets[key] = val
		}
	}

	agent, err := c.KubernetesAgentGetter.GetAgent(r, cluster, namespace)

	// Get all synced secrets
	for _, group := range envVals.Container.Env.Synced {
		// Load all secrets from the group
		eg, _ := envgroup.GetEnvGroup(agent, group.Name, namespace, uint(group.Version))
		// TODO: read the secret values so that we can write them to GitHub
		_ = eg
	}

	if request.GithubActionConfig != nil {
		_, _, err := createGitAction(
			c.Config(),
			user.ID,
			cluster.ProjectID,
			cluster.ID,
			request.GithubActionConfig,
			request.Name,
			namespace,
			release,
			buildSecrets,
		)

		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}

	if request.BuildConfig != nil {
		_, err = createBuildConfig(c.Config(), release, request.BuildConfig)
	}

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.Config().AnalyticsClient.Track(analytics.ApplicationLaunchSuccessTrack(
		&analytics.ApplicationLaunchSuccessTrackOpts{
			ApplicationScopedTrackOpts: analytics.GetApplicationScopedTrackOpts(
				user.ID,
				cluster.ProjectID,
				cluster.ID,
				release.Name,
				release.Namespace,
				chart.Metadata.Name,
			),
			FlowID: operationID,
		},
	))
}

func createReleaseFromHelmRelease(
	config *config.Config,
	projectID, clusterID uint,
	helmRelease *release.Release,
) (*models.Release, error) {
	token, err := encryption.GenerateRandomBytes(16)

	if err != nil {
		return nil, err
	}

	// create release with webhook token in db
	image, ok := helmRelease.Config["image"].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("Could not find field image in config")
	}

	repository := image["repository"]
	repoStr, ok := repository.(string)

	if !ok {
		return nil, fmt.Errorf("Could not find field repository in config")
	}

	release := &models.Release{
		ClusterID:    clusterID,
		ProjectID:    projectID,
		Namespace:    helmRelease.Namespace,
		Name:         helmRelease.Name,
		WebhookToken: token,
		ImageRepoURI: repoStr,
	}

	return config.Repo.Release().CreateRelease(release)
}

func createGitAction(
	config *config.Config,
	userID, projectID, clusterID uint,
	request *types.CreateGitActionConfigRequest,
	name, namespace string,
	release *models.Release,
	buildSecrets map[string]string,
) (*types.GitActionConfig, []byte, error) {
	// if the registry was provisioned through Porter, create a repository if necessary
	if release != nil && request.RegistryID != 0 {
		// read the registry
		reg, err := config.Repo.Registry().ReadRegistry(projectID, request.RegistryID)

		if err != nil {
			return nil, nil, err
		}

		_reg := registry.Registry(*reg)
		regAPI := &_reg

		// parse the name from the registry
		nameSpl := strings.Split(request.ImageRepoURI, "/")
		repoName := nameSpl[len(nameSpl)-1]

		err = regAPI.CreateRepository(config.Repo, repoName)

		if err != nil {
			return nil, nil, err
		}
	}

	repoSplit := strings.Split(request.GitRepo, "/")

	if len(repoSplit) != 2 {
		return nil, nil, fmt.Errorf("invalid formatting of repo name")
	}

	// generate porter jwt token
	jwt, err := token.GetTokenForAPI(userID, projectID)

	if err != nil {
		return nil, nil, err
	}

	encoded, err := jwt.EncodeToken(config.TokenConf)

	if err != nil {
		return nil, nil, err
	}

	// create the commit in the git repo
	gaRunner := &actions.GithubActions{
		InstanceName:           config.ServerConf.InstanceName,
		ServerURL:              config.ServerConf.ServerURL,
		GithubOAuthIntegration: nil,
		GithubAppID:            config.GithubAppConf.AppID,
		GithubAppSecretPath:    config.GithubAppConf.SecretPath,
		GithubInstallationID:   request.GitRepoID,
		GitRepoName:            repoSplit[1],
		GitRepoOwner:           repoSplit[0],
		Repo:                   config.Repo,
		ProjectID:              projectID,
		ClusterID:              clusterID,
		ReleaseName:            name,
		ReleaseNamespace:       namespace,
		GitBranch:              request.GitBranch,
		DockerFilePath:         request.DockerfilePath,
		FolderPath:             request.FolderPath,
		ImageRepoURL:           request.ImageRepoURI,
		PorterToken:            encoded,
		Version:                "v0.1.0",
		ShouldCreateWorkflow:   request.ShouldCreateWorkflow,
		DryRun:                 release == nil,
		BuildSecrets:           buildSecrets,
	}

	// Save the github err for after creating the git action config. However, we
	// need to call Setup() in order to get the workflow file before writing the
	// action config, in the case of a dry run, since the dry run does not create
	// a git action config.
	workflowYAML, githubErr := gaRunner.Setup()

	if gaRunner.DryRun {
		if githubErr != nil {
			return nil, nil, githubErr
		}

		return nil, workflowYAML, nil
	}

	// handle write to the database
	ga, err := config.Repo.GitActionConfig().CreateGitActionConfig(&models.GitActionConfig{
		ReleaseID:      release.ID,
		GitRepo:        request.GitRepo,
		GitBranch:      request.GitBranch,
		ImageRepoURI:   request.ImageRepoURI,
		GitRepoID:      request.GitRepoID,
		DockerfilePath: request.DockerfilePath,
		FolderPath:     request.FolderPath,
		IsInstallation: true,
		Version:        "v0.1.0",
	})

	if err != nil {
		return nil, nil, err
	}

	// update the release in the db with the image repo uri
	release.ImageRepoURI = ga.ImageRepoURI

	_, err = config.Repo.Release().UpdateRelease(release)

	if err != nil {
		return nil, nil, err
	}

	if githubErr != nil {
		return nil, nil, githubErr
	}

	return ga.ToGitActionConfigType(), workflowYAML, nil
}

func createBuildConfig(
	config *config.Config,
	release *models.Release,
	bcRequest *types.CreateBuildConfigRequest,
) (*types.BuildConfig, error) {
	data, err := json.Marshal(bcRequest.Config)
	if err != nil {
		return nil, err
	}

	// handle write to the database
	bc, err := config.Repo.BuildConfig().CreateBuildConfig(&models.BuildConfig{
		Builder:    bcRequest.Builder,
		Buildpacks: strings.Join(bcRequest.Buildpacks, ","),
		Config:     data,
	})
	if err != nil {
		return nil, err
	}

	release.BuildConfig = bc.ID

	_, err = config.Repo.Release().UpdateRelease(release)
	if err != nil {
		return nil, err
	}

	return bc.ToBuildConfigType(), nil
}

type containerEnvConfig struct {
	Container struct {
		Env struct {
			Normal map[string]string `yaml:"normal"`
		} `yaml:"env"`
	} `yaml:"container"`
}

func getGARunner(
	config *config.Config,
	userID, projectID, clusterID uint,
	ga *models.GitActionConfig,
	name, namespace string,
	release *models.Release,
	helmRelease *release.Release,
) (*actions.GithubActions, error) {
	cEnv := &containerEnvConfig{}

	rawValues, err := yaml.Marshal(helmRelease.Config)

	if err == nil {
		err = yaml.Unmarshal(rawValues, cEnv)

		// if unmarshal error, just set to empty map
		if err != nil {
			cEnv.Container.Env.Normal = make(map[string]string)
		}
	}

	repoSplit := strings.Split(ga.GitRepo, "/")

	if len(repoSplit) != 2 {
		return nil, fmt.Errorf("invalid formatting of repo name")
	}

	// create the commit in the git repo
	return &actions.GithubActions{
		ServerURL:              config.ServerConf.ServerURL,
		GithubOAuthIntegration: nil,
		BuildEnv:               cEnv.Container.Env.Normal,
		GithubAppID:            config.GithubAppConf.AppID,
		GithubAppSecretPath:    config.GithubAppConf.SecretPath,
		GithubInstallationID:   ga.GitRepoID,
		GitRepoName:            repoSplit[1],
		GitRepoOwner:           repoSplit[0],
		Repo:                   config.Repo,
		ProjectID:              projectID,
		ClusterID:              clusterID,
		ReleaseName:            name,
		GitBranch:              ga.GitBranch,
		DockerFilePath:         ga.DockerfilePath,
		FolderPath:             ga.FolderPath,
		ImageRepoURL:           ga.ImageRepoURI,
		Version:                "v0.1.0",
	}, nil
}
