package dbsyncer

import (
	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/bundle"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/statussyncer/transport2db/db"
	datatypes "github.com/stolostron/hub-of-hubs-data-types"
	"github.com/stolostron/hub-of-hubs-data-types/bundle/status"
)

// NewSubscriptionStatusesDBSyncer creates a new instance of genericDBSyncer to sync subscription-statuses.
func NewSubscriptionStatusesDBSyncer(log logr.Logger) DBSyncer {
	dbSyncer := &genericDBSyncer{
		log:              log,
		transportMsgKey:  datatypes.SubscriptionStatusMsgKey,
		dbSchema:         db.StatusSchema,
		dbTableName:      db.SubscriptionStatusesTableName,
		createBundleFunc: bundle.NewSubscriptionStatusesBundle,
		bundlePriority:   conflator.SubscriptionStatusPriority,
		bundleSyncMode:   status.CompleteStateMode,
	}

	log.Info("initialized subscription-statuses db syncer")

	return dbSyncer
}
