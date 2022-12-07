# Kubernetes Cluster API Provider Cox

<!-- <p align="center"><img alt="capi" src="https://cluster-api.sigs.k8s.io/#kubernetes-cluster-apidiv-stylefloat-right-position-relative-display-inlineimg-srcimagesintroductionsvg-width160px-div" width="160x" /><img alt="capi" src="https://www.google.com/url?sa=i&url=https%3A%2F%2Fwww.coxedge.com%2Fschedule&psig=AOvVaw36DdSzXhauYaKA4uJPD0RA&ust=1670324903288000&source=images&cd=vfe&ved=0CBAQjRxqFwoTCIDOgKer4vsCFQAAAAAdAAAAABAD" width="192x" /></p> -->

Kubernetes-native declarative infrastructure for [Cox Edge](https://www.coxedge.com).

## What is the Cluster API Provider Cox Edge

The [Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings declarative, Kubernetes-style APIs to cluster creation, configuration and
management. The API itself is shared across multiple cloud providers allowing for true Cox Edge
hybrid deployments of Kubernetes. 

## Compatibility with Cluster API

This provider's versions are compatible with the following versions of Cluster API:

|                             |Cluster API v1alpha4 (v0.4) |Cluster API v1beta1 (v1.x)  |
| --------------------------- |:-------------------------: |:-------------------------: |
| Cox Edge v1beta1  `(v0.4.x)`|              ☓             |              ✓             |

## Prerequisites

- You will need to update your clusterctl config to be able to discover the provider, which is located by default ~/.cluster-api/clusterctl.yaml.
```yaml
providers:
  # Add the cox infrastructure provider to the clusterctl config for discovery
  - name: coxedge
    type: InfrastructureProvider
    url: https://github.com/spectrocloud/cluster-api-provider-coxedge/releases/latest/
```

- Ensure that the Cox provider has the required credentials. You will need to add your credentials in the [examples/coxcluster.yaml](https://github.com/spectrocloud/cluster-api-provider-coxedge/blob/spv1docs/examples/coxcluster.yaml#L36) file.
```yaml
stringData:
  COX_API_KEY: <YOUR API KEY>
  COX_SERVICE: edge-service
  COX_ENVIRONMENT: <ENVIRONMENT NAME>
  # COX_ORGANIZATION: <ORGANIZATION ID>
```  

## Installation

### For Development

- #### Creating a kind cluster
```shell
kind create cluster
```

- #### Initialize the management cluster
```shell
clusterctl init --infrastructure coxedge
```

- #### Building Image 
Change `REGISTRY` and `IMAGE_NAME` according to your setup
```shell
make docker-build && make docker-push
```

- #### Cluster creation
```shell
make release-manifests-clusterctl

kubectl apply -f build/releases/infrastructure-cox/latest/infrastructure-components.yaml

kubectl apply -f examples/coxcluster.yaml
```
#### NOTE
Please Note that the coxcluster.yaml file must have your required credentials.

### Getting cluster info

- #### View cluster status
```shell
kubectl get cluster
```

- #### At glance view of cluster and resources
```shell
clusterctl describe cluster <cluster_name>
```
