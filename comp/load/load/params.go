package load

import "time"

// Params defines the parameters for the config component.
type Params struct {
	// Enabled determines whether the load tracker is enabled.
	Enabled bool
	// ReportingPeriod is the period at which the load tracker will report its load.
	ReportingPeriod time.Duration
	// LowWaterMark is the low watermark for the load tracker.
	LowWaterMark float64
	// HighWaterMark is the high watermark for the load tracker.
	HighWaterMark float64
}

// NewEnabledParams constructs params for an enabled component
func NewEnabledParams() Params {
	return Params{
		Enabled: true,
	}
}

// NewDisabledParams constructs params for a disabled component
func NewDisabledParams() Params {
	return Params{
		Enabled: false,
	}
}
