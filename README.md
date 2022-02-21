# Kubernetes Cluster API Provider Cox

Kubernetes-native declarative infrastructure for [Cox Edge](https://www.coxedge.com).

## Installation

Before you can deploy the infrastructure controller, you’ll need to deploy Cluster API itself.

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

Apply the config:
```shell
kubectl apply -f ./coxedge-config.yaml
```

To deploy from the latest build:
```shell
# Build and push the controller image
make docker-build docker-push IMG=$DOCKER_USER/cluster-api-provider-cox-controller:latest

# Deploy the provider to your current cluster
make deploy IMG=$DOCKER_USER/cluster-api-provider-cox-controller:latest
```

