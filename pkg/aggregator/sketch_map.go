// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package aggregator

import (
	"github.com/DataDog/datadog-agent/pkg/util/cache"
	"math"

	"github.com/DataDog/datadog-agent/pkg/aggregator/ckey"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/opentelemetry-mapping-go/pkg/quantile"
)

type sketchMapValue struct {
	agent *quantile.Agent
	refs  cache.SmallRetainer
}
type sketchMap map[int64]map[ckey.ContextKey]sketchMapValue

// Len returns the number of sketches stored
func (m sketchMap) Len() int {
	l := 0
	for _, byCtx := range m {
		l += len(byCtx)
	}
	return l
}

// insert v into a sketch for the given (ts, contextKey)
// NOTE: ts is truncated to bucketSize
func (m sketchMap) insert(ts int64, ck ckey.ContextKey, v float64, sampleRate float64, refs cache.InternRetainer) bool {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return false
	}

	m.getOrCreate(ts, ck, refs).Insert(v, sampleRate)
	return true
}

func (m sketchMap) insertInterp(ts int64, ck ckey.ContextKey, lower float64, upper float64, count uint, refs cache.InternRetainer) bool {
	if math.IsInf(lower, 0) || math.IsNaN(lower) {
		return false
	}

	if math.IsInf(upper, 0) || math.IsNaN(upper) {
		return false
	}

	m.getOrCreate(ts, ck, refs).InsertInterpolate(lower, upper, count)
	return true
}

// Note: refs will have all its references Import-ed.
func (m sketchMap) getOrCreate(ts int64, ck ckey.ContextKey, refs cache.InternRetainer) *quantile.Agent {
	// level 1: ts -> ctx
	byCtx, ok := m[ts]
	if !ok {
		byCtx = make(map[ckey.ContextKey]sketchMapValue)
		m[ts] = byCtx
	}

	// level 2: ctx -> sketch
	s, ok := byCtx[ck]
	if !ok {
		s = sketchMapValue{agent: &quantile.Agent{}}
		m[ts][ck] = s
	}

	// Keep references for this context's dependencies.
	entry := m[ts][ck]
	entry.refs.Import(refs)
	m[ts][ck] = entry

	return s.agent
}

// flushBefore calls f for every sketch inserted before beforeTs, removing flushed sketches
// from the map.
func (m sketchMap) flushBefore(beforeTs int64, f func(ckey.ContextKey, metrics.SketchPoint, cache.InternRetainer)) {
	for ts, byCtx := range m {
		if ts >= beforeTs {
			continue
		}

		for ck, as := range byCtx {
			f(ck, metrics.SketchPoint{
				Sketch: as.agent.Finish(),
				Ts:     ts,
			}, &as.refs)
		}

		delete(m, ts)
	}
}
