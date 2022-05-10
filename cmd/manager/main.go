// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	"github.com/stolostron/hub-of-hubs-manager/pkg/compressor"
	"github.com/stolostron/hub-of-hubs-manager/pkg/nonk8sapi"
	"github.com/stolostron/hub-of-hubs-manager/pkg/scheme"
	"github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/db/postgresql"
	specsyncer "github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/syncer"
	spectransport "github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/transport"
	speckafka "github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/transport/kafka"
	specsyncservice "github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/transport/syncservice"
	"github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/spec2db"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statistics"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/db2status"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/db/workerpool"
	statussyncer "github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/syncer"
	statustransport "github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/transport"
	statuskafka "github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/transport/kafka"
	statussyncservice "github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/transport/syncservice"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	metricsHost                                        = "0.0.0.0"
	metricsPort                                  int32 = 8384
	envVarControllerNamespace                          = "POD_NAMESPACE"
	envVarWatchNamespace                               = "WATCH_NAMESPACE"
	envVarProcessDatabaseURL                           = "PROCESS_DATABASE_URL"
	envVarTransportBridgeDatabaseURL                   = "TRANSPORT_BRIDGE_DATABASE_URL"
	envVarTransportMsgCompressionType                  = "TRANSPORT_MESSAGE_COMPRESSION_TYPE"
	envVarTransportType                                = "TRANSPORT_TYPE"
	envVarSpecSyncInterval                             = "SPEC_SYNC_INTERVAL"
	envVarStatusSyncInterval                           = "STATUS_SYNC_INTERVAL"
	envVarLabelsTrimmingInterval                       = "DELETED_LABELS_TRIMMING_INTERVAL"
	environmentVariableClusterAPIURL                   = "CLUSTER_API_URL"
	environmentVariableClusterAPICABundlePath          = "CLUSTER_API_CA_BUNDLE_PATH"
	environmentVariableAuthorizationURL                = "AUTHORIZATION_URL"
	environmentVariableAuthorizationCABundlePath       = "AUTHORIZATION_CA_BUNDLE_PATH"
	environmentVariableServerCertificatePath           = "SERVER_CERTIFICATE_PATH"
	environmentVariableServerKeyPath                   = "SERVER_KEY_PATH"
	environmentVariableServerBasePath                  = "SERVER_BASE_PATH"
	kafkaTransportTypeName                             = "kafka"
	syncServiceTransportTypeName                       = "sync-service"
	leaderElectionLockName                             = "hub-of-hubs-manager-lock"
)

var (
	errEnvVarNotFound          = errors.New("environment variable not found")
	errEnvVarIllegalValue      = errors.New("environment variable illegal value")
	errFailedToLoadCertificate = errors.New("failed to load certificate/key")
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

func readEnvVars() (string, string, string, string, string, string, string, string, string, string, string, string, string,
	time.Duration, time.Duration, time.Duration, error,
) {
	controllerNamespace, found := os.LookupEnv(envVarControllerNamespace)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarControllerNamespace)
	}

	watchNamespace, found := os.LookupEnv(envVarWatchNamespace)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarWatchNamespace)
	}

	processDatabaseURL, found := os.LookupEnv(envVarProcessDatabaseURL)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarProcessDatabaseURL)
	}

	transportBridgeDatabaseURL, found := os.LookupEnv(envVarTransportBridgeDatabaseURL)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportBridgeDatabaseURL)
	}

	transportType, found := os.LookupEnv(envVarTransportType)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportType)
	}

	transportMsgCompressionType, found := os.LookupEnv(envVarTransportMsgCompressionType)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarTransportMsgCompressionType)
	}

	clusterAPIURL, found := os.LookupEnv(environmentVariableClusterAPIURL)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableClusterAPIURL)
	}

	clusterAPICABundlePath, found := os.LookupEnv(environmentVariableClusterAPICABundlePath)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableClusterAPICABundlePath)
	}

	authorizationURL, found := os.LookupEnv(environmentVariableAuthorizationURL)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableAuthorizationURL)
	}

	authorizationCABundlePath, found := os.LookupEnv(environmentVariableAuthorizationCABundlePath)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableAuthorizationCABundlePath)
	}

	nonK8sAPIServerCertificatePath, found := os.LookupEnv(environmentVariableServerCertificatePath)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableServerCertificatePath)
	}

	nonK8sAPIServerKeyPath, found := os.LookupEnv(environmentVariableServerKeyPath)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableServerKeyPath)
	}

	nonK8sAPIServerBasePath, found := os.LookupEnv(environmentVariableServerBasePath)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, environmentVariableServerBasePath)
	}

	specSyncIntervalString, found := os.LookupEnv(envVarSpecSyncInterval)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarSpecSyncInterval)
	}

	specSyncInterval, err := time.ParseDuration(specSyncIntervalString)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			specSyncInterval, err)
	}

	statusSyncIntervalString, found := os.LookupEnv(envVarStatusSyncInterval)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarStatusSyncInterval)
	}

	statusSyncInterval, err := time.ParseDuration(statusSyncIntervalString)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			statusSyncInterval, err)
	}

	deletedLabelsTrimmingIntervalString, found := os.LookupEnv(envVarLabelsTrimmingInterval)
	if !found {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("%w: %s", errEnvVarNotFound, envVarLabelsTrimmingInterval)
	}

	deletedLabelsTrimmingInterval, err := time.ParseDuration(deletedLabelsTrimmingIntervalString)
	if err != nil {
		return "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0, 0, fmt.Errorf("the environment var %s is not a valid duration - %w",
			deletedLabelsTrimmingInterval, err)
	}

	return controllerNamespace, watchNamespace, processDatabaseURL, transportBridgeDatabaseURL, transportType, transportMsgCompressionType,
		clusterAPIURL, clusterAPICABundlePath, authorizationURL, authorizationCABundlePath, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath,
		nonK8sAPIServerBasePath, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval, nil
}

func readCertificates(clusterAPICABundlePath, authorizationCABundlePath, certificatePath, keyPath string) ([]byte, []byte, tls.Certificate, error) {
	var (
		clusterAPICABundle    []byte
		authorizationCABundle []byte
		certificate           tls.Certificate
	)

	if clusterAPICABundlePath != "" {
		clusterAPICABundle, err := ioutil.ReadFile(clusterAPICABundlePath)
		if err != nil {
			return clusterAPICABundle, authorizationCABundle, certificate,
				fmt.Errorf("%w: %s", errFailedToLoadCertificate, clusterAPICABundlePath)
		}
	}

	if authorizationCABundlePath != "" {
		authorizationCABundle, err := ioutil.ReadFile(authorizationCABundlePath)
		if err != nil {
			return clusterAPICABundle, authorizationCABundle, certificate,
				fmt.Errorf("%w: %s", errFailedToLoadCertificate, authorizationCABundle)
		}
	}

	certificate, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		return clusterAPICABundle, authorizationCABundle, certificate,
			fmt.Errorf("%w: %s/%s", errFailedToLoadCertificate, certificatePath, keyPath)
	}

	return clusterAPICABundle, authorizationCABundle, certificate, nil
}

// function to determine whether the transport component requires initial-dependencies between bundles to be checked
// (on load). If the returned is false, then we may assume that dependency of the initial bundle of
// each type is met. Otherwise, there are no guarantees and the dependencies must be checked.
func requireInitialDependencyChecks(transportType string) bool {
	switch transportType {
	case kafkaTransportTypeName:
		return false
		// once kafka consumer loads up, it starts reading from the earliest un-processed bundle,
		// as in all bundles that precede the latter have been processed, which include its dependency
		// bundle.

		// the order guarantee also guarantees that if while loading this component, a new bundle is written to a-
		// partition, then surely its dependency was written before it (leaf-hub-status-sync on kafka guarantees).
	case syncServiceTransportTypeName:
		fallthrough
	default:
		return true
	}
}

// function to choose spec transport type based on env var.
func getSpecTransport(transportType string, transportMsgCompressorType string) (spectransport.Transport, error) {
	msgCompressor, err := compressor.NewCompressor(compressor.CompressionType(transportMsgCompressorType))
	if err != nil {
		return nil, fmt.Errorf("failed to create message-compressor: %w", err)
	}

	switch transportType {
	case kafkaTransportTypeName:
		kafkaProducer, err := speckafka.NewProducer(msgCompressor, ctrl.Log.WithName("kafka"))
		if err != nil {
			return nil, fmt.Errorf("failed to create kafka-producer: %w", err)
		}

		return kafkaProducer, nil
	case syncServiceTransportTypeName:
		syncService, err := specsyncservice.NewSyncService(msgCompressor, ctrl.Log.WithName("sync-service"))
		if err != nil {
			return nil, fmt.Errorf("failed to create sync-service: %w", err)
		}

		return syncService, nil
	default:
		return nil, fmt.Errorf("%w: %s - %s is not a valid option", errEnvVarIllegalValue, envVarTransportType,
			transportType)
	}
}

// function to choose status transport type based on env var.
func getStatusTransport(transportType string, conflationMgr *conflator.ConflationManager,
	statistics *statistics.Statistics,
) (statustransport.Transport, error) {
	switch transportType {
	case kafkaTransportTypeName:
		kafkaConsumer, err := statuskafka.NewConsumer(ctrl.Log.WithName("kafka"), conflationMgr, statistics)
		if err != nil {
			return nil, fmt.Errorf("failed to create kafka-consumer: %w", err)
		}

		return kafkaConsumer, nil
	case syncServiceTransportTypeName:
		syncService, err := statussyncservice.NewSyncService(ctrl.Log.WithName("sync-service"), conflationMgr, statistics)
		if err != nil {
			return nil, fmt.Errorf("failed to create sync-service: %w", err)
		}

		return syncService, nil
	default:
		return nil, fmt.Errorf("%w: %s - %s is not a valid option", errEnvVarIllegalValue, envVarTransportType,
			transportType)
	}
}

func createManager(controllerNamespace, watchNamespace string, processPostgreSQL, transportBridgePostgreSQL *postgresql.PostgreSQL, workersPool *workerpool.DBWorkerPool,
	specTransportObj spectransport.Transport, statusTransportObj statustransport.Transport, conflationManager *conflator.ConflationManager, conflationReadyQueue *conflator.ConflationReadyQueue,
	statistics *statistics.Statistics, clusterAPIURL, authorizationURL, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath, nonK8sAPIServerBasePath string, clusterAPICABundle, authorizationCABundle []byte,
	specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval time.Duration,
) (ctrl.Manager, error) {
	options := ctrl.Options{
		Namespace:               watchNamespace,
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
		LeaderElection:          true,
		LeaderElectionNamespace: controllerNamespace,
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

	if err := nonk8sapi.AddNonK8sApiServer(mgr, clusterAPIURL, authorizationURL, nonK8sAPIServerBasePath, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath, clusterAPICABundle, authorizationCABundle, processPostgreSQL); err != nil {
		return nil, fmt.Errorf("failed to add non-k8s-api-server: %w", err)
	}

	if err := spec2db.AddSpec2DBControllers(mgr, processPostgreSQL); err != nil {
		return nil, fmt.Errorf("failed to add spec-to-db controllers: %w", err)
	}

	if err := specsyncer.AddDB2TransportSyncers(mgr, transportBridgePostgreSQL, specTransportObj, specSyncInterval); err != nil {
		return nil, fmt.Errorf("failed to add db-to-transport syncers: %w", err)
	}

	if err := specsyncer.AddStatusDBWatchers(mgr, transportBridgePostgreSQL, transportBridgePostgreSQL, deletedLabelsTrimmingInterval); err != nil {
		return nil, fmt.Errorf("failed to add status db watchers: %w", err)
	}

	if err := db2status.AddDBSyncers(mgr, processPostgreSQL, statusSyncInterval); err != nil {
		return nil, fmt.Errorf("failed to add status db syncers: %w", err)
	}

	if err := statussyncer.AddTransport2DBSyncers(mgr, workersPool, conflationManager, conflationReadyQueue, statusTransportObj, statistics); err != nil {
		return nil, fmt.Errorf("failed to add transport-to-db syncers: %w", err)
	}

	return mgr, nil
}

// function to handle defers with exit, see https://stackoverflow.com/a/27629493/553720.
func doMain() int {
	log := initializeLogger()

	printVersion(log)

	controllerNamespace, watchNamespace, processDatabaseURL, transportBridgeDatabaseURL, transportType, transportMsgCompressionType, clusterAPIURL, clusterAPICABundlePath, authorizationURL, authorizationCABundlePath, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath, nonK8sAPIServerBasePath, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval, err := readEnvVars()
	if err != nil {
		log.Error(err, "initialization error")
		return 1
	}

	clusterAPICABundle, authorizationCABundle, _, err := readCertificates(clusterAPICABundlePath, authorizationCABundlePath, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath)
	if err != nil {
		log.Error(err, "failed to read certificates")
		return 1
	}

	// create statistics
	stats, err := statistics.NewStatistics(ctrl.Log.WithName("statistics"))
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "statistics")
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

	// db layer initialization - worker pool + connection pool
	dbWorkerPool, err := workerpool.NewDBWorkerPool(ctrl.Log.WithName("db-worker-pool"), transportBridgeDatabaseURL, stats)
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "DBWorkerPool")
		return 1
	}

	if err = dbWorkerPool.Start(); err != nil {
		log.Error(err, "initialization error", "failed to start", "DBWorkerPool")
		return 1
	}
	defer dbWorkerPool.Stop()

	// conflationReadyQueue is shared between conflation manager and dispatcher
	conflationReadyQueue := conflator.NewConflationReadyQueue(stats)
	requireInitialDependencyChecks := requireInitialDependencyChecks(transportType)
	conflationManager := conflator.NewConflationManager(ctrl.Log.WithName("conflation"), conflationReadyQueue,
		requireInitialDependencyChecks, stats) // manage all Conflation Units

	// status transport layer initialization
	statusTransportObj, err := getStatusTransport(transportType, conflationManager, stats)
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "status transport")
		return 1
	}

	statusTransportObj.Start()
	defer statusTransportObj.Stop()

	// spec transport layer initialization
	specTransportObj, err := getSpecTransport(transportType, transportMsgCompressionType)
	if err != nil {
		log.Error(err, "initialization error", "failed to initialize", "spec transport")
		return 1
	}

	specTransportObj.Start()
	defer specTransportObj.Stop()

	mgr, err := createManager(controllerNamespace, watchNamespace, processPostgreSQL, transportBridgePostgreSQL, dbWorkerPool, specTransportObj, statusTransportObj,
		conflationManager, conflationReadyQueue, stats, clusterAPIURL, authorizationURL, nonK8sAPIServerCertificatePath, nonK8sAPIServerKeyPath, nonK8sAPIServerBasePath,
		clusterAPICABundle, authorizationCABundle, specSyncInterval, statusSyncInterval, deletedLabelsTrimmingInterval)
	if err != nil {
		log.Error(err, "failed to create manager")
		return 1
	}

	log.Info("starting the Cmd.")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited non-zero")
		return 1
	}

	return 0
}

func main() {
	os.Exit(doMain())
}
