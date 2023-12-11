package load_tracker

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/load/load"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// MockLoadTracker is a mock of the load tracker Component useful for testing
type MockLoadTracker struct {
	*loadTrackerImpl
}

// SetLoadFunc sets the load function
//
// It does so by injecting a fake load collector into the load tracker and removing
// all the others.
func (m *MockLoadTracker) SetLoadFunc(load func() float64) {
	m.loadTrackerImpl.loadTracker.Clear()
	m.loadTrackerImpl.loadTracker.AddCollector("mock", load, 1)
}

var _ load.Component = (*MockLoadTracker)(nil)

// NewMock returns a MockSecretResolver
func NewMock(deps dependencies) load.Mock {
	ret := &MockLoadTracker{
		loadTrackerImpl: newLoadTracker(deps).(*loadTrackerImpl),
	}
	ret.loadTrackerImpl.loadTracker.Clear()
	return ret
}

// MockModule is a module containing the mock, useful for testing
var MockModule = fxutil.Component(
	fx.Provide(NewMock),
)
