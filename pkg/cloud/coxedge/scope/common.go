package scope

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Credentials struct {
	CoxApiKey      string
	CoxEnvironment string
	CoxService     string
}

func GetCredentials(client client.Client, namespace string, name string) (*Credentials, error) {
	var tokenSecret *corev1.Secret

	coxSecretName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := client.Get(context.Background(), coxSecretName, tokenSecret); err != nil {
		return nil, errors.Errorf("error getting referenced token secret/%s: %s", coxSecretName, err)
	}

	coxApiKey, keyExists := tokenSecret.Data[coxedge.CoxApiKey]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxApiKey, coxSecretName)
	}

	coxEnvironment, keyExists := tokenSecret.Data[coxedge.CoxEnvironment]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxEnvironment, coxSecretName)
	}

	coxService, keyExists := tokenSecret.Data[coxedge.CoxService]
	if !keyExists {
		return nil, errors.Errorf("error key %s does not exist in secret/%s", coxedge.CoxService, coxSecretName)
	}

	return &Credentials{
		CoxApiKey:      string(coxApiKey),
		CoxEnvironment: string(coxEnvironment),
		CoxService:     string(coxService)}, nil
}
