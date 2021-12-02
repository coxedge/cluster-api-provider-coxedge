# Kubernetes Cluster API Provider Cox

Kubernetes-native declarative infrastructure for [Cox Edge](https://www.coxedge.com).

## Installation

This assumes you already have a CAPI management plane.

To deploy from the latest build:
```shell
# Build and push the controller image
make docker-build docker-push IMG=$DOCKER_USER/cluster-api-provider-cox-controller:latest

# Deploy the provider to your current cluster
make deploy IMG=$DOCKER_USER/cluster-api-provider-cox-controller:latest
```

Finally, ensure that the cox provider has the required parameters, create a ConfigMap with the following fields:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coxedge
  namespace: cluster-api-provider-cox-system
data:
  coxapikey: <COX_API_KEY>
  coxservice: <COX_SERVICE>
  coxenvironment: <COX_ENVIRONMENT>
```

```shell
kubectl apply -f ./coxedge-config.yaml
```