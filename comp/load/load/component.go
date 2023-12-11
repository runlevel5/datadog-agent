package load

// Component is the component type.
type Component interface {
	// Stop stops the component.
	Stop()

	// AddCollector adds a collector.
	AddCollector(name string, weight float64, collect func() float64)

	// LoadNow returns the current load.
	LoadNow() float64

	// SetWatermarks sets the watermarks.
	SetWatermarks(lowWatermark, highWatermark float64)

	// IsOverloaded returns true if the load is overloaded.
	IsOverloaded() bool

	// YieldOnOverload yields if the process is overloaded.
	YieldOnOverload() bool
}
