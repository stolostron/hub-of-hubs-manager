[comment]: # ( Copyright Contributors to the Open Cluster Management project )

# Hub-of-Hubs-All-in-One

[![Go Report Card](https://goreportcard.com/badge/github.com/stolostron/hub-of-hubs-all-in-one)](https://goreportcard.com/report/github.com/stolostron/hub-of-hubs-all-in-one)
[![Go Reference](https://pkg.go.dev/badge/github.com/stolostron/hub-of-hubs-all-in-one.svg)](https://pkg.go.dev/github.com/stolostron/hub-of-hubs-all-in-one)
[![License](https://img.shields.io/github/license/stolostron/hub-of-hubs-all-in-one)](/LICENSE)

The all-in-one component of [Hub-of-Hubs](https://github.com/stolostron/hub-of-hubs).

Go to the [Contributing guide](CONTRIBUTING.md) to learn how to get involved.

<!-- ## The dependencies chart

![Dependencies](diagrams/dependencies.svg)

## The reconciliation flow

![Reconciliation Flow](diagrams/flowchart.svg) -->

## Getting Started

## Environment variables

The following environment variables are required for the most tasks below:

* `REGISTRY`, for example `docker.io/morvencao`.
* `IMAGE_TAG`, for example `v0.1.0`.

## Build

```
make build
```

## Run Locally

Disable the currently running controller in the cluster (if previously deployed):

```
kubectl scale deployment hub-of-hubs-all-in-one -n open-cluster-management --replicas 0
```

Set the following environment variables:

* POD_NAMESPACE
* WATCH_NAMESPACE
* DATABASE_URL
* HOH_STATUS_SYNC_INTERVAL

`POD_NAMESPACE` should usually be `open-cluster-management`.

`WATCH_NAMESPACE` can be defined empty so the controller will watch all the namespaces.

Set the `DATABASE_URL` according to the PostgreSQL URL format: `postgres://YourUserName:YourURLEscapedPassword@YourHostname:5432/YourDatabaseName?sslmode=verify-full&pool_max_conns=50`.

:exclamation: Remember to URL-escape the password, you can do it in bash:

```
python -c "import sys, urllib as ul; print ul.quote_plus(sys.argv[1])" 'YourPassword'
```

`HOH_STATUS_SYNC_INTERVAL` is the interval status sync, default value is `5s`.

```
./bin/hub-of-hubs-all-in-one --kubeconfig $TOP_HUB_CONFIG
```

## Build image

```
make build-images
```

## Deploy to a cluster

1.  Create a secret with your database url:

    ```
    kubectl create secret generic hub-of-hubs-database-secret -n open-cluster-management --from-literal=url=$DATABASE_URL
    ```

1.  Deploy the operator:

    ```
    COMPONENT=$(basename $(pwd)) envsubst < deploy/operator.yaml.template | kubectl apply -n open-cluster-management -f -
    ```
