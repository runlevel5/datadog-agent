package load

import (
	"testing"

	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

func TestBundleDependencies(t *testing.T) {
	fxutil.TestBundle(t, Bundle, fx.Supply(BundleParams{}))
}

func TestMockBundleDependencies(t *testing.T) {
	fxutil.TestBundle(t, MockBundle, fx.Supply(BundleParams{}))
}
