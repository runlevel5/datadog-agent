package load

import (
	"runtime/metrics"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go/log"
)

// pollFrequency is the frequency at which we poll runtime/metrics.
//
// Our goal is to use a window that's smaller than the agent's window so that
// we're less likely to have an empty window followed by a window that includes
// counts from the previous period. This could cause spikes/noise when zooming
// into metrics in a graph. The current frequency of 1s is chosen to be 10x
// smaller than the agent's window. However, in practice, we'll end up with 2s
// due to the statsd client's aggregation.
//
// [1] https://github.com/DataDog/datadog-go/blob/e612112c8bb396b33ad5d9edd645d289b07d0e40/statsd/options.go/#L23
// [2] https://docs.datadoghq.com/developers/dogstatsd/data_aggregation/#how-is-aggregation-performed-with-the-dogstatsd-server
const (
	pollFrequency = 1 * time.Second
)

var (
	runtimeSamples = []metrics.Sample{
		{Name: "/gc/pauses:seconds"},             // histogram
		{Name: "/sched/pauses/total/gc:seconds"}, // histogram (go1.22+)
		{Name: "/gc/heap/allocs:bytes"},
		{Name: "/gc/heap/frees:bytes"},
		{Name: "/sched/gomaxprocs:threads"},
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/sched/latencies:seconds"}, // histogram
	}
)

type goStats struct {
	GCPauses     metrics.Float64Histogram
	GCAllocBytes uint64
	GCFreedBytes uint64
	Gomaxprocs   uint64
	Goroutines   uint64
	SchedLatency metrics.Float64Histogram
}

// See https://www.cockroachlabs.com/blog/rubbing-control-theory/ to understand why goroutine scheduling lantencies are important.

func load() {
	var v goStats

	metrics.Read(runtimeSamples)
	for _, s := range runtimeSamples {
		switch s.Name {
		case "/gc/pauses:seconds":
		case "/sched/pauses/total/gc:seconds":
			v.GCPauses = *s.Value.Float64Histogram()
		case "/gc/heap/allocs:bytes":
			v.GCAllocBytes = s.Value.Uint64()
		case "/gc/heap/frees:bytes":
			v.GCFreedBytes = s.Value.Uint64()
		case "/sched/gomaxprocs:threads":
			v.Gomaxprocs = s.Value.Uint64()
		case "/sched/goroutines:goroutines":
			v.Goroutines = s.Value.Uint64()
		case "/sched/latencies:seconds":
			v.SchedLatency = *s.Value.Float64Histogram()
		}
	}
}

type runtimeMetrics struct {
	sync.Mutex

	ticker  *time.Ticker
	metrics map[string]*runtimeMetric
}

type runtimeMetric struct {
	cumulative bool

	currentValue  metrics.Value
	previousValue metrics.Value
}

func newRuntimeMetricStore(descs []metrics.Description) *runtimeMetrics {
	rms := &runtimeMetrics{
		ticker:  time.NewTicker(pollFrequency),
		metrics: map[string]*runtimeMetric{},
	}

	for _, d := range descs {
		cumulative := d.Cumulative

		// /sched/latencies:seconds is incorrectly set as non-cumulative,
		// fixed by https://go-review.googlesource.com/c/go/+/486755
		// TODO: Use a build tag to apply this logic to Go versions < 1.20.
		if d.Name == "/sched/latencies:seconds" {
			cumulative = true
		}

		rms.metrics[d.Name] = &runtimeMetric{
			cumulative: cumulative,
		}
	}

	// update once to always start with at least one currentValue for each metric
	rms.update()

	return rms
}

func (rms *runtimeMetrics) update() {
	// TODO: Reuse this slice to avoid allocations? Note: I don't see these
	// allocs show up in profiling.
	samples := make([]metrics.Sample, len(rms.metrics))
	i := 0
	// NOTE: Map iteration in Go is randomized, so we end up randomizing the
	// samples slice. In theory this should not impact correctness, but it's
	// worth keeping in mind in case problems are observed in the future.
	for name := range rms.metrics {
		samples[i].Name = name
		i++
	}
	metrics.Read(samples)
	for _, s := range samples {
		runtimeMetric := rms.metrics[s.Name]

		runtimeMetric.previousValue = runtimeMetric.currentValue
		runtimeMetric.currentValue = s.Value
	}
}

func (rms *runtimeMetrics) report() {
	rms.update()
	for name, rm := range rms.metrics {
		switch rm.currentValue.Kind() {
		case metrics.KindUint64:
			v := rm.currentValue.Uint64()
			if rm.cumulative {
				v -= rm.previousValue.Uint64()
			} else {
			}
		case metrics.KindFloat64:
			v := rm.currentValue.Float64()
			if rm.cumulative {
				// Note: This branch should ALWAYS be taken as of go1.21.
				v -= rm.previousValue.Float64()
				if v == 0 {
					continue
				}
			}
		case metrics.KindFloat64Histogram:
			v := rm.currentValue.Float64Histogram()
			var equal bool
			if rm.cumulative {
				// Note: This branch should ALWAYS be taken as of go1.21.
				v, equal = sub(v, rm.previousValue.Float64Histogram())
				// if the histogram didn't change between two reporting
				// cycles, don't submit anything. this avoids having
				// inaccurate drops to zero for percentile metrics
				if equal {
					continue
				}
			}
			stats := statsFromHist(v)
			_ = stats
		case metrics.KindBad:
			// with a once
			// This should never happen because all metrics are supported
			// by construction.
			log.Error("runtimemetrics: encountered an unknown metric, this should never happen and might indicate a bug",
				log.String("metric_name", name))
		default:
			// This may happen as new metric kinds get added.
			//
			// The safest thing to do here is to simply log it somewhere once
			// as something to look into, but ignore it for now.
			once.Do(func() {
				log.Error("runtimemetrics: unsupported metric kind, support for that kind should be added in pkg/runtimemetrics",
					log.String("metric_name", name),
					log.Object("kind", rm.currentValue.Kind()))
			})
		}
	}
}
