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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
	Scheme             *runtime.Scheme
	DefaultCredentials *scope.Credentials
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
	coxCluster := &coxv1.CoxCluster{}
	coxClusterName := client.ObjectKey{
		Namespace: coxMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, coxClusterName, coxCluster); err != nil {
		return reconcile.Result{}, err
	}

	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:             r.Client,
		Logger:             logger,
		Cluster:            cluster,
		CoxMachine:         coxMachine,
		CoxCluster:         coxCluster,
		Machine:            machine,
		DefaultCredentials: r.DefaultCredentials,
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

	return r.reconcile(ctx, machineScope, logger)
}

func (r *CoxMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Reconciling CoxMachine")
	coxMachine := machineScope.CoxMachine

	// Add the finalizer to the CoxMachine if it does not exist yet.
	controllerutil.AddFinalizer(coxMachine, coxv1.MachineFinalizer)

	// Make sure that the cluster infrastructure is ready.
	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure that bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	// If the CoxMachine is in an error state, return early.
	if coxMachine.Status.ErrorMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return ctrl.Result{}, fmt.Errorf(*coxMachine.Status.ErrorMessage)
	}

	// Set the ProviderID if the CoxMachine is already present
	if machineScope.GetProviderID() == "" {
		workload, err := machineScope.CoxClient.GetWorkloadByName(machineScope.CoxMachine.Name)
		if err != nil && err != coxedge.ErrWorkloadNotFound {
			return ctrl.Result{}, err
		}

		// If machine is not ready check for provisioning status
		if !machineScope.CoxMachine.Status.Ready && machineScope.CoxMachine.Status.TaskID != "" {
			t, err := machineScope.CoxClient.GetTask(machineScope.CoxMachine.Status.TaskID)
			if err != nil {
				return ctrl.Result{}, err
			}

			machineScope.CoxMachine.Status.TaskStatus = t.Data.Status

			switch machineScope.CoxMachine.Status.TaskStatus {
			case "SUCCESS":
				// once the workload is "RUNNING" set provider ID and machine status to ready
				machineScope.CoxMachine.Status.Ready = true
				machineScope.SetProviderID(t.Data.Result.WorkloadID)
			case "FAILURE":
				return ctrl.Result{}, fmt.Errorf("provisioning of workload failed")
			default:
				return ctrl.Result{
					// Requeue until the machine is ready
					RequeueAfter: 1 * time.Minute,
				}, nil
			}
		}

		if workload != nil {
			machineScope.SetProviderID(workload.ID)
		}
	}

	var (
		workload   *coxedge.Workload
		err        error
		workloadID string
	)
	providerID := machineScope.GetInstanceID()
	if providerID != "" {
		workload, _, err = machineScope.CoxClient.GetWorkload(providerID)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	bootstrapData, err := machineScope.GetRawBootstrapData()
	if err != nil {
		return ctrl.Result{}, err
	}

	machineType := machineScope.CoxMachine.Spec.Type
	if len(machineType) == 0 {
		machineType = coxedge.TypeVM
	}

	if workload == nil {
		// create workload
		data := &coxedge.CreateWorkloadRequest{
			Name:                machineScope.Name(),
			Type:                machineType,
			Image:               machineScope.CoxMachine.Spec.Image,
			AddAnyCastIPAddress: machineScope.CoxMachine.Spec.AddAnyCastIPAddress,
			FirstBootSSHKey:     strings.Join(machineScope.CoxMachine.Spec.SSHAuthorizedKeys, "\n"),
			Specs:               machineScope.CoxMachine.Spec.Specs,
			UserData:            bootstrapData,
		}

		data.Ports = []coxedge.Port{}

		for _, port := range machineScope.CoxMachine.Spec.Ports {
			p := coxedge.Port{
				Protocol:       port.Protocol,
				PublicPort:     port.PublicPort,
				PublicPortDesc: port.PublicPortDesc,
			}

			data.Ports = append(data.Ports, p)
		}

		data.Deployments = []coxedge.Deployment{}
		for _, deployment := range machineScope.CoxMachine.Spec.Deployments {
			d := coxedge.Deployment{
				Name:               deployment.Name,
				Pops:               deployment.Pops,
				EnableAutoScaling:  deployment.EnableAutoScaling,
				InstancesPerPop:    deployment.InstancesPerPop,
				CPUUtilization:     deployment.CPUUtilization,
				MinInstancesPerPop: deployment.MinInstancesPerPop,
				MaxInstancesPerPop: deployment.MaxInstancesPerPop,
			}
			data.Deployments = append(data.Deployments, d)
		}

		resp, errResp, err := machineScope.CoxClient.CreateWorkload(data)

		if err != nil {
			jsn, _ := json.Marshal(errResp)
			return ctrl.Result{}, fmt.Errorf("error occured while creating workload: %v - response: %v", err, string(jsn))
		}

		// Since the workload has just been created we have to requeue and poll for provisioning status with task ID
		machineScope.CoxMachine.Status.TaskID = resp.TaskId

		return ctrl.Result{
			RequeueAfter: 1 * time.Minute,
		}, nil
	} else {
		workloadID = workload.Data.ID
	}

	instances, _, err := machineScope.CoxClient.GetInstances(workloadID)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(instances.Data) == 0 {
		logger.Info("Instance not ready yet.")
		return ctrl.Result{
			RequeueAfter: 1 * time.Minute,
		}, nil
	}
	// For a CoxMachine we currently just assume 1 CAPI Machine == 1 Cox Workload == 1 Cox Instance
	instance := instances.Data[0]

	machineScope.SetAddresses([]corev1.NodeAddress{
		{
			Type:    corev1.NodeExternalIP,
			Address: instance.PublicIPAddress,
		},
		{
			Type:    corev1.NodeInternalIP,
			Address: instance.IPAddress,
		},
	})

	return ctrl.Result{
		// Requeue to make sure that the CoxMachine controller detects when the VM died on CoxEdge
		RequeueAfter: 5 * time.Minute,
	}, nil
}

func (r *CoxMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Deleting machine")
	// check if workload exists
	providerID := machineScope.GetInstanceID()
	wl, resp, err := machineScope.CoxClient.GetWorkload(providerID)
	if err != nil {
		if resp.StatusCode == 404 {
			logger.Info("unable to find CoxMachine", "errors", resp.Errors)
		} else {
			return ctrl.Result{}, err
		}
	}

	if wl != nil {
		if providerID != "" {
			_, _, err := machineScope.CoxClient.DeleteWorkload(providerID)
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
