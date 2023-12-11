package load

import "github.com/DataDog/datadog-agent/comp/load/load"

// BundleParams defines the parameters for this bundle.
type BundleParams struct {
	LoadTrackerParams
}

// LoadTrackerParams defines the parameters of the load tracking component
type LoadTrackerParams = load.Params
