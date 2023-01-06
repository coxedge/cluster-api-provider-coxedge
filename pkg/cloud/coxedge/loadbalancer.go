package coxedge

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	EnvKeyLBBackends = "LB_BACKENDS"
	EnvKeyLBPort     = "LB_PORT"
)

type LoadBalancer struct {
	Spec   LoadBalancerSpec
	Status LoadBalancerStatus
}

type LoadBalancerSpec struct {
	Name     string
	Port     string
	Image    string
	Backends []string
	POP      []string
}

type LoadBalancerStatus struct {
	PublicIP string
}

// LoadBalancerHelper is a manager for creating workload-based load-balancers
type LoadBalancerHelper struct {
	Client *Client
}

func NewLoadBalancerHelper(client *Client) *LoadBalancerHelper {
	return &LoadBalancerHelper{Client: client}
}

func (l *LoadBalancerHelper) GetLoadBalancer(ctx context.Context, name string) (*LoadBalancer, error) {
	workload, err := l.Client.GetWorkloadByName(name)
	if err != nil {
		return nil, err
	}

	instances, err := l.Client.GetInstances(workload.ID)
	if err != nil {
		return nil, err
	}

	return parseLoadBalancerFromWorkload(workload, instances.Data)
}

func (l *LoadBalancerHelper) CreateLoadBalancer(ctx context.Context, payload *LoadBalancerSpec) error {
	_, err := l.Client.CreateWorkload(&CreateWorkloadRequest{
		Name:                payload.Name,
		Type:                TypeContainer,
		Image:               payload.Image,
		AddAnyCastIPAddress: true,
		Ports: []Port{
			{
				Protocol:   PortProtocolTCP,
				PublicPort: payload.Port,
			},
		},
		EnvironmentVariables: []EnvironmentVariable{
			{
				Key:   EnvKeyLBPort,
				Value: payload.Port,
			},
			{
				Key:   EnvKeyLBBackends,
				Value: strings.Join(payload.Backends, ";"),
			},
		},
		Deployments: []Deployment{
			{
				Name:            "default",
				Pops:            payload.POP,
				InstancesPerPop: "1",
			},
		},
		Specs: SpecSP1,
	})
	if err != nil {
		return fmt.Errorf("failed to create loadBalancer: %w", err)
	}
	return nil
}

func (l *LoadBalancerHelper) UpdateLoadBalancer(ctx context.Context, payload *LoadBalancerSpec) error {
	workload, err := l.Client.GetWorkloadByName(payload.Name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	existingLoadBalancerSpec, err := parseLoadBalancerSpecFromWorkload(workload)
	if err != nil {
		return err
	}

	// TODO support updating the port (needs updates to the network policy in CoxEdge)
	if payload.Port != existingLoadBalancerSpec.Port {
		return errors.New("updating the LoadBalancer port is not supported")
	}

	workload.EnvironmentVariable = []EnvironmentVariable{
		{
			Key:   EnvKeyLBBackends,
			Value: strings.Join(payload.Backends, ";"),
		},
		{
			Key:   EnvKeyLBPort,
			Value: existingLoadBalancerSpec.Port,
		},
	}

	_, err = l.Client.UpdateWorkload(workload.ID, *workload)
	if err != nil {
		return fmt.Errorf("failed to update loadBalancer: %w", err)
	}
	return nil
}

func (l *LoadBalancerHelper) DeleteLoadBalancer(ctx context.Context, name string) error {
	workload, err := l.Client.GetWorkloadByName(name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	_, err = l.Client.DeleteWorkload(workload.ID)
	if err != nil {
		return err
	}
	return nil
}

func parseLoadBalancerFromWorkload(workload *WorkloadData, workloadInstances []InstanceData) (*LoadBalancer, error) {
	spec, err := parseLoadBalancerSpecFromWorkload(workload)
	if err != nil {
		return nil, err
	}

	status, err := parseLoadBalancerStatusFromWorkload(workload, workloadInstances)
	if err != nil {
		return nil, err
	}

	return &LoadBalancer{
		Spec:   *spec,
		Status: *status,
	}, nil
}

func parseLoadBalancerStatusFromWorkload(workload *WorkloadData, workloadInstances []InstanceData) (*LoadBalancerStatus, error) {
	status := &LoadBalancerStatus{}

	for _, inst := range workloadInstances {
		if inst.Status == "RUNNING" {
			if workload != nil {
				status.PublicIP = workload.AnycastIPAddress
			}
		}
	}

	return status, nil
}

func parseLoadBalancerSpecFromWorkload(workload *WorkloadData) (*LoadBalancerSpec, error) {
	var backends []string
	var port string
	for _, kv := range workload.EnvironmentVariable {
		switch kv.Key {
		case EnvKeyLBBackends:
			backends = strings.Split(kv.Value, ";")
		case EnvKeyLBPort:
			port = kv.Value
		}
	}

	if backends == nil {
		return nil, errors.New("workload is not a load-balancer")
	}

	return &LoadBalancerSpec{
		Name:     workload.Name,
		Port:     port,
		Image:    workload.Image,
		Backends: backends,
		POP:      workload.Deployments[0].Pops,
	}, nil
}
