package load_tracker

import (
	"go.uber.org/fx"

	"github.com/DataDog/datadog-agent/comp/load/load"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
)

// This is the implementation of the load tracker component.
// It is backed by a load tracker instance named "default".
type loadTrackerImpl struct {
	loadTracker *LoadTracker
	enabled     bool
}

type dependencies struct {
	fx.In

	Params load.Params
}

// Start starts the load tracker.
func (l *loadTrackerImpl) Start() {
	l.loadTracker.Start()
}

// stop stops the load tracker.
func (l *loadTrackerImpl) Stop() {
	if l.enabled {
		l.loadTracker.Stop()
	}
}

// AddCollector adds a collector to the load tracker.
func (l *loadTrackerImpl) AddCollector(name string, weight float64, collect func() float64) {
	l.loadTracker.AddCollector(name, collect, weight)
}

// LoadNow returns the current load.
func (l *loadTrackerImpl) LoadNow() float64 {
	return l.loadTracker.LoadNow()
}

// SetWatermarks sets the watermarks.
func (l *loadTrackerImpl) SetWatermarks(lowWatermark, highWatermark float64) {
	l.loadTracker.SetWatermarks(lowWatermark, highWatermark)
}

// IsOverloaded returns true if the load is above the high watermark.
func (l loadTrackerImpl) IsOverloaded() bool {
	return l.loadTracker.IsOverloaded()
}

func (l *loadTrackerImpl) YieldOnOverload() bool {
	return l.loadTracker.YieldOnOverload()
}

func newLoadTracker(deps dependencies) load.Component {
	params := deps.Params
	ret := &loadTrackerImpl{
		loadTracker: NewLoadTrackerWithWatermarks("default", params.ReportingPeriod, params.LowWaterMark, params.HighWaterMark),
		enabled:     deps.Params.Enabled,
	}
	if params.Enabled {
		ret.loadTracker.Start()
	}
	return ret
}

var _ load.Component = (*loadTrackerImpl)(nil)

// Module defines the fx options for this component.
var Module = fxutil.Component(
	fx.Provide(newLoadTracker),
)

// NewServerlessLoadTracker returns a new load tracker for serverless.
func NewServerlessLoadTracker() load.Component {
	return newLoadTracker(dependencies{
		Params: load.Params{
			Enabled:         true,
			ReportingPeriod: 10,
			LowWaterMark:    0.8,
			HighWaterMark:   1.0,
		},
	})
}
