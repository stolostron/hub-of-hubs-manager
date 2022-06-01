package dbsyncer

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-manager/pkg/bundle/status"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/bundle"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/conflator"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/db"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/helpers"
	"github.com/stolostron/hub-of-hubs-manager/pkg/statussyncer/transport2db/transport"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// genericDBSyncer implements generic status resource db sync business logic.
type genericDBSyncer struct {
	log             logr.Logger
	transportMsgKey string

	dbSchema    string
	dbTableName string

	createBundleFunc func() bundle.Bundle
	bundlePriority   conflator.ConflationPriority
	bundleSyncMode   status.BundleSyncMode
}

// RegisterCreateBundleFunctions registers create bundle functions within the transport instance.
func (syncer *genericDBSyncer) RegisterCreateBundleFunctions(transportInstance transport.Transport) {
	transportInstance.Register(&transport.BundleRegistration{
		MsgID:            syncer.transportMsgKey,
		CreateBundleFunc: syncer.createBundleFunc,
		Predicate:        func() bool { return true }, // always get generic status resources
	})
}

// RegisterBundleHandlerFunctions registers bundle handler functions within the conflation manager.
// handler function need to do "diff" between objects received in the bundle and the objects in db.
// leaf hub sends only the current existing objects, and status transport bridge should understand implicitly which
// objects were deleted.
// therefore, whatever is in the db and cannot be found in the bundle has to be deleted from the db.
// for the objects that appear in both, need to check if something has changed using resourceVersion field comparison
// and if the object was changed, update the db with the current object.
func (syncer *genericDBSyncer) RegisterBundleHandlerFunctions(conflationManager *conflator.ConflationManager) {
	conflationManager.Register(conflator.NewConflationRegistration(
		syncer.bundlePriority,
		syncer.bundleSyncMode,
		helpers.GetBundleType(syncer.createBundleFunc()),
		func(ctx context.Context, bundle bundle.Bundle, dbClient db.StatusTransportBridgeDB) error {
			return syncer.handleResourcesBundle(ctx, bundle, dbClient)
		},
	))
}

func (syncer *genericDBSyncer) handleResourcesBundle(ctx context.Context, bundle bundle.Bundle,
	dbClient db.GenericStatusResourceDB,
) error {
	logBundleHandlingMessage(syncer.log, bundle, startBundleHandlingMessage)
	leafHubName := bundle.GetLeafHubName()

	resourceIdentifierToVersionInfoMapFromDB, err := dbClient.GetResourceIdentifiersToVersionByLeafHub(ctx,
		syncer.dbSchema, syncer.dbTableName, leafHubName)
	if err != nil {
		return fmt.Errorf("failed fetching leaf hub '%s.%s' IDs from db - %w", syncer.dbSchema, syncer.dbTableName, err)
	}

	batchBuilder := dbClient.NewGenericBatchBuilder(syncer.dbSchema, syncer.dbTableName, leafHubName)

	for _, object := range bundle.GetObjects() {
		specificObj, ok := object.(metav1.Object)
		if !ok {
			continue
		}

		resourceNameIdentifier := fmt.Sprintf("%s.%s", specificObj.GetNamespace(), specificObj.GetName())
		resourceVersionInfoFromDB, objExistsInDB := resourceIdentifierToVersionInfoMapFromDB[resourceNameIdentifier]

		if !objExistsInDB { // object not found in the db table
			batchBuilder.Insert(string(specificObj.GetUID()), object)
			continue
		}

		delete(resourceIdentifierToVersionInfoMapFromDB, resourceNameIdentifier)

		if specificObj.GetResourceVersion() == resourceVersionInfoFromDB.Version {
			continue // update object in db only if what we got is a different (newer) version of the resource.
		}

		batchBuilder.Update(resourceVersionInfoFromDB.UID, object)
	}

	// delete objects that in the db but were not sent in the bundle (leaf hub sends only living resources).
	for _, resourceVersionInfoFromDB := range resourceIdentifierToVersionInfoMapFromDB {
		batchBuilder.Delete(resourceVersionInfoFromDB.UID)
	}

	if err := dbClient.SendBatch(ctx, batchBuilder.Build()); err != nil {
		return fmt.Errorf("failed to perform batch - %w", err)
	}

	logBundleHandlingMessage(syncer.log, bundle, finishBundleHandlingMessage)

	return nil
}
