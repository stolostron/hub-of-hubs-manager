// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/db/postgresql"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/scheme"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/dbtotransport"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/spectodb"
	dbtostatus "github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/dbtostatus"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/transport"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/transport/compressor"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/transport/kafka"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/transport/syncservice"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	metricsHost                             = "0.0.0.0"
	metricsPort                       int32 = 8384
	envVarControllerNamespace               = "POD_NAMESPACE"
	envVarWatchNamespace                    = "WATCH_NAMESPACE"
	envVarProcessDatabaseURL                = "PROCESS_DATABASE_URL"
	envVarTransportBridgeDatabaseURL        = "TRANSPORT_BRIDGE_DATABASE_URL"
	envVarTransportMsgCompressionType       = "TRANSPORT_MESSAGE_COMPRESSION_TYPE"
	envVarTransportType                     = "TRANSPORT_TYPE"
	kafkaTransportTypeName                  = "kafka"
	syncServiceTransportTypeName            = "sync-service"
	envVarSpecSyncInterval                  = "SPEC_SYNC_INTERVAL"
	envVarStatusSyncInterval                = "STATUS_SYNC_INTERVAL"
	envVarLabelsTrimmingInterval            = "DELETED_LABELS_TRIMMING_INTERVAL"
	leaderElectionLockName                  = "hub-of-hubs-all-in-one-lock"
)

var (
	errEnvVarNotFound     = errors.New("environment variable not found")
	errEnvVarIllegalValue = errors.New("environment variable illegal value")
)

func initializeLogger() logr.Logger {
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.Logger())
	log := ctrl.Log.WithName("cmd")

	return log
}

func printVersion(log logr.Logger) {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func readEnvVars() (string, string, string, string, string, string, time.Duration, time.Duration, time.Duration, error) {
	leaderElectionNamespace, found := os.LookupEnv(envVarControllerNamespace)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarControllerNamespace)
	}

	watchNamespace, found := os.LookupEnv(envVarWatchNamespace)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarWatchNamespace)
	}

	processDatabaseURL, found := os.LookupEnv(envVarProcessDatabaseURL)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarProcessDatabaseURL)
	}

	transportBridgeDatabaseURL, found := os.LookupEnv(envVarTransportBridgeDatabaseURL)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportBridgeDatabaseURL)
	}

	transportType, found := os.LookupEnv(envVarTransportType)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportType)
	}

	transportMsgCompressionType, found := os.LookupEnv(envVarTransportMsgCompressionType)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportMsgCompressionType)
	}

	specSyncIntervalString, found := os.LookupEnv(envVarSpecSyncInterval)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarSpecSyncInterval)
	}

	specSyncInterval, err := time.ParseDuration(specSyncIntervalString)
	if err != nil {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			specSyncInterval, err)
	}

	statusSyncIntervalString, found := os.LookupEnv(envVarStatusSyncInterval)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarStatusSyncInterval)
	}

	statusSyncInterval, err := time.ParseDuration(statusSyncIntervalString)
	if err != nil {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			statusSyncInterval, err)
	}

	deletedLabelsTrimmingIntervalString, found := os.LookupEnv(envVarLabelsTrimmingInterval)
	if !found {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarLabelsTrimmingInterval)
	}

	deletedLabelsTrimmingInterval, err := time.ParseDuration(deletedLabelsTrimmingIntervalString)
	if err != nil {
		return "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			deletedLabelsTrimmingInterval, err)
	}

	return leaderElectionNamespace, watchNamespace, processDatabaseURL, transportBridgeDatabaseURL, transportType, transportMsgCompressionType, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval, nil
}

// function to choose transport type based on env var.
func getTransport(transportType string, transportMsgCompressorType string) (transport.Transport, error) {
	msgCompressor, err := compressor.NewCompressor(compressor.CompressionType(transportMsgCompressorType))
	if err != nil {
		return nil, fmt.Errorf("failed to create message-compressor: %w", err)
	}

	switch transportType {
	case kafkaTransportTypeName:
		kafkaProducer, err := kafka.NewProducer(msgCompressor, ctrl.Log.WithName("kafka"))
		if err != nil {
			return nil, fmt.Errorf("failed to create kafka-producer: %w", err)
		}

		return kafkaProducer, nil
	case syncServiceTransportTypeName:
		syncService, err := syncservice.NewSyncService(msgCompressor, ctrl.Log.WithName("sync-service"))
		if err != nil {
			return nil, fmt.Errorf("failed to create sync-service: %w", err)
		}

		return syncService, nil
	default:
		return nil, fmt.Errorf("%w: %s - %s is not a valid option", errEnvVarIllegalValue, envVarTransportType,
			transportType)
	}
}

func createManager(leaderElectionNamespace, watchNamespace string, processPostgreSQL, transportBridgePostgreSQL *postgresql.PostgreSQL, transportObj transport.Transport,
	specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval time.Duration,
) (ctrl.Manager, error) {
	options := ctrl.Options{
		Namespace:               watchNamespace,
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		LeaderElection:          true,
		LeaderElectionNamespace: leaderElectionNamespace,
		LeaderElectionID:        leaderElectionLockName,
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More Info: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(watchNamespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(watchNamespace, ","))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new manager: %w", err)
	}

	if err := scheme.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("failed to add schemes: %w", err)
	}

	if err := spectodb.AddSpecToDBControllers(mgr, processPostgreSQL); err != nil {
		return nil, fmt.Errorf("failed to add spectodb controllers: %w", err)
	}

	if err := dbtostatus.AddDBSyncers(mgr, processPostgreSQL, statusSyncInterval); err != nil {
		return nil, fmt.Errorf("failed to add db syncers: %w", err)
	}

	if err := dbtotransport.AddDBToTransportSyncers(mgr, transportBridgePostgreSQL, transportObj, specSyncInterval); err != nil {
		return nil, fmt.Errorf("failed to add dbtotransport syncers: %w", err)
	}

	if err := dbtotransport.AddStatusDBWatchers(mgr, transportBridgePostgreSQL, transportBridgePostgreSQL, deletedLabelsTrimmingInterval); err != nil {
		return nil, fmt.Errorf("failed to add status db watchers: %w", err)
	}

	return mgr, nil
}

// function to handle defers with exit, see https://stackoverflow.com/a/27629493/553720.
func doMain() int {
	log := initializeLogger()

	printVersion(log)

	leaderElectionNamespace, watchNamespace, processDatabaseURL, transportBridgeDatabaseURL, transportType, transportMsgCompressionType, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval, err := readEnvVars()
	if err != nil {
		log.Error(err, "initialization error")
		return 1
	}

	// db layer initialization
	processPostgreSQL, err := postgresql.NewPostgreSQL(processDatabaseURL)
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "process PostgreSQL")
		return 1
	}

	defer processPostgreSQL.Stop()

	transportBridgePostgreSQL, err := postgresql.NewPostgreSQL(transportBridgeDatabaseURL)
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "transport-bridge PostgreSQL")
		return 1
	}

	defer transportBridgePostgreSQL.Stop()

	// transport layer initialization
	transportObj, err := getTransport(transportType, transportMsgCompressionType)
	if err != nil {
		log.Error(err, "transport initialization error")
		return 1
	}

	transportObj.Start()
	defer transportObj.Stop()

	mgr, err := createManager(leaderElectionNamespace, watchNamespace, processPostgreSQL, transportBridgePostgreSQL, transportObj, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval)
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

func main() {
	os.Exit(doMain())
}
