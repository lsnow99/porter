package kubernetes_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/porter-dev/porter/internal/kubernetes"
	"github.com/porter-dev/porter/internal/models"
	"k8s.io/client-go/tools/clientcmd"
)

type kubeConfigTest struct {
	msg             string
	raw             []byte
	allowedContexts []string
	expected        []models.Context
}

type kubeConfigTestValidateError struct {
	msg             string
	raw             []byte
	allowedContexts []string
	contextName     string
	errorContains   string // a string that the error message should contain
}

var ValidateErrorTests = []kubeConfigTestValidateError{
	kubeConfigTestValidateError{
		msg:             "No configuration",
		raw:             []byte(""),
		allowedContexts: []string{},
		contextName:     "",
		errorContains:   "invalid configuration: no configuration has been provided",
	},
	kubeConfigTestValidateError{
		msg:             "Context name does not exist",
		raw:             []byte(noContexts),
		allowedContexts: []string{"porter-test-1"},
		contextName:     "context-test",
		errorContains:   "invalid configuration: context was not found for specified context: context-test",
	},
	kubeConfigTestValidateError{
		msg:             "Cluster to join does not exist",
		raw:             []byte(noClusters),
		allowedContexts: []string{"porter-test-1"},
		contextName:     "context-test",
		errorContains:   "invalid configuration: context was not found for specified context: context-test",
	},
	kubeConfigTestValidateError{
		msg:             "User to join does not exist",
		raw:             []byte(noUsers),
		allowedContexts: []string{"porter-test-1"},
		contextName:     "context-test",
		errorContains:   "invalid configuration: context was not found for specified context: context-test",
	},
}

func TestValidateErrors(t *testing.T) {
	for _, c := range ValidateErrorTests {

		_, err := kubernetes.GetRestrictedClientConfigFromBytes(c.raw, c.contextName, c.allowedContexts)

		if err == nil {
			t.Fatalf("Testing %s did not return an error\n", c.msg)
		}

		if !strings.Contains(err.Error(), c.errorContains) {
			t.Errorf("Testing %s -- Error was:\n \"%s\" \n It did not contain string \"%s\"\n", c.msg, err.Error(), c.errorContains)
		}
	}
}

var BasicContextAllowedTests = []kubeConfigTest{
	kubeConfigTest{
		msg:             "basic test",
		raw:             []byte(basic),
		allowedContexts: []string{"context-test"},
		expected: []models.Context{
			models.Context{
				Name:     "context-test",
				Server:   "https://localhost",
				Cluster:  "cluster-test",
				User:     "test-admin",
				Selected: true,
			},
		},
	},
}

func TestBasicAllowed(t *testing.T) {
	for _, c := range BasicContextAllowedTests {
		res, err := kubernetes.GetContextsFromBytes(c.raw, c.allowedContexts)

		if err != nil {
			t.Fatalf("Testing %s returned an error: %v\n", c.msg, err.Error())
		}

		isEqual := reflect.DeepEqual(c.expected, res)

		if !isEqual {
			t.Errorf("Testing: %s, Expected: %v, Got: %v\n", c.msg, c.expected, res)
		}
	}
}

var BasicContextAllTests = []kubeConfigTest{
	kubeConfigTest{
		msg:             "basic test",
		raw:             []byte(basic),
		allowedContexts: []string{},
		expected: []models.Context{
			models.Context{
				Name:     "context-test",
				Server:   "https://localhost",
				Cluster:  "cluster-test",
				User:     "test-admin",
				Selected: false,
			},
		},
	},
}

func TestBasicAll(t *testing.T) {
	for _, c := range BasicContextAllTests {
		res, err := kubernetes.GetContextsFromBytes(c.raw, c.allowedContexts)

		if err != nil {
			t.Fatalf("Testing %s returned an error: %v\n", c.msg, err.Error())
		}

		isEqual := reflect.DeepEqual(c.expected, res)

		if !isEqual {
			t.Errorf("Testing: %s, Expected: %v, Got: %v\n", c.msg, c.expected, res)
		}
	}
}

func TestGetRestrictedClientConfig(t *testing.T) {
	contexts := []string{"context-test"}
	contextName := "context-test"

	clientConf, err := kubernetes.GetRestrictedClientConfigFromBytes([]byte(basic), contextName, contexts)

	if err != nil {
		t.Fatalf("Fatal error: %s\n", err.Error())
	}

	rawConf, err := clientConf.RawConfig()

	if err != nil {
		t.Fatalf("Fatal error: %s\n", err.Error())
	}

	if cluster, clusterFound := rawConf.Clusters["cluster-test"]; !clusterFound || cluster.Server != "https://localhost" {
		t.Errorf("invalid cluster returned")
	}

	if _, contextFound := rawConf.Contexts["context-test"]; !contextFound {
		t.Errorf("invalid context returned")
	}

	if _, authInfoFound := rawConf.AuthInfos["test-admin"]; !authInfoFound {
		t.Errorf("invalid auth info returned")
	}
}

type saCandidatesTest struct {
	name     string
	raw      []byte
	expected []*models.ServiceAccountCandidate
}

var SACandidatesTests = []saCandidatesTest{
	saCandidatesTest{
		name: "test without cluster ca data",
		raw:  []byte(ClusterCAWithoutData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-cluster-ca-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.X509,
				Kubeconfig:      []byte(ClusterCAWithoutData),
			},
		},
	},
	saCandidatesTest{
		name: "x509 test with cert and key data",
		raw:  []byte(x509WithData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.X509,
				Kubeconfig:      []byte(x509WithData),
			},
		},
	},
	saCandidatesTest{
		name: "x509 test without cert data",
		raw:  []byte(x509WithoutCertData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-client-cert-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.X509,
				Kubeconfig:      []byte(x509WithoutCertData),
			},
		},
	},
	saCandidatesTest{
		name: "x509 test without key data",
		raw:  []byte(x509WithoutKeyData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-client-key-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.X509,
				Kubeconfig:      []byte(x509WithoutKeyData),
			},
		},
	},
	saCandidatesTest{
		name: "x509 test without cert and key data",
		raw:  []byte(x509WithoutCertAndKeyData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-client-cert-data,upload-client-key-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.X509,
				Kubeconfig:      []byte(x509WithoutCertAndKeyData),
			},
		},
	},
	saCandidatesTest{
		name: "bearer token test with data",
		raw:  []byte(BearerTokenWithData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.Bearer,
				Kubeconfig:      []byte(BearerTokenWithData),
			},
		},
	},
	saCandidatesTest{
		name: "bearer token test without data",
		raw:  []byte(BearerTokenWithoutData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-token-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.Bearer,
				Kubeconfig:      []byte(BearerTokenWithoutData),
			},
		},
	},
	saCandidatesTest{
		name: "gcp test",
		raw:  []byte(GCPPlugin),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-gcp-key-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.GCP,
				Kubeconfig:      []byte(GCPPlugin),
			},
		},
	},
	saCandidatesTest{
		name: "aws iam authenticator test",
		raw:  []byte(AWSIamAuthenticatorExec),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-aws-key-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.AWS,
				Kubeconfig:      []byte(AWSIamAuthenticatorExec),
			},
		},
	},
	saCandidatesTest{
		name: "aws eks get-token test",
		raw:  []byte(AWSEKSGetTokenExec),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-aws-key-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.AWS,
				Kubeconfig:      []byte(AWSEKSGetTokenExec),
			},
		},
	},
	saCandidatesTest{
		name: "oidc without ca data",
		raw:  []byte(OIDCAuthWithoutData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "upload-oidc-idp-issuer-ca-data",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.OIDC,
				Kubeconfig:      []byte(OIDCAuthWithoutData),
			},
		},
	},
	saCandidatesTest{
		name: "oidc with ca data",
		raw:  []byte(OIDCAuthWithData),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.OIDC,
				Kubeconfig:      []byte(OIDCAuthWithData),
			},
		},
	},
	saCandidatesTest{
		name: "basic auth test",
		raw:  []byte(BasicAuth),
		expected: []*models.ServiceAccountCandidate{
			&models.ServiceAccountCandidate{
				ActionNames:     "",
				Kind:            "connector",
				ClusterName:     "cluster-test",
				ClusterEndpoint: "https://localhost",
				AuthMechanism:   models.Basic,
				Kubeconfig:      []byte(BasicAuth),
			},
		},
	},
}

func TestGetServiceAccountCandidates(t *testing.T) {
	for _, c := range SACandidatesTests {
		result, err := kubernetes.GetServiceAccountCandidates(c.raw)

		if err != nil {
			t.Fatalf("error occurred %v\n", err)
		}

		// make result into a map so it's easier to compare
		resMap := make(map[string]*models.ServiceAccountCandidate)

		for _, res := range result {
			resMap[res.Kind+"-"+res.ClusterEndpoint+"-"+res.AuthMechanism] = res
		}

		for _, exp := range c.expected {
			res, ok := resMap[exp.Kind+"-"+exp.ClusterEndpoint+"-"+exp.AuthMechanism]

			if !ok {
				t.Fatalf("%s failed: no matching result for %s\n", c.name,
					exp.Kind+"-"+exp.ClusterEndpoint+"-"+exp.AuthMechanism)
			}

			// compare basic string fields
			if exp.AuthMechanism != res.AuthMechanism {
				t.Errorf("%s failed on auth mechanism: expected %s, got %s\n",
					c.name, exp.AuthMechanism, res.AuthMechanism)
			}

			if exp.ClusterName != res.ClusterName {
				t.Errorf("%s failed on cluster name: expected %s, got %s\n",
					c.name, exp.ClusterName, res.ClusterName)
			}

			if exp.ClusterEndpoint != res.ClusterEndpoint {
				t.Errorf("%s failed on cluster endpoint: expected %s, got %s\n",
					c.name, exp.ClusterEndpoint, res.ClusterEndpoint)
			}

			// compare action names by splitting the arrays (actions may be out of order)
			resActions := strings.Split(res.ActionNames, ",")
			expActions := strings.Split(exp.ActionNames, ",")

			if len(resActions) != len(expActions) {
				t.Errorf("%s failed on action names: expected length %d, got length %d\n",
					c.name, len(expActions), len(resActions))
			} else {
				for _, actionName := range expActions {
					if !strings.Contains(res.ActionNames, actionName) {
						t.Errorf("%s failed on action names: expected res to contain %s, got %s\n",
							c.name, actionName, res.ActionNames)
					}
				}
			}

			// compare kubeconfig by transforming into a client config
			resConfig, _ := clientcmd.NewClientConfigFromBytes(res.Kubeconfig)
			expConfig, err := clientcmd.NewClientConfigFromBytes(exp.Kubeconfig)

			if err != nil {
				t.Fatalf("config from bytes, error occurred %v\n", err)
			}

			resRawConf, _ := resConfig.RawConfig()
			expRawConf, err := expConfig.RawConfig()

			if err != nil {
				t.Fatalf("raw config conversion, error occurred %v\n", err)
			}

			if !reflect.DeepEqual(resRawConf, expRawConf) {
				t.Errorf("%s failed: expected %v, got %v\n", c.name, expRawConf, resRawConf)
			}
		}
	}
}

const noContexts string = `
apiVersion: v1
kind: Config
preferences: {}
clusters:
- cluster:
    server: https://localhost
  name: porter-test-1
current-context: context-test
users:
- name: test-admin
  user:
`

const noClusters string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
contexts:
- context:
    cluster: porter-test-1
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
`

const noUsers string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: default
clusters:
- cluster:
    server: https://localhost
  name: porter-test-1
contexts:
- context:
    cluster: porter-test-1
    user: test-admin
  name: context-test
`

const noContextClusters string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: default
clusters:
- cluster:
    server: https://localhost
  name: porter-test-1
contexts:
- context:
    # cluster: porter-test-1
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
`

const noContextUsers string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: default
clusters:
- cluster:
    server: https://localhost
  name: porter-test-1
contexts:
- context:
    cluster: porter-test-1
    # user: test-admin
  name: context-test
users:
- name: test-admin
  user:
`

const basic string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
  - name: test-admin
`

const ClusterCAWithoutData string = `
apiVersion: v1
kind: Config
clusters:
- name: cluster-test
  cluster:
    server: https://localhost
    certificate-authority: /fake/path/to/ca.pem
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJ=
    client-key-data: LS0tLS1CRUdJTiBDRVJ=
current-context: context-test
`

const x509WithData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJ=
    client-key-data: LS0tLS1CRUdJTiBDRVJ=
`

const x509WithoutCertData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    client-certificate: /fake/path/to/cert.pem
    client-key-data: LS0tLS1CRUdJTiBDRVJ=
`

const x509WithoutKeyData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJ=
    client-key: /fake/path/to/key.pem
`

const x509WithoutCertAndKeyData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    client-certificate: /fake/path/to/cert.pem
    client-key: /fake/path/to/key.pem
`

const BearerTokenWithData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    token: LS0tLS1CRUdJTiBDRVJ=
`

const BearerTokenWithoutData string = `
apiVersion: v1
kind: Config
preferences: {}
current-context: context-test
clusters:
- cluster:
    server: https://localhost
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
users:
- name: test-admin
  user:
    tokenFile: /path/to/token/file.txt
`
const GCPPlugin string = `
apiVersion: v1
kind: Config
clusters:
- name: cluster-test
  cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
users:
- name: test-admin
  user:
    auth-provider:
      name: gcp
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
`

const AWSIamAuthenticatorExec = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
kind: Config
preferences: {}
users:
- name: test-admin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: aws-iam-authenticator
      args:
        - "token"
        - "-i"
        - "cluster-test"
`

const AWSEKSGetTokenExec = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
kind: Config
preferences: {}
users:
- name: test-admin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      command: aws
      args:
        - "eks"
        - "get-token"
        - "--cluster-name"
        - "cluster-test"
`

const OIDCAuthWithoutData = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
kind: Config
preferences: {}
users:
- name: test-admin
  user:
    auth-provider:
      config:
        client-id: porter-api
        id-token: token
        idp-issuer-url: https://localhost
        idp-certificate-authority: /fake/path/to/ca.pem
      name: oidc
`

const OIDCAuthWithData = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
kind: Config
preferences: {}
users:
- name: test-admin
  user:
    auth-provider:
      config:
        client-id: porter-api
        id-token: token
        idp-issuer-url: https://localhost
        idp-certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
      name: oidc
`

const BasicAuth = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJ=
  name: cluster-test
contexts:
- context:
    cluster: cluster-test
    user: test-admin
  name: context-test
current-context: context-test
kind: Config
preferences: {}
users:
- name: test-admin
  user:
    username: admin
    password: changeme
`
