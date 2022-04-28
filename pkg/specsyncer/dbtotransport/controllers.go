package dbtotransport

import (
	"fmt"
	"time"

	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/db"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/dbtotransport/statuswatcher"
	transportsyncer "github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/dbtotransport/transportsyncer"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/transport"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddDBToTransportSyncers adds the controllers that send info from DB to transport layer to the Manager.
func AddDBToTransportSyncers(mgr ctrl.Manager, specDB db.SpecDB, transportObj transport.Transport,
	specSyncInterval time.Duration,
) error {
	addDBSyncerFunctions := []func(ctrl.Manager, db.SpecDB, transport.Transport, time.Duration) error{
		transportsyncer.AddHoHConfigDBToTransportSyncer,
		transportsyncer.AddPoliciesDBToTransportSyncer,
		transportsyncer.AddPlacementRulesDBToTransportSyncer,
		transportsyncer.AddPlacementBindingsDBToTransportSyncer,
		transportsyncer.AddApplicationsDBToTransportSyncer,
		transportsyncer.AddSubscriptionsDBToTransportSyncer,
		transportsyncer.AddChannelsDBToTransportSyncer,
		transportsyncer.AddManagedClusterLabelsDBToTransportSyncer,
		transportsyncer.AddPlacementsDBToTransportSyncer,
		transportsyncer.AddManagedClusterSetsDBToTransportSyncer,
		transportsyncer.AddManagedClusterSetBindingsDBToTransportSyncer,
	}
	for _, addDBSyncerFunction := range addDBSyncerFunctions {
		if err := addDBSyncerFunction(mgr, specDB, transportObj, specSyncInterval); err != nil {
			return fmt.Errorf("failed to add DB Syncer: %w", err)
		}
	}

	return nil
}

// AddStatusDBWatchers adds the controllers that watch the status DB to update the spec DB to the Manager.
func AddStatusDBWatchers(mgr ctrl.Manager, specDB db.SpecDB, statusDB db.StatusDB, deletedLabelsTrimmingInterval time.Duration) error {
	if err := statuswatcher.AddManagedClusterLabelsStatusWatcher(mgr, specDB, statusDB, deletedLabelsTrimmingInterval); err != nil {
		return fmt.Errorf("failed to add status watcher: %w", err)
	}

	return nil
}
