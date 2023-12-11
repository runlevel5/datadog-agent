package load_tracker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadReporter(t *testing.T) {
	// Create a new load tracker
	lt := NewLoadTracker("test", 10*time.Millisecond)
	lt.Start()
	defer lt.Stop()

	// Add two collectors
	lt.AddCollector("test1", func() float64 {
		return 1.0
	}, 1.0)
	lt.AddCollector("test2", func() float64 {
		return 2.0
	}, 0.5)

	// Wait for the load tracker to collect a value
	time.Sleep(100 * time.Millisecond)

	// Check the load
	l := lt.LoadNow()
	require.Equal(t, 4./3., l)

	// Output
	t.Logf(lt.AsString())

	// For a telemetry export
	lt.exportLoadAvg()
	require.Equal(t, 1.0, tlmLoad.WithTags(map[string]string{"tracker": "test", "collector": "test1"}).Get())
	require.Equal(t, 2.0, tlmLoad.WithTags(map[string]string{"tracker": "test", "collector": "test2"}).Get())
	require.Equal(t, 1.0, tlmOverload.WithTags(map[string]string{"tracker": "test"}).Get())
}

func TestOverload(t *testing.T) {
	lt := NewLoadTrackerWithWatermarks("test", 10*time.Millisecond, 0.2, 0.3)
	lt.AddCollector("test1", func() float64 {
		return 1.0
	}, 1.0)
	lt.run()

	require.Truef(t, lt.IsOverloaded(), "load should be overloaded")
	lt.YieldOnOverload()
}
