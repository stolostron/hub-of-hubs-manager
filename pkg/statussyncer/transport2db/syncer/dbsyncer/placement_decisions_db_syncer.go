package dbsyncer

import (
	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/bundle"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/db"
	datatypes "github.com/stolostron/hub-of-hubs-data-types"
	"github.com/stolostron/hub-of-hubs-data-types/bundle/status"
)

// NewPlacementDecisionsDBSyncer creates a new instance of genericDBSyncer to sync placement-decisions.
func NewPlacementDecisionsDBSyncer(log logr.Logger) DBSyncer {
	dbSyncer := &genericDBSyncer{
		log:              log,
		transportMsgKey:  datatypes.PlacementDecisionMsgKey,
		dbSchema:         db.StatusSchema,
		dbTableName:      db.PlacementDecisionsTableName,
		createBundleFunc: bundle.NewPlacementDecisionsBundle,
		bundlePriority:   conflator.PlacementDecisionPriority,
		bundleSyncMode:   status.CompleteStateMode,
	}

	log.Info("initialized placement-decisions db syncer")

	return dbSyncer
}
