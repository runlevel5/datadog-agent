package load

// Mock implements mock-specific methods for the resources component.
//
// Usage:
//
//	fxutil.Test[dependencies](
//	   t,
//	   resources.MockModule,
//	   fx.Replace(resources.MockParams{Data: someData}),
//	)
type Mock interface {
	Component

	// SetLoadFunc allows the caller to overwrite the current load.
	SetLoadFunc(load func() float64)
}
