package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/porter-dev/porter/internal/models"
	"github.com/porter-dev/porter/internal/oauth"
	"github.com/porter-dev/porter/internal/repository"
	"golang.org/x/oauth2"

	ints "github.com/porter-dev/porter/internal/models/integrations"

	ptypes "github.com/porter-dev/porter/api/types"

	"github.com/digitalocean/godo"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
)

// Registry wraps the gorm Registry model
type Registry models.Registry

func GetECRRegistryURL(awsIntRepo repository.AWSIntegrationRepository, projectID, awsIntID uint) (string, error) {
	awsInt, err := awsIntRepo.ReadAWSIntegration(projectID, awsIntID)

	if err != nil {
		return "", err
	}

	sess, err := awsInt.GetSession()

	if err != nil {
		return "", err
	}

	ecrSvc := ecr.New(sess)

	output, err := ecrSvc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})

	if err != nil {
		return "", err
	}

	return *output.AuthorizationData[0].ProxyEndpoint, nil
}

// ListRepositories lists the repositories for a registry
func (r *Registry) ListRepositories(
	repo repository.Repository,
	doAuth *oauth2.Config, // only required if using DOCR
) ([]*ptypes.RegistryRepository, error) {
	// switch on the auth mechanism to get a token
	if r.AWSIntegrationID != 0 {
		return r.listECRRepositories(repo)
	}

	if r.GCPIntegrationID != 0 {
		return r.listGCRRepositories(repo)
	}

	if r.DOIntegrationID != 0 {
		return r.listDOCRRepositories(repo, doAuth)
	}

	if r.BasicIntegrationID != 0 {
		return r.listPrivateRegistryRepositories(repo)
	}

	return nil, fmt.Errorf("error listing repositories")
}

type gcrJWT struct {
	AccessToken  string `json:"token"`
	ExpiresInSec int    `json:"expires_in"`
}

type gcrErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type gcrRepositoryResp struct {
	Repositories []string `json:"repositories"`
	Errors       []gcrErr `json:"errors"`
}

func (r *Registry) GetGCRToken(repo repository.Repository) (*oauth2.Token, error) {
	getTokenCache := r.getTokenCacheFunc(repo)

	gcp, err := repo.GCPIntegration().ReadGCPIntegration(
		r.ProjectID,
		r.GCPIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	// get oauth2 access token
	return gcp.GetBearerToken(
		getTokenCache,
		r.setTokenCacheFunc(repo),
		"https://www.googleapis.com/auth/devstorage.read_write",
	)
}

func (r *Registry) listGCRRepositories(
	repo repository.Repository,
) ([]*ptypes.RegistryRepository, error) {
	gcp, err := repo.GCPIntegration().ReadGCPIntegration(
		r.ProjectID,
		r.GCPIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	// Just use service account key to authenticate, since scopes may not be in place
	// for oauth. This also prevents us from making more requests.
	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		"https://gcr.io/v2/_catalog",
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("_json_key", string(gcp.GCPKeyData))

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	gcrResp := gcrRepositoryResp{}

	if err := json.NewDecoder(resp.Body).Decode(&gcrResp); err != nil {
		return nil, fmt.Errorf("Could not read GCR repositories: %v", err)
	}

	if len(gcrResp.Errors) > 0 {
		errMsg := ""
		for _, gcrErr := range gcrResp.Errors {
			errMsg += fmt.Sprintf(": Code %s, message %s", gcrErr.Code, gcrErr.Message)
		}

		return nil, fmt.Errorf(errMsg)
	}

	res := make([]*ptypes.RegistryRepository, 0)

	parsedURL, err := url.Parse("https://" + r.URL)

	if err != nil {
		return nil, err
	}

	for _, repo := range gcrResp.Repositories {
		res = append(res, &ptypes.RegistryRepository{
			Name: repo,
			URI:  parsedURL.Host + "/" + repo,
		})
	}

	return res, nil
}

func (r *Registry) listECRRepositories(repo repository.Repository) ([]*ptypes.RegistryRepository, error) {
	aws, err := repo.AWSIntegration().ReadAWSIntegration(
		r.ProjectID,
		r.AWSIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	sess, err := aws.GetSession()

	if err != nil {
		return nil, err
	}

	svc := ecr.New(sess)

	resp, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{})

	if err != nil {
		return nil, err
	}

	res := make([]*ptypes.RegistryRepository, 0)

	for _, repo := range resp.Repositories {
		res = append(res, &ptypes.RegistryRepository{
			Name:      *repo.RepositoryName,
			CreatedAt: *repo.CreatedAt,
			URI:       *repo.RepositoryUri,
		})
	}

	return res, nil
}

func (r *Registry) listDOCRRepositories(
	repo repository.Repository,
	doAuth *oauth2.Config,
) ([]*ptypes.RegistryRepository, error) {
	oauthInt, err := repo.OAuthIntegration().ReadOAuthIntegration(
		r.ProjectID,
		r.DOIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	tok, _, err := oauth.GetAccessToken(oauthInt.SharedOAuthModel, doAuth, oauth.MakeUpdateOAuthIntegrationTokenFunction(oauthInt, repo))

	if err != nil {
		return nil, err
	}

	client := godo.NewFromToken(tok)

	urlArr := strings.Split(r.URL, "/")

	if len(urlArr) != 2 {
		return nil, fmt.Errorf("invalid digital ocean registry url")
	}

	name := urlArr[1]

	repos, _, err := client.Registry.ListRepositories(context.TODO(), name, &godo.ListOptions{})

	if err != nil {
		return nil, err
	}

	res := make([]*ptypes.RegistryRepository, 0)

	for _, repo := range repos {
		res = append(res, &ptypes.RegistryRepository{
			Name: repo.Name,
			URI:  r.URL + "/" + repo.Name,
		})
	}

	return res, nil
}

func (r *Registry) listPrivateRegistryRepositories(
	repo repository.Repository,
) ([]*ptypes.RegistryRepository, error) {
	// handle dockerhub different, as it doesn't implement the docker registry http api
	if strings.Contains(r.URL, "docker.io") {
		// in this case, we just return the single dockerhub repository that's linked
		res := make([]*ptypes.RegistryRepository, 0)

		res = append(res, &ptypes.RegistryRepository{
			Name: strings.Split(r.URL, "docker.io/")[1],
			URI:  r.URL,
		})

		return res, nil
	}

	basic, err := repo.BasicIntegration().ReadBasicIntegration(
		r.ProjectID,
		r.BasicIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	// Just use service account key to authenticate, since scopes may not be in place
	// for oauth. This also prevents us from making more requests.
	client := &http.Client{}

	// get the host and scheme to make the request
	parsedURL, err := url.Parse(r.URL)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s://%s/v2/_catalog", parsedURL.Scheme, parsedURL.Host),
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(string(basic.Username), string(basic.Password))

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	// if the status code is 404, fallback to the Docker Hub implementation
	if resp.StatusCode == 404 {
		req, err := http.NewRequest(
			"GET",
			fmt.Sprintf("%s/", r.URL),
			nil,
		)

		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(string(basic.Username), string(basic.Password))

		resp, err = client.Do(req)

		if err != nil {
			return nil, err
		}
	}

	gcrResp := gcrRepositoryResp{}

	if err := json.NewDecoder(resp.Body).Decode(&gcrResp); err != nil {
		return nil, fmt.Errorf("Could not read private registry repositories: %v", err)
	}

	res := make([]*ptypes.RegistryRepository, 0)

	if err != nil {
		return nil, err
	}

	for _, repo := range gcrResp.Repositories {
		res = append(res, &ptypes.RegistryRepository{
			Name: repo,
			URI:  parsedURL.Host + "/" + repo,
		})
	}

	return res, nil
}

func (r *Registry) getTokenCacheFunc(
	repo repository.Repository,
) ints.GetTokenCacheFunc {
	return func() (tok *ints.TokenCache, err error) {
		reg, err := repo.Registry().ReadRegistry(r.ProjectID, r.ID)

		if err != nil {
			return nil, err
		}

		return &reg.TokenCache.TokenCache, nil
	}
}

func (r *Registry) setTokenCacheFunc(
	repo repository.Repository,
) ints.SetTokenCacheFunc {
	return func(token string, expiry time.Time) error {
		_, err := repo.Registry().UpdateRegistryTokenCache(
			&ints.RegTokenCache{
				TokenCache: ints.TokenCache{
					Token:  []byte(token),
					Expiry: expiry,
				},
				RegistryID: r.ID,
			},
		)

		return err
	}
}

// CreateRepository creates a repository for a registry, if needed
// (currently only required for ECR)
func (r *Registry) CreateRepository(
	repo repository.Repository,
	name string,
) error {
	// if aws, create repository
	if r.AWSIntegrationID != 0 {
		return r.createECRRepository(repo, name)
	}

	// otherwise, no-op
	return nil
}

func (r *Registry) createECRRepository(
	repo repository.Repository,
	name string,
) error {
	aws, err := repo.AWSIntegration().ReadAWSIntegration(
		r.ProjectID,
		r.AWSIntegrationID,
	)

	if err != nil {
		return err
	}

	sess, err := aws.GetSession()

	if err != nil {
		return err
	}

	svc := ecr.New(sess)

	// determine if repository already exists
	_, err = svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{&name},
	})

	// if the repository was not found, create it
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
		_, err = svc.CreateRepository(&ecr.CreateRepositoryInput{
			RepositoryName: &name,
		})

		return err
	} else if err != nil {
		return err
	}

	return nil
}

// ListImages lists the images for an image repository
func (r *Registry) ListImages(
	repoName string,
	repo repository.Repository,
	doAuth *oauth2.Config, // only required if using DOCR
) ([]*ptypes.Image, error) {
	// switch on the auth mechanism to get a token
	if r.AWSIntegrationID != 0 {
		return r.listECRImages(repoName, repo)
	}

	if r.GCPIntegrationID != 0 {
		return r.listGCRImages(repoName, repo)
	}

	if r.DOIntegrationID != 0 {
		return r.listDOCRImages(repoName, repo, doAuth)
	}

	if r.BasicIntegrationID != 0 {
		return r.listPrivateRegistryImages(repoName, repo)
	}

	return nil, fmt.Errorf("error listing images")
}

func (r *Registry) listECRImages(repoName string, repo repository.Repository) ([]*ptypes.Image, error) {
	aws, err := repo.AWSIntegration().ReadAWSIntegration(
		r.ProjectID,
		r.AWSIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	sess, err := aws.GetSession()

	if err != nil {
		return nil, err
	}

	svc := ecr.New(sess)

	resp, err := svc.ListImages(&ecr.ListImagesInput{
		RepositoryName: &repoName,
	})

	if err != nil {
		return nil, err
	}

	describeResp, err := svc.DescribeImages(&ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		ImageIds:       resp.ImageIds,
	})

	if err != nil {
		return nil, err
	}

	imageDetails := describeResp.ImageDetails

	nextToken := describeResp.NextToken

	for nextToken != nil {
		describeResp, err := svc.DescribeImages(&ecr.DescribeImagesInput{
			RepositoryName: &repoName,
			ImageIds:       resp.ImageIds,
		})

		if err != nil {
			return nil, err
		}

		nextToken = describeResp.NextToken
		imageDetails = append(imageDetails, describeResp.ImageDetails...)
	}

	res := make([]*ptypes.Image, 0)

	for _, img := range imageDetails {
		for _, tag := range img.ImageTags {
			res = append(res, &ptypes.Image{
				Digest:         *img.ImageDigest,
				Tag:            *tag,
				RepositoryName: repoName,
				PushedAt:       img.ImagePushedAt,
			})
		}
	}

	return res, nil
}

type gcrImageResp struct {
	Tags []string `json:"tags"`
}

func (r *Registry) listGCRImages(repoName string, repo repository.Repository) ([]*ptypes.Image, error) {
	gcp, err := repo.GCPIntegration().ReadGCPIntegration(
		r.ProjectID,
		r.GCPIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	// use JWT token to request catalog
	client := &http.Client{}

	parsedURL, err := url.Parse("https://" + r.URL)

	if err != nil {
		return nil, err
	}

	trimmedPath := strings.Trim(parsedURL.Path, "/")

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://%s/v2/%s/%s/tags/list", parsedURL.Host, trimmedPath, repoName),
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("_json_key", string(gcp.GCPKeyData))

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	gcrResp := gcrImageResp{}

	if err := json.NewDecoder(resp.Body).Decode(&gcrResp); err != nil {
		return nil, fmt.Errorf("Could not read GCR repositories: %v", err)
	}

	res := make([]*ptypes.Image, 0)

	for _, tag := range gcrResp.Tags {
		res = append(res, &ptypes.Image{
			RepositoryName: repoName,
			Tag:            tag,
		})
	}

	return res, nil
}

func (r *Registry) listDOCRImages(
	repoName string,
	repo repository.Repository,
	doAuth *oauth2.Config,
) ([]*ptypes.Image, error) {
	oauthInt, err := repo.OAuthIntegration().ReadOAuthIntegration(
		r.ProjectID,
		r.DOIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	tok, _, err := oauth.GetAccessToken(oauthInt.SharedOAuthModel, doAuth, oauth.MakeUpdateOAuthIntegrationTokenFunction(oauthInt, repo))

	if err != nil {
		return nil, err
	}

	client := godo.NewFromToken(tok)

	urlArr := strings.Split(r.URL, "/")

	if len(urlArr) != 2 {
		return nil, fmt.Errorf("invalid digital ocean registry url")
	}

	name := urlArr[1]

	tags, _, err := client.Registry.ListRepositoryTags(context.TODO(), name, repoName, &godo.ListOptions{})

	if err != nil {
		return nil, err
	}

	res := make([]*ptypes.Image, 0)

	for _, tag := range tags {
		res = append(res, &ptypes.Image{
			RepositoryName: repoName,
			Tag:            tag.Tag,
		})
	}

	return res, nil
}

func (r *Registry) listPrivateRegistryImages(repoName string, repo repository.Repository) ([]*ptypes.Image, error) {
	// handle dockerhub different, as it doesn't implement the docker registry http api
	if strings.Contains(r.URL, "docker.io") {
		return r.listDockerHubImages(repoName, repo)
	}

	basic, err := repo.BasicIntegration().ReadBasicIntegration(
		r.ProjectID,
		r.BasicIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	// Just use service account key to authenticate, since scopes may not be in place
	// for oauth. This also prevents us from making more requests.
	client := &http.Client{}

	// get the host and scheme to make the request
	parsedURL, err := url.Parse(r.URL)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s://%s/v2/%s/tags/list", parsedURL.Scheme, parsedURL.Host, repoName),
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(string(basic.Username), string(basic.Password))

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	gcrResp := gcrImageResp{}

	if err := json.NewDecoder(resp.Body).Decode(&gcrResp); err != nil {
		return nil, fmt.Errorf("Could not read private registry repositories: %v", err)
	}

	res := make([]*ptypes.Image, 0)

	for _, tag := range gcrResp.Tags {
		res = append(res, &ptypes.Image{
			RepositoryName: repoName,
			Tag:            tag,
		})
	}

	return res, nil
}

type dockerHubImageResult struct {
	Name string `json:"name"`
}

type dockerHubImageResp struct {
	Results []dockerHubImageResult `json:"results"`
}

type dockerHubLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type dockerHubLoginResp struct {
	Token string `json:"token"`
}

func (r *Registry) listDockerHubImages(repoName string, repo repository.Repository) ([]*ptypes.Image, error) {
	basic, err := repo.BasicIntegration().ReadBasicIntegration(
		r.ProjectID,
		r.BasicIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	// first, make a request for the access token

	data, err := json.Marshal(&dockerHubLoginReq{
		Username: string(basic.Username),
		Password: string(basic.Password),
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		"https://hub.docker.com/v2/users/login",
		strings.NewReader(string(data)),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	tokenObj := dockerHubLoginResp{}

	if err := json.NewDecoder(resp.Body).Decode(&tokenObj); err != nil {
		return nil, fmt.Errorf("Could not decode Dockerhub token from response: %v", err)
	}

	req, err = http.NewRequest(
		"GET",
		fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags", strings.Split(r.URL, "docker.io/")[1]),
		nil,
	)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenObj.Token))

	resp, err = client.Do(req)

	if err != nil {
		return nil, err
	}

	imageResp := dockerHubImageResp{}

	if err := json.NewDecoder(resp.Body).Decode(&imageResp); err != nil {
		return nil, fmt.Errorf("Could not read private registry repositories: %v", err)
	}

	res := make([]*ptypes.Image, 0)

	for _, result := range imageResp.Results {
		res = append(res, &ptypes.Image{
			RepositoryName: repoName,
			Tag:            result.Name,
		})
	}

	return res, nil
}

// GetDockerConfigJSON returns a dockerconfigjson file contents with "auths"
// populated.
func (r *Registry) GetDockerConfigJSON(
	repo repository.Repository,
	doAuth *oauth2.Config, // only required if using DOCR
) ([]byte, error) {
	var conf *configfile.ConfigFile
	var err error

	// switch on the auth mechanism to get a token
	if r.AWSIntegrationID != 0 {
		conf, err = r.getECRDockerConfigFile(repo)
	}

	if r.GCPIntegrationID != 0 {
		conf, err = r.getGCRDockerConfigFile(repo)
	}

	if r.DOIntegrationID != 0 {
		conf, err = r.getDOCRDockerConfigFile(repo, doAuth)
	}

	if r.BasicIntegrationID != 0 {
		conf, err = r.getPrivateRegistryDockerConfigFile(repo)
	}

	if err != nil {
		return nil, err
	}

	return json.Marshal(conf)
}

func (r *Registry) getECRDockerConfigFile(
	repo repository.Repository,
) (*configfile.ConfigFile, error) {
	aws, err := repo.AWSIntegration().ReadAWSIntegration(
		r.ProjectID,
		r.AWSIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	sess, err := aws.GetSession()

	if err != nil {
		return nil, err
	}

	ecrSvc := ecr.New(sess)

	output, err := ecrSvc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})

	if err != nil {
		return nil, err
	}

	token := *output.AuthorizationData[0].AuthorizationToken

	decodedToken, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(string(decodedToken), ":", 2)

	if len(parts) < 2 {
		return nil, err
	}

	key := r.URL

	if !strings.Contains(key, "http") {
		key = "https://" + key
	}

	return &configfile.ConfigFile{
		AuthConfigs: map[string]types.AuthConfig{
			key: {
				Username: parts[0],
				Password: parts[1],
				Auth:     token,
			},
		},
	}, nil
}

func (r *Registry) getGCRDockerConfigFile(
	repo repository.Repository,
) (*configfile.ConfigFile, error) {
	gcp, err := repo.GCPIntegration().ReadGCPIntegration(
		r.ProjectID,
		r.GCPIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	key := r.URL

	if !strings.Contains(key, "http") {
		key = "https://" + key
	}

	parsedURL, _ := url.Parse(key)

	return &configfile.ConfigFile{
		AuthConfigs: map[string]types.AuthConfig{
			parsedURL.Host: {
				Username: "_json_key",
				Password: string(gcp.GCPKeyData),
				Auth:     generateAuthToken("_json_key", string(gcp.GCPKeyData)),
			},
		},
	}, nil
}

func (r *Registry) getDOCRDockerConfigFile(
	repo repository.Repository,
	doAuth *oauth2.Config,
) (*configfile.ConfigFile, error) {
	oauthInt, err := repo.OAuthIntegration().ReadOAuthIntegration(
		r.ProjectID,
		r.DOIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	tok, _, err := oauth.GetAccessToken(oauthInt.SharedOAuthModel, doAuth, oauth.MakeUpdateOAuthIntegrationTokenFunction(oauthInt, repo))

	if err != nil {
		return nil, err
	}

	key := r.URL

	if !strings.Contains(key, "http") {
		key = "https://" + key
	}

	parsedURL, _ := url.Parse(key)

	return &configfile.ConfigFile{
		AuthConfigs: map[string]types.AuthConfig{
			parsedURL.Host: {
				Username: tok,
				Password: tok,
				Auth:     generateAuthToken(tok, tok),
			},
		},
	}, nil
}

func (r *Registry) getPrivateRegistryDockerConfigFile(
	repo repository.Repository,
) (*configfile.ConfigFile, error) {
	basic, err := repo.BasicIntegration().ReadBasicIntegration(
		r.ProjectID,
		r.BasicIntegrationID,
	)

	if err != nil {
		return nil, err
	}

	key := r.URL

	if !strings.Contains(key, "http") {
		key = "https://" + key
	}

	parsedURL, _ := url.Parse(key)

	authConfigKey := parsedURL.Host

	if strings.Contains(r.URL, "index.docker.io") {
		authConfigKey = "https://index.docker.io/v1/"
	}

	return &configfile.ConfigFile{
		AuthConfigs: map[string]types.AuthConfig{
			authConfigKey: {
				Username: string(basic.Username),
				Password: string(basic.Password),
				Auth:     generateAuthToken(string(basic.Username), string(basic.Password)),
			},
		},
	}, nil
}

func generateAuthToken(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
