package scope

import (
	"context"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	EnvCoxAPIKey       = "COX_API_KEY"
	EnvCoxService      = "COX_SERVICE"
	EnvCoxEnvironment  = "COX_ENVIRONMENT"
	EnvCoxOrganization = "COX_ORGANIZATION"
)

type Credentials struct {
	CoxAPIKey       string
	CoxEnvironment  string
	CoxService      string
	CoxOrganization string
	CoxAPIBaseURL	string
}

func (c *Credentials) IsEmpty() bool {
	return c == nil || (len(c.CoxAPIKey) == 0 && len(c.CoxEnvironment) == 0 && len(c.CoxService) == 0)
}

func GetCredentials(client client.Client, namespace string, name string) (*Credentials, error) {
	tokenSecret := &corev1.Secret{}
	coxSecretName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := client.Get(context.Background(), coxSecretName, tokenSecret); err != nil {
		return nil, errors.Errorf("error getting referenced token secret/%s: %s", coxSecretName, err)
	}

	CoxAPIKey, keyExists := tokenSecret.Data[coxedge.CoxAPIKey]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxAPIKey, coxSecretName)
	}

	coxEnvironment, keyExists := tokenSecret.Data[coxedge.CoxEnvironment]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxEnvironment, coxSecretName)
	}

	coxService, keyExists := tokenSecret.Data[coxedge.CoxService]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxService, coxSecretName)
	}

	coxOrganization, _ := tokenSecret.Data[coxedge.CoxOrganization]

	coxAPIBaseURL, _ := tokenSecret.Data[coxedge.CoxAPIBaseURL]

	return &Credentials{
		CoxAPIKey:       string(CoxAPIKey),
		CoxEnvironment:  string(coxEnvironment),
		CoxService:      string(coxService),
		CoxOrganization: string(coxOrganization),
		CoxAPIBaseURL: string(coxAPIBaseURL),
	}, nil
}

func ParseFromEnv() (*Credentials, error) {
	CoxAPIKey, keyExists := os.LookupEnv(EnvCoxAPIKey)
	if !keyExists {
		return nil, errors.Errorf("key '%s' does not exist in env", EnvCoxAPIKey)
	}

	coxEnvironment, keyExists := os.LookupEnv(EnvCoxEnvironment)
	if !keyExists {
		return nil, errors.Errorf("key '%s' does not exist in env", EnvCoxEnvironment)
	}

	coxService, keyExists := os.LookupEnv(EnvCoxService)
	if !keyExists {
		return nil, errors.Errorf("key '%s' does not exist in env", EnvCoxService)
	}

	coxOrganization, keyExists := os.LookupEnv(EnvCoxOrganization)
	if !keyExists {
		coxOrganization = ""
	}

	return &Credentials{
		CoxAPIKey:       CoxAPIKey,
		CoxEnvironment:  coxEnvironment,
		CoxService:      coxService,
		CoxOrganization: coxOrganization,
	}, nil
}
