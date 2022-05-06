[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# Hub-of-Hubs-Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/stolostron/hub-of-hubs-manager)](https://goreportcard.com/report/github.com/stolostron/hub-of-hubs-manager)
[![Go Reference](https://pkg.go.dev/badge/github.com/stolostron/hub-of-hubs-manager.svg)](https://pkg.go.dev/github.com/stolostron/hub-of-hubs-manager)
[![License](https://img.shields.io/github/license/stolostron/hub-of-hubs-manager)](/LICENSE)

The manager component of [Hub-of-Hubs](https://github.com/stolostron/hub-of-hubs).

Go to the [Contributing guide](CONTRIBUTING.md) to learn how to get involved.

<!-- ## The dependencies chart

![Dependencies](diagrams/dependencies.svg)

## The reconciliation flow

![Reconciliation Flow](diagrams/flowchart.svg) -->

## Getting Started

## Environment variables

The following environment variables are required for the most tasks below:

* `REGISTRY`, for example `quay.io/morvencao`.
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

Set the following environment variables:

* POD_NAMESPACE
* WATCH_NAMESPACE
* PROCESS_DATABASE_URL
* TRANSPORT_BRIDGE_DATABASE_URL
* TRANSPORT_TYPE
* TRANSPORT_MESSAGE_COMPRESSION_TYPE
* KAFKA_PRODUCER_ID
* KAFKA_CONSUMER_ID
* KAFKA_PRODUCER_TOPIC
* KAFKA_CONSUMER_TOPIC
* KAFKA_BOOTSTRAP_SERVERS
* KAFKA_MESSAGE_SIZE_LIMIT_KB
* SYNC_SERVICE_PROTOCOL
* SYNC_SERVICE_HOST
* SYNC_SERVICE_PORT
* SYNC_SERVICE_POLLING_INTERVAL
* COMMITTER_INTERVAL
* STATISTICS_LOG_INTERVAL_SECONDS
* STATUS_SYNC_INTERVAL
* STATUS_SYNC_INTERVAL
* DELETED_LABELS_TRIMMING_INTERVAL

<!-- `POD_NAMESPACE` should usually be `open-cluster-management`.

`WATCH_NAMESPACE` can be defined empty so the controller will watch all the namespaces.

Set the `DATABASE_URL` according to the PostgreSQL URL format: `postgres://YourUserName:YourURLEscapedPassword@YourHostname:5432/YourDatabaseName?sslmode=verify-full&pool_max_conns=50`.

:exclamation: Remember to URL-escape the password, you can do it in bash:

```
python -c "import sys, urllib as ul; print ul.quote_plus(sys.argv[1])" 'YourPassword'
```

`STATUS_SYNC_INTERVAL` is the interval status sync, default value is `5s`. -->

Run with hub-of-hubs kubeconfig:

```bash
./bin/hub-of-hubs-manager --kubeconfig $TOP_HUB_CONFIG
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
    TRANSPORT_TYPE=kafka envsubst < deploy/operator.yaml.template | kubectl apply -n open-cluster-management -f -
    ```
