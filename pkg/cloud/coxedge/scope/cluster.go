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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	coxv1 "github.com/platform9/cluster-api-provider-cox/api/v1beta1"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"

	"k8s.io/klog/v2/klogr"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1beta1.Cluster
	CoxCluster *coxv1.CoxCluster
	CoxClient  *coxedge.Client
}

// NewClusterScope creates a new ClusterScope from the supplied parameters.
// This is meant to be called for each reconcile iteration only on ClusterReconciler.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}
	if params.CoxCluster == nil {
		return nil, errors.New("CoxCluster is required when creating a ClusterScope")
	}
	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.CoxCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	creds, err := GetCredentials(params.Client, params.CoxCluster.Namespace, params.CoxCluster.Spec.Credentials.Name)
	if err != nil {
		return nil, err
	}

	coxClient, err := coxedge.NewClient(creds.CoxService, creds.CoxEnvironment, creds.CoxApiKey, nil)
	if err != nil {
		return nil, errors.Errorf("error while trying to create instance of coxedge client %s", err.Error())
	}

	return &ClusterScope{
		Logger:      params.Logger,
		client:      params.Client,
		Cluster:     params.Cluster,
		CoxCluster:  params.CoxCluster,
		CoxClient:   coxClient,
		patchHelper: helper,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	Cluster    *clusterv1beta1.Cluster
	CoxCluster *coxv1.CoxCluster
	CoxClient  *coxedge.Client
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.CoxCluster)
}

// Name returns the cluster name.
func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

// Namespace returns the cluster namespace.
func (s *ClusterScope) Namespace() string {
	return s.Cluster.GetNamespace()
}

// SetReady sets the CoxCluster Ready Status
func (s *ClusterScope) SetReady() {
	s.CoxCluster.Status.Ready = true
}
