// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/apimachinery/pkg/api/equality"
	channelsv1 "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func addChannelController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&channelsv1.Channel{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return object.GetNamespace() != "open-cluster-management"
		})).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("channels-spec-syncer"),
			tableName:              "channels",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/channel-cleanup",
			createInstance:         func() client.Object { return &channelsv1.Channel{} },
			cleanStatus:            cleanChannelStatus,
			areEqual:               areChannelsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add channel controller to the manager: %w", err)
	}

	return nil
}

func cleanChannelStatus(instance client.Object) {
	channel, ok := instance.(*channelsv1.Channel)
	if !ok {
		panic("wrong instance passed to cleanChannelStatus: not a Channel")
	}

	channel.Status = channelsv1.ChannelStatus{}
}

func areChannelsEqual(instance1, instance2 client.Object) bool {
	channel1, ok1 := instance1.(*channelsv1.Channel)
	channel2, ok2 := instance2.(*channelsv1.Channel)

	if !ok1 || !ok2 {
		return false
	}

	specMatch := equality.Semantic.DeepEqual(channel1.Spec, channel2.Spec)
	annotationsMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())
	labelsMatch := equality.Semantic.DeepEqual(instance1.GetLabels(), instance2.GetLabels())

	return specMatch && annotationsMatch && labelsMatch
}
