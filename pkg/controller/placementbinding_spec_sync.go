// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addPlacementBindingController(mgr ctrl.Manager, databaseConnectionPool *pgxpool.Pool) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&policiesv1.PlacementBinding{}).
		Complete(&genericSpecToDBReconciler{
			client:                 mgr.GetClient(),
			databaseConnectionPool: databaseConnectionPool,
			log:                    ctrl.Log.WithName("placementbindings-spec-syncer"),
			tableName:              "placementbindings",
			finalizerName:          "hub-of-hubs.open-cluster-management.io/placementbinding-cleanup",
			createInstance:         func() client.Object { return &policiesv1.PlacementBinding{} },
			cleanStatus:            cleanPlacementBindingStatus,
			areEqual:               arePlacementBindingsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add placement binding controller to the manager: %w", err)
	}

	return nil
}

func cleanPlacementBindingStatus(instance client.Object) {
	placementBinding, ok := instance.(*policiesv1.PlacementBinding)

	if !ok {
		panic("wrong instance passed to cleanPlacementBindingStatus: not a PlacementBinding")
	}

	placementBinding.Status = policiesv1.PlacementBindingStatus{}
}

func arePlacementBindingsEqual(instance1, instance2 client.Object) bool {
	placementBinding1, ok1 := instance1.(*policiesv1.PlacementBinding)
	placementBinding2, ok2 := instance2.(*policiesv1.PlacementBinding)

	if !ok1 || !ok2 {
		return false
	}

	placementRefMatch := equality.Semantic.DeepEqual(placementBinding1.PlacementRef, placementBinding2.PlacementRef)
	subjectsMatch := equality.Semantic.DeepEqual(placementBinding1.Subjects, placementBinding2.Subjects)
	annotationsMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())
	labelsMatch := equality.Semantic.DeepEqual(instance1.GetLabels(), instance2.GetLabels())

	return placementRefMatch && subjectsMatch && annotationsMatch && labelsMatch
}
