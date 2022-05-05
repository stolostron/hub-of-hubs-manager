package dbsyncer

import (
	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/bundle"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/db"
	datatypes "github.com/stolostron/hub-of-hubs-data-types"
	"github.com/stolostron/hub-of-hubs-data-types/bundle/status"
)

// NewPlacementRulesDBSyncer creates a new instance of genericDBSyncer to sync placement-rules.
func NewPlacementRulesDBSyncer(log logr.Logger) DBSyncer {
	dbSyncer := &genericDBSyncer{
		log:              log,
		transportMsgKey:  datatypes.PlacementRuleMsgKey,
		dbSchema:         db.StatusSchema,
		dbTableName:      db.PlacementRulesTableName,
		createBundleFunc: bundle.NewPlacementRulesBundle,
		bundlePriority:   conflator.PlacementRulePriority,
		bundleSyncMode:   status.CompleteStateMode,
	}

	log.Info("initialized placement-rules db syncer")

	return dbSyncer
}
