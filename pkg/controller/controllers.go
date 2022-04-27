// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	policiesv1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"
	placementrulesv1 "github.com/open-cluster-management/multicloud-operators-placementrule/pkg/apis/apps/v1"
	configv1 "github.com/stolostron/hub-of-hubs-data-types/apis/config/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	channelsv1 "open-cluster-management.io/multicloud-operators-channel/pkg/apis/apps/v1"
	subscriptionsv1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/v1"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// AddToScheme adds all the resources to be processed to the Scheme.
func AddToScheme(scheme *runtime.Scheme) error {
	for _, schemeBuilder := range getSchemeBuilders() {
		if err := schemeBuilder.AddToScheme(scheme); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}

	// install cluster v1alpha1 / v1beta1 schemes
	if err := clusterv1beta1.Install(scheme); err != nil {
		return fmt.Errorf("failed to install scheme: %w", err)
	}

	if err := clusterv1alpha1.Install(scheme); err != nil {
		return fmt.Errorf("failed to install scheme: %w", err)
	}

	return nil
}

func getSchemeBuilders() []*scheme.Builder {
	return []*scheme.Builder{
		policiesv1.SchemeBuilder, placementrulesv1.SchemeBuilder, configv1.SchemeBuilder,
		applicationv1beta1.SchemeBuilder, channelsv1.SchemeBuilder, subscriptionsv1.SchemeBuilder,
	}
}

// AddControllers adds all the controllers to the Manager.
func AddControllers(mgr ctrl.Manager, dbConnectionPool *pgxpool.Pool) error {
	addControllerFunctions := []func(ctrl.Manager, *pgxpool.Pool) error{
		addPolicyController, addPlacementRuleController,
		addPlacementBindingController, addHubOfHubsConfigController, addApplicationController,
		addSubscriptionController, addChannelController,
		addManagedClusterSetController, addManagedClusterSetBindingController, addPlacementController,
	}

	for _, addControllerFunction := range addControllerFunctions {
		if err := addControllerFunction(mgr, dbConnectionPool); err != nil {
			return fmt.Errorf("failed to add controller: %w", err)
		}
	}

	return nil
}
