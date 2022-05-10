/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	coxv1 "github.com/coxedge/cluster-api-provider-cox/api/v1beta1"
	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Client             client.Client
	Logger             logr.Logger
	Cluster            *clusterv1beta1.Cluster
	Machine            *clusterv1beta1.Machine
	CoxCluster         *coxv1.CoxCluster
	CoxMachine         *coxv1.CoxMachine
	DefaultCredentials *Credentials
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration
// both CoxClusterReconciler and CoxMachineReconciler.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.CoxCluster == nil {
		return nil, errors.New("CoxCluster  is required when creating a MachineScope")
	}
	if params.CoxMachine == nil {
		return nil, errors.New("CoxMachine is required when creating a MachineScope")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.CoxMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	var creds *Credentials
	if params.CoxCluster.Spec.Credentials != nil && len(params.CoxCluster.Spec.Credentials.Name) > 0 {
		creds, err = GetCredentials(params.Client, params.CoxCluster.Namespace, params.CoxCluster.Spec.Credentials.Name)
		if err != nil {
			return nil, err
		}
	} else if params.DefaultCredentials != nil {
		creds = params.DefaultCredentials
	} else {
		return nil, errors.New("no default or cluster-specific credentials provided")
	}

	coxClient, err := coxedge.NewClient(creds.CoxService, creds.CoxEnvironment, creds.CoxAPIKey, nil)
	if err != nil {
		return nil, errors.Errorf("error while trying to create instance of coxedge client %s", err.Error())
	}

	return &MachineScope{
		client:      params.Client,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		CoxMachine:  params.CoxMachine,
		CoxCluster:  params.CoxCluster,
		CoxClient:   coxClient,
		Logger:      params.Logger,
		patchHelper: helper,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster    *clusterv1beta1.Cluster
	Machine    *clusterv1beta1.Machine
	CoxCluster *coxv1.CoxCluster
	CoxMachine *coxv1.CoxMachine
	CoxClient  *coxedge.Client
}

// Close the MachineScope by updating the machine spec, machine status.
func (m *MachineScope) Close() error {
	return m.patchHelper.Patch(context.TODO(), m.CoxMachine)
}

// Name returns the CoxMachine name
func (m *MachineScope) Name() string {
	return m.CoxMachine.Name
}

// Namespace returns the CoxMachine namespace
func (m *MachineScope) Namespace() string {
	return m.CoxMachine.Namespace
}

// GetProviderID returns the DOMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	return m.CoxMachine.Spec.ProviderID
}

// SetProviderID sets the DOMachine providerID in spec from device id.
func (m *MachineScope) SetProviderID(deviceID string) {
	pid := fmt.Sprintf("coxedge://%s", deviceID)

	m.CoxMachine.Spec.ProviderID = pid
}

// GetInstanceID returns the DOMachine droplet instance id by parsing Spec.ProviderID.
func (m *MachineScope) GetInstanceID() string {
	return strings.Replace(m.CoxMachine.Spec.ProviderID, "coxedge://", "", -1)
}

// SetErrorMessage sets the CoxMachine status error message.
func (m *MachineScope) SetErrorMessage(v error) {
	m.CoxMachine.Status.ErrorMessage = pointer.StringPtr(v.Error())
}

// SetAddresses sets the address status.
func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.CoxMachine.Status.Addresses = addrs
}

// GetRawBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetRawBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for CoxMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return string(value), nil
}
