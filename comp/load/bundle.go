package load

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/load/load"
	"github.com/DataDog/datadog-agent/comp/load/load/load_tracker"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// Bundle defines the fx options for this bundle.
var Bundle = fxutil.Bundle(
	fx.Provide(func(params BundleParams) load.Params { return params.LoadTrackerParams }),
	load_tracker.Module,
)
