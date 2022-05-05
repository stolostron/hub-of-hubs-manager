package dbsyncer

import (
	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/bundle"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/db"
	datatypes "github.com/stolostron/hub-of-hubs-data-types"
	"github.com/stolostron/hub-of-hubs-data-types/bundle/status"
)

// NewPlacementsDBSyncer creates a new instance of genericDBSyncer to sync placements.
func NewPlacementsDBSyncer(log logr.Logger) DBSyncer {
	dbSyncer := &genericDBSyncer{
		log:              log,
		transportMsgKey:  datatypes.PlacementMsgKey,
		dbSchema:         db.StatusSchema,
		dbTableName:      db.PlacementsTableName,
		createBundleFunc: bundle.NewPlacementsBundle,
		bundlePriority:   conflator.PlacementPriority,
		bundleSyncMode:   status.CompleteStateMode,
	}

	log.Info("initialized placements db syncer")

	return dbSyncer
}
