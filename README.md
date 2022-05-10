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

* POD_NAMESPACE - the leader election namespace
* WATCH_NAMESPACE - the watched namespaces, multiple namespace splited by comma.
* PROCESS_DATABASE_URL - the URL of the database server for process user
* TRANSPORT_BRIDGE_DATABASE_URL - the URL of the database server for transport-bridge user
* TRANSPORT_TYPE - the transport type, "kafka" or "sync-service"
* TRANSPORT_MESSAGE_COMPRESSION_TYPE - the compression type for transport message, "gzip" or "no-op"
* KAFKA_PRODUCER_ID - the ID for kafka producer, "hub-of-hubs" by default
* KAFKA_CONSUMER_ID - the ID for kafka consumer, "hub-of-hubs" by default
* KAFKA_PRODUCER_TOPIC - the topic for kafka producer at hub-of-hubs side, should be "spec"
* KAFKA_CONSUMER_TOPIC - the topic for kafka consumer at hub-of-hubs side, should be "status"
* KAFKA_BOOTSTRAP_SERVERS - the bootstrap server of kafka, "kafka-brokers-cluster-kafka-bootstrap.kafka.svc:9092" by default
* KAFKA_MESSAGE_SIZE_LIMIT_KB - the limit size of kafka message, should be < 1000
* SYNC_SERVICE_PROTOCOL - the protocol of sync-service, "http" by default
* SYNC_SERVICE_HOST - the host of of sync-service, "sync-service-css.sync-service.svc.cluster.local" by default
* SYNC_SERVICE_PORT - the port of of sync-service, "9689" by default
* SYNC_SERVICE_POLLING_INTERVAL - the polling interval of sync-service, "5" by default
* COMMITTER_INTERVAL - the committer interval of status-transport-bridge, "40s" by default
* STATISTICS_LOG_INTERVAL_SECONDS - the interval for the statistics log, "0" by default
* SPEC_SYNC_INTERVAL - the interval of spec-sync, "5s" by default
* STATUS_SYNC_INTERVAL - the interval of status-sync, "5s" by default
* DELETED_LABELS_TRIMMING_INTERVAL - the interval of deleted label trimming, "1h" by default
* CLUSTER_API_URL - the URL of the Kubernetes API server
* CLUSTER_API_CA_BUNDLE_PATH` - the CA bundle for the Kubernetes API server. If not provided, verification of the server certificates is skipped.
* AUTHORIZATION_URL - the URL of the authorization server
* AUTHORIZATION_CA_BUNDLE_PATH - the CA bundle for the authorization server. If not provided, verification of the server certificates is skipped.
* SERVER_CERTIFICATE_PATH - the path to the file that contains the certificate for this server's TLS
* SERVER_KEY_PATH - the path to the file that contains the private key for this server's TLS
* SERVER_BASE_PATH - the base path of non-k8s-api server

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
