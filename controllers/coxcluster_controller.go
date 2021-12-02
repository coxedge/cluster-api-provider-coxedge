/*
Copyright 2021.

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

package controllers

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	coxv1 "github.com/platform9/cluster-api-provider-cox/api/v1beta1"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge/scope"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	K8sApiPort = 6443
)

// CoxClusterReconciler reconciles a CoxCluster object
type CoxClusterReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	CoxClient *coxedge.Client
}

//+kubebuilder:rbac:groups=cluster.capi.pf9.io,resources=coxclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.capi.pf9.io,resources=coxclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CoxCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CoxClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	var coxCluster coxv1.CoxCluster
	if err := r.Get(ctx, req.NamespacedName, &coxCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, coxCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("OwnerCluster is not set yet. Requeuing...")
		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, &coxCluster.ObjectMeta) {
		log.Info("CoxCluster or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	// Create the cluster scope
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Logger:     log,
		Client:     r.Client,
		Cluster:    cluster,
		CoxCluster: &coxCluster,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted clusters
	if !cluster.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(&coxCluster, coxv1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}
	return r.reconcileNormal(&coxCluster, clusterScope)
}

func (r *CoxClusterReconciler) reconcileNormal(coxCluster *coxv1.CoxCluster, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	controllerutil.AddFinalizer(coxCluster, coxv1.ClusterFinalizer)
	workloads, _, err := r.CoxClient.GetWorkloads()
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, workload := range workloads.Data {
		if workload.Name == coxCluster.Name {
			// get instance
			clusterScope.CoxCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
				Host: workload.AnycastIPAddress,
				Port: K8sApiPort,
			}

			clusterScope.CoxCluster.Status.Ready = true
			break
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CoxClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coxv1.CoxCluster{}).
		Complete(r)
}
