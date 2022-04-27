// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	appsv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addPlacementRuleController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PlacementRule{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("placementrules-spec-syncer"),
			tableName:              "placementrules",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/placementrule-cleanup",
			createInstance:         func() client.Object { return &appsv1.PlacementRule{} },
			cleanStatus:            cleanPlacementRuleStatus,
			areEqual:               arePlacementRulesEqual,
		}); err != nil {
		return fmt.Errorf("failed to add placement rule controller to the manager: %w", err)
	}

	return nil
}

func cleanPlacementRuleStatus(instance client.Object) {
	placementRule, ok := instance.(*appsv1.PlacementRule)

	if !ok {
		panic("wrong instance passed to cleanPlacementRuleStatus: not a PlacementRule")
	}

	placementRule.Status = appsv1.PlacementRuleStatus{}
}

func arePlacementRulesEqual(instance1, instance2 client.Object) bool {
	placementRule1, ok1 := instance1.(*appsv1.PlacementRule)
	placementRule2, ok2 := instance2.(*appsv1.PlacementRule)

	if !ok1 || !ok2 {
		return false
	}

	specMatch := equality.Semantic.DeepEqual(placementRule1.Spec, placementRule2.Spec)
	annotationsMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())
	labelsMatch := equality.Semantic.DeepEqual(instance1.GetLabels(), instance2.GetLabels())

	return specMatch && annotationsMatch && labelsMatch
}
