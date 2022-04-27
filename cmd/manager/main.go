// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/controller"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/dbsyncers"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	metricsHost                                  = "0.0.0.0"
	metricsPort                            int32 = 8384
	environmentVariableControllerNamespace       = "POD_NAMESPACE"
	environmentVariableDatabaseURL               = "DATABASE_URL"
	environmentVariableWatchNamespace            = "WATCH_NAMESPACE"
	environmentVariableSyncInterval              = "HOH_STATUS_SYNC_INTERVAL"
)

func printVersion(log logr.Logger) {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

// function to handle defers with exit, see https://stackoverflow.com/a/27629493/553720.
func doMain() int {
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.Logger())
	log := ctrl.Log.WithName("cmd")

	printVersion(log)

	leaderElectionNamespace, found := os.LookupEnv(environmentVariableControllerNamespace)
	if !found {
		log.Error(nil, "Not found:", "environment variable", environmentVariableControllerNamespace)
		return 1
	}

	namespace, found := os.LookupEnv(environmentVariableWatchNamespace)
	if !found {
		log.Error(nil, "Failed to get watch namespace")
		return 1
	}

	databaseURL, found := os.LookupEnv(environmentVariableDatabaseURL)
	if !found {
		log.Error(nil, "Not found:", "environment variable", environmentVariableDatabaseURL)
		return 1
	}

	// when switched to controller runtime 0.7, use the context returned by ctrl.SetupSignalHandler()
	dbConnectionPool, err := pgxpool.Connect(context.TODO(), databaseURL)
	if err != nil {
		log.Error(err, "Failed to connect to the database")
		return 1
	}
	defer dbConnectionPool.Close()

	syncIntervalString, found := os.LookupEnv(environmentVariableSyncInterval)
	if !found {
		log.Error(nil, "Not found:", "environment variable", environmentVariableSyncInterval)
		return 1
	}

	syncInterval, err := time.ParseDuration(syncIntervalString)
	if err != nil {
		log.Error(err, "the environment var ", environmentVariableSyncInterval, " is not valid duration")
		return 1
	}

	mgr, err := createManager(leaderElectionNamespace, namespace, metricsHost, metricsPort, dbConnectionPool, syncInterval)
	if err != nil {
		log.Error(err, "Failed to create manager")
		return 1
	}

	log.Info("Starting the Cmd.")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		return 1
	}

	return 0
}

func createManager(leaderElectionNamespace, namespace, metricsHost string, metricsPort int32,
	dbConnectionPool *pgxpool.Pool, syncInterval time.Duration,
) (ctrl.Manager, error) {
	options := ctrl.Options{
		Namespace:               namespace,
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		LeaderElection:          true,
		LeaderElectionNamespace: leaderElectionNamespace,
		LeaderElectionID:        "hub-of-hubs-all-in-one-sync-lock",
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new manager: %w", err)
	}

	if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add schemes: %w", err)
	}

	if err := dbsyncers.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add schemes: %w", err)
	}

	if err := controller.AddControllers(mgr, dbConnectionPool); err != nil {
		return nil, fmt.Errorf("failed to add controllers: %w", err)
	}

	if err := dbsyncers.AddDBSyncers(mgr, dbConnectionPool, syncInterval); err != nil {
		return nil, fmt.Errorf("failed to add db syncers: %w", err)
	}

	return mgr, nil
}

func main() {
	os.Exit(doMain())
}
