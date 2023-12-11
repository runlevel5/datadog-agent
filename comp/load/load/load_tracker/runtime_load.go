package load_tracker

import "time"

type RuntimeMetricsLoadCollector struct {
	runtimeMetrics *runtimeMetrics
	goStats        *goStats
}

func NewRuntimeMetricsLoadCollector() *RuntimeMetricsLoadCollector {
	return &RuntimeMetricsLoadCollector{}
}

func (c *RuntimeMetricsLoadCollector) GoroutineScheduleLoad() float64 {
	return c.goStats.SchedLatencyStats.P99 / 1.0 * time.Millisecond
}
