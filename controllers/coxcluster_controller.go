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
	"fmt"
	"reflect"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/pkg/errors"
	coxv1 "github.com/platform9/cluster-api-provider-cox/api/v1beta1"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge/scope"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	defaultKubeApiserverPort = 6443
	defaultBackend           = "example.com:80"
)

// CoxClusterReconciler reconciles a CoxCluster object
type CoxClusterReconciler struct {
	client.Client
	DefaultCredentials *scope.Credentials
	Scheme             *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=coxclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=coxclusters/status,verbs=get;update;patch

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
		Logger:             log,
		Client:             r.Client,
		Cluster:            cluster,
		CoxCluster:         &coxCluster,
		DefaultCredentials: r.DefaultCredentials,
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
		return r.reconcileDelete(ctx, clusterScope)
	}
	return r.reconcileNormal(ctx, clusterScope)
}

func (r *CoxClusterReconciler) reconcileNormal(ctx context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	coxCluster := clusterScope.CoxCluster
	controllerutil.AddFinalizer(coxCluster, coxv1.ClusterFinalizer)

	// Hacky way to retrieve the control plane endpoints from the machines
	var apiserverAddresses []string
	coxMachines := &coxv1.CoxMachineList{}
	err := r.Client.List(ctx, coxMachines)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, coxMachine := range coxMachines.Items {
		if coxMachine.Labels[clusterv1.ClusterLabelName] != clusterScope.Name() {
			continue
		}

		if _, ok := coxMachine.Labels[clusterv1.MachineControlPlaneLabelName]; !ok {
			continue
		}

		for _, addr := range coxMachine.Status.Addresses {
			if addr.Type != corev1.NodeExternalIP {
				continue
			}
			apiserverAddresses = append(apiserverAddresses, fmt.Sprintf("%s:%d", addr.Address, defaultKubeApiserverPort))
			break
		}
	}
	if len(apiserverAddresses) == 0 {
		// Needs to be set to some value
		apiserverAddresses = []string{defaultBackend}
	}

	// Ensure that the loadBalancer is created
	lbClient := coxedge.NewLoadBalancerHelper(clusterScope.CoxClient)
	loadBalancerSpec := coxedge.LoadBalancerSpec{
		Name:     genClusterLoadBalancerName(clusterScope.Name()),
		Port:     fmt.Sprintf("%d", defaultKubeApiserverPort),
		Backends: apiserverAddresses,
	}
	existingLoadBalancer, err := lbClient.GetLoadBalancer(ctx, loadBalancerSpec.Name)
	if err != nil {
		if err != coxedge.ErrWorkloadNotFound {
			return ctrl.Result{}, err
		}
		err = lbClient.CreateLoadBalancer(ctx, &loadBalancerSpec)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Created LoadBalancer deployment", "spec", loadBalancerSpec)
		return ctrl.Result{Requeue: true}, nil
	} else {
		// The name can be normalized to
		loadBalancerSpec.Name = existingLoadBalancer.Spec.Name
		if !reflect.DeepEqual(existingLoadBalancer.Spec, loadBalancerSpec) {
			existingLoadBalancer.Status = coxedge.LoadBalancerStatus{}
			err = lbClient.UpdateLoadBalancer(ctx, &loadBalancerSpec)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Updated LoadBalancer deployment", "old", existingLoadBalancer.Spec, "new", loadBalancerSpec)
		}
	}

	if existingLoadBalancer != nil && len(existingLoadBalancer.Status.PublicIP) == 0 {
		log.Info("LoadBalancer is not ready yet.")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// Hack: requeue as long as the load balancer does not yet have an appropriate backend.
	if apiserverAddresses[0] == defaultBackend {
		log.Info("LoadBalancer does not yet have a valid apiserver to use as backend.")
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// Set the controlPlaneRef
	port, err := strconv.Atoi(existingLoadBalancer.Spec.Port)
	if err != nil {
		return ctrl.Result{}, err
	}
	clusterScope.CoxCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: existingLoadBalancer.Status.PublicIP,
		Port: int32(port),
	}
	clusterScope.CoxCluster.Status.Ready = true

	log.Info("Cluster reconciled.")
	return ctrl.Result{}, nil
}

func (r *CoxClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	loadBalancerName := genClusterLoadBalancerName(clusterScope.Name())
	lbClient := coxedge.NewLoadBalancerHelper(clusterScope.CoxClient)
	err := lbClient.DeleteLoadBalancer(ctx, loadBalancerName)
	if err != nil {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(clusterScope.CoxCluster, coxv1.ClusterFinalizer)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CoxClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&coxv1.CoxCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	// Add a watch on clusterv1.Cluster object for unpause notifications.
	if err = c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(coxv1.GroupVersion.WithKind("CoxCluster"))),
		predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

func genClusterLoadBalancerName(clusterName string) string {
	return fmt.Sprintf("lb-%s", clusterName)
}
