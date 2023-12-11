package load_tracker

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/telemetry"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// LoadTracker tracks the load of the process. It is used to determine when
// the process is overloaded and should stop accepting new work.
//
// We can't rely on CPU usage because the process may be I/O bound or because
// we might not know the number of CPUs available to the process, in particular when
// running in a container.
type LoadTracker struct {
	sync.Mutex

	name          string
	period        time.Duration
	stop          chan struct{}
	wg            sync.WaitGroup
	overload      uint32
	lowWatermark  float64
	highWatermark float64

	collectors map[string]*collector

	tlmLoad telemetry.Gauge
}

type collector struct {
	name      string
	weight    float64
	collector LoadCollector
	loads     []float64
}

const (
	// DefaultLoadPeriod is the default period at which the load tracker
	// collects load values.
	DefaultLoadPeriod = 1 * time.Second
	// DefaultKeptLoadValues is the default number of load values kept by the
	// load tracker.
	DefaultKeptLoadValues = 10
	// DefaultLowWatermark is the default low watermark for the load tracker.
	// When the load is below the low watermark, the process is not overloaded.
	DefaultLowWatermark = 0.8
	// DefaultHighWatermark is the default high watermark for the load tracker.
	// When the load is above the high watermark, the process is overloaded.
	DefaultHighWatermark = 1
)

var (
	tlmLoad     telemetry.Gauge
	tlmOverload telemetry.Gauge
	tlmOnce     sync.Once
)

// LoadCollector is a function that returns a normalized load value.
type LoadCollector func() float64

// NewLoadTracker creates a new load tracker.
func NewLoadTracker(name string, period time.Duration) *LoadTracker {
	tlmOnce.Do(func() {
		tlmLoad = telemetry.NewGauge(
			"load_tracker",
			"load10",
			[]string{"tracker", "collector"},
			"Observed load of the process (average for the last 10 reporting periods).",
		)
		tlmOverload = telemetry.NewGauge(
			"load_tracker",
			"overload",
			[]string{"tracker"},
			"Whether the process is overloaded.",
		)
	})
	return &LoadTracker{
		name:          name,
		period:        period,
		collectors:    make(map[string]*collector),
		stop:          make(chan struct{}),
		lowWatermark:  DefaultLowWatermark,
		highWatermark: DefaultHighWatermark,
	}
}

// NewLoadTrackerWithWatermarks creates a new load tracker with custom watermarks.
func NewLoadTrackerWithWatermarks(name string, period time.Duration, lowWatermark, highWatermark float64) *LoadTracker {
	lt := NewLoadTracker(name, period)
	lt.lowWatermark = lowWatermark
	lt.highWatermark = highWatermark
	return lt
}

// Start starts the load tracker.
func (lt *LoadTracker) Start() {
	lt.wg.Add(1)
	go lt.run()
}

func (lt *LoadTracker) run() {
	tickerTlm := time.NewTicker(DefaultKeptLoadValues * lt.period)
	defer tickerTlm.Stop()
	ticker := time.NewTicker(lt.period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lt.collectLoads()
			ln := lt.LoadNow()
			if ln > lt.highWatermark {
				atomic.StoreUint32(&lt.overload, 1)
			} else if ln < lt.lowWatermark {
				atomic.StoreUint32(&lt.overload, 0)
			}
			break
		case <-tickerTlm.C:
			lt.exportLoadAvg()
			break
		case <-lt.stop:
			lt.wg.Done()
			return
		}
	}
}

func (lt *LoadTracker) collectLoads() {
	lt.Lock()
	defer lt.Unlock()

	for i := range lt.collectors {
		c := lt.collectors[i]
		value := c.collector()
		if len(c.loads) < cap(c.loads) {
			c.loads = append(c.loads, value)
		} else {
			c.loads = append(c.loads[1:], value)
		}
	}
}

func (lt *LoadTracker) AddCollector(name string, collect LoadCollector, weight float64) {
	lt.Lock()
	defer lt.Unlock()

	lt.collectors[name] = &collector{
		name:      name,
		weight:    weight,
		collector: collect,
		loads:     make([]float64, 0, DefaultKeptLoadValues),
	}
}

// Stop stops the load tracker.
func (lt *LoadTracker) Stop() {
	close(lt.stop)
	lt.wg.Wait()
}

// LoadNow reports the load average since the last load tracker tick.
func (lt *LoadTracker) LoadNow() float64 {
	lt.Lock()
	defer lt.Unlock()

	var value float64
	var totalWeight float64
	for i := range lt.collectors {
		c := lt.collectors[i]
		value += c.weight * c.loads[len(c.loads)-1]
		totalWeight += c.weight
	}
	return value / totalWeight
}

// LogLoads logs the state of the load tracker.
func (lt *LoadTracker) LogLoads() {
	log.Infof(lt.AsString())
}

// AsString returns a string representation of the load tracker.
func (lt *LoadTracker) AsString() string {
	ln := lt.LoadNow()

	lt.Lock()
	defer lt.Unlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("load: total: %f", ln))
	for i := range lt.collectors {
		c := lt.collectors[i]
		sb.WriteByte('\n')
		sb.WriteString(fmt.Sprintf("load: %s (%f): %#v", c.name, c.weight, c.loads[len(c.loads)-1]))
	}
	return sb.String()
}

func (lt *LoadTracker) exportLoadAvg() {
	lt.Lock()
	defer lt.Unlock()

	var value float64
	var totalWeight float64
	for i := range lt.collectors {
		value += lt.exportLoadAvgCollector(lt.collectors[i]) * lt.collectors[i].weight
	}
	tlmLoad.Set(value/totalWeight, lt.name, "total")
	tlmOverload.Set(float64(atomic.LoadUint32(&lt.overload)), lt.name)
}

func (lt *LoadTracker) exportLoadAvgCollector(c *collector) float64 {
	var value float64
	for i := range c.loads {
		value += c.loads[i]
	}
	value /= float64(len(c.loads))
	tlmLoad.Set(value, lt.name, c.name)
	return value
}

// IsOverloaded returns whether the process is overloaded.
func (lt *LoadTracker) IsOverloaded() bool {
	return atomic.LoadUint32(&lt.overload) >= uint32(1)
}

func (lt *LoadTracker) YieldOnOverload() bool {
	if lt.IsOverloaded() {
		runtime.Gosched()
		return true
	}
	return false
}

// SetWatermarks sets the watermarks for the load tracker.
func (lt *LoadTracker) SetWatermarks(lw, hw float64) {
	lt.Lock()
	defer lt.Unlock()

	lt.lowWatermark = lw
	lt.highWatermark = hw
}

// Clear clears the load tracker.
func (lt *LoadTracker) Clear() {
	lt.Lock()
	defer lt.Unlock()
	lt.collectors = make(map[string]*collector)
}
