# Kubernetes Cluster API Provider Cox

Kubernetes-native declarative infrastructure for [Cox Edge](https://www.coxedge.com).

## Installation

Before you can deploy the infrastructure controller, you’ll need to configure 
and deploy Cluster API itself.

First, you will need to update your `clusterctl` config to be able to discover 
the provider, which is located by default `~/.cluster-api/clusterctl.yaml`.

```yaml
providers:
  # Add the cox infrastructure provider to the clusterctl config for discovery
  - name: cox
    type: InfrastructureProvider
    url: https://github.com/coxedge/cluster-api-provider-cox/releases/latest/
  # or, use a local provider (replace the `/path/to` with the path to this repository).
  - name: cox-local
    type: InfrastructureProvider
    url: /path/to/cluster-api-provider-cox/build/release/infrastructure-cox/latest/infrastructure-components.yaml
```

Then, deploy the core components of Cluster API. Clusterctl uses the kubeconfig
present in `KUBECONFIG` unless configured otherwise. To deploy:

```shell
clusterctl init
```

### release version

To deploy the provider with clusterctl:
```shell
clusterctl init --infrastructure cox
```

### dev version

#### Building Image 
Change `REGISTRY` `IMAGE_NAME` according to your setup
```shell
make docker-build && make docker-push
```

To deploy the provider with clusterctl:
```shell
clusterctl init --infrastructure cox-local
```

Or you can run 
```shell
make release-manifests-clusterctl

kubectl apply -f build/releases/infrastructure-cox/latest/infrastructure-components.yaml
```

### Sample Cluster Creation
```shell
kubectl apply -f examples/coxcluster.yaml
```