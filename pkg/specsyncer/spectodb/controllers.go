// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package spectodb

import (
	"fmt"

	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/db"
	"github.com/stolostron/hub-of-hubs-all-in-one/pkg/specsyncer/spectodb/controller"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AddSpecToDBControllers adds all the spectodb controllers to the Manager.
func AddSpecToDBControllers(mgr ctrl.Manager, specDB db.SpecDB) error {
	addControllerFunctions := []func(ctrl.Manager, db.SpecDB) error{
		controller.AddPolicyController,
		controller.AddPlacementRuleController,
		controller.AddPlacementBindingController,
		controller.AddHubOfHubsConfigController,
		controller.AddApplicationController,
		controller.AddSubscriptionController,
		controller.AddChannelController,
		controller.AddManagedClusterSetController,
		controller.AddManagedClusterSetBindingController,
		controller.AddPlacementController,
	}

	for _, addControllerFunction := range addControllerFunctions {
		if err := addControllerFunction(mgr, specDB); err != nil {
			return fmt.Errorf("failed to add controller: %w", err)
		}
	}

	return nil
}
