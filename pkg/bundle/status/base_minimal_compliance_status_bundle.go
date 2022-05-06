package status

import v1 "github.com/open-cluster-management/governance-policy-propagator/api/v1"

// MinimalPolicyComplianceStatus holds information for minimal policy compliance status.
type MinimalPolicyComplianceStatus struct {
	PolicyID             string               `json:"policyId"`
	RemediationAction    v1.RemediationAction `json:"remediationAction"`
	NonCompliantClusters int                  `json:"nonCompliantClusters"`
	AppliedClusters      int                  `json:"appliedClusters"`
}

// BaseMinimalComplianceStatusBundle the base struct for minimal compliance status bundle.
type BaseMinimalComplianceStatusBundle struct {
	Objects       []*MinimalPolicyComplianceStatus `json:"objects"`
	LeafHubName   string                           `json:"leafHubName"`
	BundleVersion *BundleVersion                   `json:"bundleVersion"`
}
