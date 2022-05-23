[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# Hub-of-Hubs-Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/stolostron/hub-of-hubs-manager)](https://goreportcard.com/report/github.com/stolostron/hub-of-hubs-manager)
[![Go Reference](https://pkg.go.dev/badge/github.com/stolostron/hub-of-hubs-manager.svg)](https://pkg.go.dev/github.com/stolostron/hub-of-hubs-manager)
[![License](https://img.shields.io/github/license/stolostron/hub-of-hubs-manager)](/LICENSE)

The manager component of [Hub-of-Hubs](https://github.com/stolostron/hub-of-hubs).

Go to the [Contributing guide](CONTRIBUTING.md) to learn how to get involved.

## Getting Started

## Environment variables

The following environment variables are required for the most tasks below:

* `REGISTRY`, for example `quay.io/open-cluster-management-hub-of-hubs`.
* `IMAGE_TAG`, for example `latest` or `v0.1.0`.

## Build

```bash
make build
```

## Run Locally

Disable the currently running controller in the cluster (if previously deployed):

```bash
kubectl scale deployment hub-of-hubs-manager -n open-cluster-management --replicas 0
```

Run with hub-of-hubs kubeconfig:

```bash
./bin/hub-of-hubs-manager --kubeconfig $TOP_HUB_CONFIG --process-database-url=$PROCESS_DATABASE_URL --transport-bridge-database-url=$TRANSPORT_BRIDGE_DATABASE_URL
```

## Build image

```bash
make build-images
```

## Deploy to a cluster

1.  Create two secrets with your database url:

    ```bash
    kubectl create secret generic hub-of-hubs-database-secret -n open-cluster-management --from-literal=url=$PROCESS_DATABASE_URL
    kubectl create secret generic hub-of-hubs-database-transport-bridge-secret -n open-cluster-management --from-literal=url=$TRANSPORT_BRIDGE_DATABASE_URL
    ```

2.  Deploy the operator:

    ```bash
    TRANSPORT_TYPE=kafka REGISTRY=quay.io/open-cluster-management-hub-of-hubs IMAGE_TAG=latest envsubst < deploy/operator.yaml.template | kubectl apply -n open-cluster-management -f -
    ```
