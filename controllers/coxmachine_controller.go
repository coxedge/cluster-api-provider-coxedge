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
	"encoding/json"
	"fmt"

	"sigs.k8s.io/cluster-api/util/predicates"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	coxv1 "github.com/platform9/cluster-api-provider-cox/api/v1beta1"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/platform9/cluster-api-provider-cox/pkg/cloud/coxedge/scope"
)

// CoxMachineReconciler reconciles a CoxMachine object
type CoxMachineReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	CoxClient *coxedge.Client
}

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=coxmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=coxmachines/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CoxMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *CoxMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {

	logger := ctrl.LoggerFrom(ctx)

	coxMachine := &coxv1.CoxMachine{}
	if err := r.Get(ctx, req.NamespacedName, coxMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, coxMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}
	logger = logger.WithValues("machine", machine.Name)
	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:     r.Client,
		Logger:     logger,
		Cluster:    cluster,
		CoxMachine: coxMachine,
		Machine:    machine,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any CoxMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !coxMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, logger)
	}

	coxCluster := &coxv1.CoxCluster{}
	coxClusterNamespacedName := client.ObjectKey{
		Namespace: coxMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Get(ctx, coxClusterNamespacedName, coxCluster); err != nil {
		logger.Info("CoxCluster is not available yet")
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, machineScope, logger)
}

func (r *CoxMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Reconciling CoxMachine")
	coxMachine := machineScope.CoxMachine
	controllerutil.AddFinalizer(coxMachine, coxv1.MachineFinalizer)

	if machineScope.GetProviderID() == "" {
		workload, err := r.CoxClient.GetWorkloadByName(machineScope.CoxMachine.Name)
		if err != nil && err != coxedge.ErrWorkloadNotFound {
			return ctrl.Result{}, err
		}
		if workload != nil {
			machineScope.SetProviderID(workload.ID)
		}
	}
	if coxMachine.Status.ErrorMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, fmt.Errorf(*coxMachine.Status.ErrorMessage)
	}

	var (
		workload   *coxedge.Workload
		err        error
		workloadID string
	)
	providerID := machineScope.GetInstanceID()
	if providerID != "" {
		workload, _, err = r.CoxClient.GetWorkload(providerID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	bootstrapData, err := machineScope.GetRawBootstrapData()
	if err != nil {
		return ctrl.Result{}, err
	}

	if workload == nil {
		// create workload
		data := &coxedge.CreateWorkloadRequest{
			Name:                machineScope.Name(),
			Type:                machineScope.CoxMachine.Spec.Type,
			Image:               machineScope.CoxMachine.Spec.Image,
			AddAnyCastIPAddress: machineScope.CoxMachine.Spec.AddAnyCastIPAddress,
			Ports:               machineScope.CoxMachine.Spec.Ports,
			FirstBootSSHKey:     machineScope.CoxMachine.Spec.FirstBootSSHKey,
			Deployments:         machineScope.CoxMachine.Spec.Deployments,
			Specs:               machineScope.CoxMachine.Spec.Specs,
			UserData:            bootstrapData,
		}

		resp, errResp, err := r.CoxClient.CreateWorkload(data)

		if err != nil {
			jsn, _ := json.Marshal(errResp)
			return ctrl.Result{}, fmt.Errorf("error occured while creating workload: %v - response: %v", err, string(jsn))
		}

		logger.Info("Waiting for workload to be provisioned")
		workloadID, err = r.CoxClient.WaitForWorkload(resp.TaskId)
		if err != nil {
			machineScope.SetErrorMessage(err)
			return ctrl.Result{}, err
		}
		machineScope.CoxMachine.Status.Ready = true

		machineScope.SetProviderID(workloadID)
	}

	return ctrl.Result{}, nil
}

func (r *CoxMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Deleting machine")
	// check if workload exists
	providerID := machineScope.GetInstanceID()
	wl, resp, err := r.CoxClient.GetWorkload(providerID)
	if err != nil {
		if resp.StatusCode == 404 {
			logger.Info("unable to find CoxMachine", "errors", resp.Errors)
		} else {
			return ctrl.Result{}, err
		}
	}

	if wl != nil {
		if providerID != "" {
			_, _, err := r.CoxClient.DeleteWorkload(providerID)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to delete the machine: %v", err)
			}
		}
	} else {
		logger.Info("unable to find CoxMachine")
	}

	controllerutil.RemoveFinalizer(machineScope.CoxMachine, coxv1.MachineFinalizer)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CoxMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&coxv1.CoxMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(coxv1.GroupVersion.WithKind("CoxMachine"))),
		).
		Watches(
			&source.Kind{Type: &coxv1.CoxCluster{}},
			handler.EnqueueRequestsFromMapFunc(r.CoxClusterToCoxMachines(ctx)),
		).
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &coxv1.CoxMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to DOMachines")
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

func (r *CoxMachineReconciler) CoxClusterToCoxMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(o client.Object) []ctrl.Request {
		var result []ctrl.Request

		c, ok := o.(*coxv1.CoxCluster)
		if !ok {
			log.Error(errors.Errorf("expected a CoxCluster but got a %T", o), "failed to get CoxMachine for CoxCluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines")
			return nil
		}
		for _, m := range machineList.Items {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}
