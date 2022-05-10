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

Ensure that the Cox provider has the required credentials, create a ConfigMap with following fields:

Create a file named `coxedge-config.yaml
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: coxedge
  namespace: capc-system
stringData:
  COX_API_KEY: <COX_API_KEY>
  COX_SERVICE: <COX_SERVICE>
  COX_ENVIRONMENT: <COX_ENVIRONMENT>
```

Apply the config to the target cluster:
```shell
kubectl apply -f ./coxedge-config.yaml
```

To deploy the provider with clusterctl:
```shell
clusterctl init --infrastructure cox
```