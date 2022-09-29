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
	"errors"
	"fmt"
	"net/http"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
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

	coxv1 "github.com/coxedge/cluster-api-provider-cox/api/v1beta1"
	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge"
	"github.com/coxedge/cluster-api-provider-cox/pkg/cloud/coxedge/scope"
	"github.com/go-logr/logr"
)

var (
	errWorkloadDeploymentInProgress = errors.New("machine deployment is still in progress")
	errWorkloadDeploymentNotFound   = errors.New("machine deployment has not been started")
)

const (
	CoxMachineControllerName = "CoxMachine"
)

const (
	CoxMachineReadyCondition clusterv1.ConditionType = "CoxMachineReady"
	// ClusterNotFoundReason used when the machine is missing the cluster
	ClusterNotFoundReason = "ClusterNotFound"
	// ClusterInfrastructureNotReadyReason used when the InfrastractureReady status is false
	ClusterInfrastructureNotReadyReason = "ClusterInfrastructureNotReady"
	// BootstrapNotAvailableReason used when the Bootstrap data reference is not yet available
	BootstrapNotAvailableReason = "BootstrapNotAvailable"
	// BootstrapDataNotFoundReason used when MachineScope fails to get Bootstrap data
	BootstrapDataNotFoundReason = "BootstrapDataNotFound"
	// MachineErroredStateReason used when CoxMachine enters errored state
	MachineErroredStateReason = "MachineErroredState"
	// WorkloadCreateFailedReason used when CoxClient fails to create a Workload
	WorkloadCreateFailedReason = "WorkloadCreateFailed"
	// FailedWorkloadReconcileReason used when failing to set ProviderID and Workload failes to reconcile
	FailedWorkloadReconcileReason = "FailedWorkloadReconcile"
	// InstanceNotReady used when the instance is not ready yet
	InstanceNotReady = "InstanceNotReady"
)

// CoxMachineReconciler reconciles a CoxMachine object
type CoxMachineReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Recorder           record.EventRecorder
	DefaultCredentials *scope.Credentials
	Tracker            *remote.ClusterCacheTracker
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
		// Allow the cluster to be empty, because it is not needed in the deletion logic.
		if apierrors.IsNotFound(err) {
			logger.Info("CoxCluster not found.", "cluster", coxClusterName)
		} else {
			return reconcile.Result{}, err
		}
	}

	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:             r.Client,
		Logger:             logger,
		Cluster:            cluster,
		CoxMachine:         coxMachine,
		CoxCluster:         coxCluster,
		Machine:            machine,
		DefaultCredentials: r.DefaultCredentials,
		Tracker:            r.Tracker,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create scope: %w", err)
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

	return r.reconcileNormal(ctx, machineScope, logger)
}

func (r *CoxMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Reconciling CoxMachine")
	coxMachine := machineScope.CoxMachine
	conditions.MarkUnknown(coxMachine, CoxMachineReadyCondition, "", "")

	// Add the finalizer to the CoxMachine if it does not exist yet.
	controllerutil.AddFinalizer(coxMachine, coxv1.MachineFinalizer)

	// Check if the cluster was found
	if machineScope.Cluster == nil {
		cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machineScope.Machine.ObjectMeta)
		if err != nil {
			logger.Info("Machine is missing cluster label or cluster does not exist")
			conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, ClusterNotFoundReason, clusterv1.ConditionSeverityInfo, "Machine is missing cluster label or cluster does not exist")
			return ctrl.Result{}, nil
		}
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, ClusterNotFoundReason, clusterv1.ConditionSeverityInfo, err.Error())
		return reconcile.Result{}, apierrors.NewNotFound(
			coxv1.GroupVersion.WithResource("coxclusters").GroupResource(),
			cluster.Spec.InfrastructureRef.Name,
		)
	}

	// Make sure that the cluster infrastructure is ready.
	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, ClusterInfrastructureNotReadyReason, clusterv1.ConditionSeverityInfo, "Cluster infrastructure is not ready yet")
		return reconcile.Result{}, nil
	}

	// Make sure that bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, BootstrapNotAvailableReason, clusterv1.ConditionSeverityInfo, "Bootstrap data secret reference is not yet available")
		return reconcile.Result{}, nil
	}

	// If the CoxMachine is in an error state, return early.
	if coxMachine.Status.ErrorMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, MachineErroredStateReason, clusterv1.ConditionSeverityInfo, *coxMachine.Status.ErrorMessage)
		return ctrl.Result{}, fmt.Errorf(*coxMachine.Status.ErrorMessage)
	}

	// Set the ProviderID if the CoxMachine is already present=
	err := r.reconcileWorkload(machineScope)
	if err != nil {
		switch err {
		case errWorkloadDeploymentNotFound, coxedge.ErrWorkloadNotFound:
			logger.Info("No CoxEdge workload found for this machine; creating it.")
			bootstrapData, err := machineScope.GetRawBootstrapData()
			if err != nil {
				conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, BootstrapDataNotFoundReason, clusterv1.ConditionSeverityInfo, err.Error())
				return ctrl.Result{}, fmt.Errorf("failed to get bootstrap data: %w", err)
			}

			data := &coxedge.CreateWorkloadRequest{
				Name:                machineScope.Name(),
				Type:                coxedge.TypeVM,
				Image:               machineScope.CoxMachine.Spec.Image,
				AddAnyCastIPAddress: machineScope.CoxMachine.Spec.AddAnyCastIPAddress,
				FirstBootSSHKey:     strings.Join(machineScope.CoxMachine.Spec.SSHAuthorizedKeys, "\n"),
				Specs:               machineScope.CoxMachine.Spec.Specs,
				UserData:            bootstrapData,
			}

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

			resp, err := machineScope.CoxClient.CreateWorkload(data)
			if err != nil {
				conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, WorkloadCreateFailedReason, clusterv1.ConditionSeverityInfo, err.Error())
				errResp := &coxedge.HTTPError{}
				if errors.As(err, &errResp) {
					jsn, _ := json.Marshal(errResp)
					r.Recorder.Eventf(machineScope.CoxMachine, corev1.EventTypeNormal, "CreatingWorkloadFailed", "Failed to create machine '%s`:`%s`", machineScope.Machine.Name, machineScope.Machine.UID, string(jsn))
					return ctrl.Result{}, fmt.Errorf("error occurred while creating workload: %v - response: %v", err, string(jsn))
				}
				r.Recorder.Eventf(machineScope.CoxMachine, corev1.EventTypeNormal, "CreatingWorkloadFailed", "Failed to create workflow for machine '%s`:`%s`", machineScope.Machine.Name, machineScope.Machine.UID, err.Error())
				return ctrl.Result{}, fmt.Errorf("error occurred while creating workload: %w", err)
			}

			r.Recorder.Eventf(machineScope.CoxMachine, corev1.EventTypeNormal, "CreatedWorkload", "Created workload for machine '%s`:`%s`", machineScope.Machine.Name, machineScope.Machine.UID)

			// Since the workload has just been created we have to requeue and poll for provisioning status with task ID
			machineScope.CoxMachine.Status.TaskID = resp.TaskID
			return ctrl.Result{}, nil
		case errWorkloadDeploymentInProgress:
			return ctrl.Result{
				// Requeue until the machine is ready
				RequeueAfter: 1 * time.Minute,
			}, nil
		default:
			conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, FailedWorkloadReconcileReason, clusterv1.ConditionSeverityInfo, err.Error())
			return ctrl.Result{}, fmt.Errorf("error while reconciling workload: %w", err)
		}
	}

	workloadID := machineScope.GetWorkloadID()
	logger.Info("Checking the workload's instance status", "workloadID", workloadID)
	instances, err := machineScope.CoxClient.GetInstances(workloadID)
	if err != nil {
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, InstanceNotReady, clusterv1.ConditionSeverityInfo, err.Error())
		return ctrl.Result{}, err
	}
	if len(instances.Data) == 0 {
		logger.Info("Instance not deployed yet.")
		conditions.MarkFalse(coxMachine, CoxMachineReadyCondition, InstanceNotReady, clusterv1.ConditionSeverityInfo, "Instance not deployed yet.")
		return ctrl.Result{
			RequeueAfter: 1 * time.Minute,
		}, nil
	}
	// For a CoxMachine we currently just assume 1 CAPI Machine == 1 Cox Workload == 1 Cox Instance
	instance := instances.Data[0]

	// It can happen that an instance is stuck in SCHEDULING for a longer time.
	if instance.Status != "RUNNING" {
		logger.Info("Instance not ready yet.")
		return ctrl.Result{
			RequeueAfter: 1 * time.Minute,
		}, nil
	}

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

	conditions.MarkTrue(machineScope.CoxMachine, CoxMachineReadyCondition)

	err = machineScope.SetNodeProviderID()
	if err != nil {
		return ctrl.Result{}, err
	}

	machineScope.CoxMachine.Status.Ready = true
	return ctrl.Result{
		// Requeue to make sure that the CoxMachine controller detects when the VM died on CoxEdge
		RequeueAfter: 5 * time.Minute,
	}, nil
}

func (r *CoxMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Deleting machine")
	err := r.reconcileWorkload(machineScope)
	if err != nil {
		switch err {
		case errWorkloadDeploymentNotFound, coxedge.ErrWorkloadNotFound:
			logger.Info("Machine does not contain a ProviderID or TaskID; assuming that the machine deployment never started.")
			controllerutil.RemoveFinalizer(machineScope.CoxMachine, coxv1.MachineFinalizer)
			return ctrl.Result{}, nil
		case errWorkloadDeploymentInProgress:
			logger.Info("Machine deployment still in progress, waiting for it to complete before deleting the machine.")
			return ctrl.Result{
				// Requeue until the machine is ready
				RequeueAfter: 1 * time.Minute,
			}, nil
		default:
			return ctrl.Result{}, err
		}
	}

	workloadID := machineScope.GetWorkloadID()
	logger.Info("Checking if the workload already has been deleted")
	_, err = machineScope.CoxClient.GetWorkload(workloadID)
	if err != nil {
		respErr := &coxedge.HTTPError{}
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			logger.Info("Could not find workload, assuming it was deleted.")
			controllerutil.RemoveFinalizer(machineScope.CoxMachine, coxv1.MachineFinalizer)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to delete the machine: %v", err)
	}

	logger.Info("Deleting the machine", "workloadID", workloadID)
	_, err = machineScope.CoxClient.DeleteWorkload(workloadID)
	if err != nil {
		r.Recorder.Eventf(machineScope.CoxMachine, corev1.EventTypeNormal, "DeletingWorkloadFailed", "Failed to delete Machine '%s", machineScope.Machine.Name)
		return ctrl.Result{}, fmt.Errorf("failed to delete the machine: %v", err)
	}

	r.Recorder.Eventf(machineScope.CoxMachine, corev1.EventTypeNormal, "DeletedWorkload", "Deleted Machine '%s`:`%s`", machineScope.Machine.Name, machineScope.Machine.UID)
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
		return fmt.Errorf("error creating controller: %w", err)
	}

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &coxv1.CoxMachineList{}, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to create mapper for Cluster to CoxMachines: %w", err)
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return fmt.Errorf("failed adding a watch for ready clusters: %w", err)
	}

	return nil
}

func (r *CoxMachineReconciler) CoxClusterToCoxMachines(ctx context.Context) handler.MapFunc {
	log := ctrl.LoggerFrom(ctx)
	return func(o client.Object) []ctrl.Request {
		var result []ctrl.Request

		c, ok := o.(*coxv1.CoxCluster)
		if !ok {
			log.Error(fmt.Errorf("expected a CoxCluster but got a %T", o), "failed to get CoxMachine for CoxCluster")
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

// reconcileWorkload tries to determine the ProviderID / WorkloadID
// associated with the CoxMachine. If it could be retrieved, it will set it has
// the ProviderID of the workload. If not, it will return an error.
//
// Note: it does not guarantee that thew referenced workload exists.
func (r *CoxMachineReconciler) reconcileWorkload(machineScope *scope.MachineScope) error {
	workload, err := machineScope.CoxClient.GetWorkloadByName(machineScope.CoxMachine.Name)
	if err != nil {
		if err != coxedge.ErrWorkloadNotFound {
			return err
		}
		if len(machineScope.CoxMachine.Spec.ProviderID) > 0 {
			return coxedge.ErrWorkloadNotFound
		}

		if machineScope.CoxMachine.Status.TaskID == "" {
			return errWorkloadDeploymentNotFound
		}

		// If machine is not ready check for provisioning status
		task, err := machineScope.CoxClient.GetTask(machineScope.CoxMachine.Status.TaskID)
		if err != nil {
			return err
		}
		machineScope.CoxMachine.Status.TaskStatus = task.Data.Status

		switch machineScope.CoxMachine.Status.TaskStatus {
		case "SUCCESS":
			machineScope.SetProviderID(task.Data.Result.WorkloadID)
		case "FAILURE":
			return fmt.Errorf("provisioning of workload failed")
		default:
			return errWorkloadDeploymentInProgress
		}
	} else {
		machineScope.SetProviderID(workload.ID)
	}
	return nil
}
