// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/db"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddPlacementController(mgr ctrl.Manager, specDB db.SpecDB) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Placement{}).
		Complete(&genericSpecToDBReconciler{
			client:         mgr.GetClient(),
			specDB:         specDB,
			log:            ctrl.Log.WithName("placements-spec-syncer"),
			tableName:      "placements",
			finalizerName:  "hub-of-hubs.open-cluster-management.io/placement-cleanup",
			createInstance: func() client.Object { return &clusterv1alpha1.Placement{} },
			cleanStatus:    cleanPlacementStatus,
			areEqual:       arePlacementsEqual,
		}); err != nil {
		return fmt.Errorf("failed to add placement controller to the manager: %w", err)
	}

	return nil
}

func cleanPlacementStatus(instance client.Object) {
	placement, ok := instance.(*clusterv1alpha1.Placement)

	if !ok {
		panic("wrong instance passed to cleanPlacementStatus: not a Placement")
	}

	placement.Status = clusterv1alpha1.PlacementStatus{}
}

func arePlacementsEqual(instance1, instance2 client.Object) bool {
	placement1, ok1 := instance1.(*clusterv1alpha1.Placement)
	placement2, ok2 := instance2.(*clusterv1alpha1.Placement)

	if !ok1 || !ok2 {
		return false
	}

	specMatch := equality.Semantic.DeepEqual(placement1.Spec, placement2.Spec)
	annotationsMatch := equality.Semantic.DeepEqual(instance1.GetAnnotations(), instance2.GetAnnotations())
	labelsMatch := equality.Semantic.DeepEqual(instance1.GetLabels(), instance2.GetLabels())

	return specMatch && annotationsMatch && labelsMatch
}
