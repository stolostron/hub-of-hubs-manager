package dbsyncer

import (
	"context"
	"fmt"
	"time"

	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/db2transport/bundle"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/db2transport/db"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/db2transport/intervalpolicy"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/db2transport/transport"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	policiesTableName = "policies"
	policiesMsgKey    = "Policies"
)

// AddPoliciesDBToTransportSyncer adds policies db to transport syncer to the manager.
func AddPoliciesDBToTransportSyncer(mgr ctrl.Manager, specDB db.SpecDB, transportObj transport.Transport,
	specSyncInterval time.Duration,
) error {
	createObjFunc := func() metav1.Object { return &policiesv1.Policy{} }
	lastSyncTimestampPtr := &time.Time{}

	if err := mgr.Add(&genericDBToTransportSyncer{
		log:            ctrl.Log.WithName("policies-db-to-transport-syncer"),
		intervalPolicy: intervalpolicy.NewExponentialBackoffPolicy(specSyncInterval),
		syncBundleFunc: func(ctx context.Context) (bool, error) {
			return syncObjectsBundle(ctx, transportObj, policiesMsgKey, specDB, policiesTableName,
				createObjFunc, bundle.NewBaseObjectsBundle, lastSyncTimestampPtr)
		},
	}); err != nil {
		return fmt.Errorf("failed to add policies db to transport syncer - %w", err)
	}

	return nil
}
