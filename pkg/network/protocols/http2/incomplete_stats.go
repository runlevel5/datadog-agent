// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

//go:build linux_bpf

package http2

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/ebpf"
	"github.com/DataDog/datadog-agent/pkg/network/config"
	"github.com/DataDog/datadog-agent/pkg/network/protocols/http"
	"github.com/DataDog/datadog-agent/pkg/network/types"
)

const (
	defaultMinAge    = 30 * time.Second
	defaultArraySize = 5
)

// incompleteBuffer is responsible for buffering incomplete transactions
type incompleteBuffer struct {
	data       map[types.ConnectionKey]*txParts
	maxEntries int
	minAgeNano int64
}

type txParts struct {
	requests []http.Transaction
}

func newTXParts(requestCapacity int) *txParts {
	return &txParts{
		requests: make([]http.Transaction, 0, requestCapacity),
	}
}

// NewIncompleteBuffer returns a new incompleteBuffer.
func NewIncompleteBuffer(c *config.Config) http.IncompleteBuffer {
	return &incompleteBuffer{
		data:       make(map[types.ConnectionKey]*txParts),
		maxEntries: c.MaxHTTPStatsBuffered,
		minAgeNano: defaultMinAge.Nanoseconds(),
	}
}

// Add adds a transaction to the buffer.
func (b *incompleteBuffer) Add(tx http.Transaction) {
	connTuple := tx.ConnTuple()
	key := types.ConnectionKey{
		SrcIPHigh: connTuple.SrcIPHigh,
		SrcIPLow:  connTuple.SrcIPLow,
		SrcPort:   connTuple.SrcPort,
	}

	parts, ok := b.data[key]
	if !ok {
		if len(b.data) >= b.maxEntries {
			return
		}

		parts = newTXParts(defaultArraySize)
		b.data[key] = parts
	}

	// copy underlying httpTX value. this is now needed because these objects are
	// now coming directly from pooled perf records
	ebpfTX, ok := tx.(*ebpfTXWrapper)
	if !ok {
		// should never happen
		return
	}

	ebpfTxCopy := new(ebpfTXWrapper)
	*ebpfTxCopy = *ebpfTX
	tx = ebpfTxCopy

	parts.requests = append(parts.requests, tx)
}

// Flush flushes the buffer and returns the joined transactions.
func (b *incompleteBuffer) Flush(time.Time) []http.Transaction {
	var (
		joined     []http.Transaction
		previous   = b.data
		nowUnix, _ = ebpf.NowNanoseconds()
	)

	b.data = make(map[types.ConnectionKey]*txParts)
	for key, parts := range previous {
		// now that we have finished matching requests and responses
		// we check if we should keep orphan requests a little longer
		for i := 0; i < len(parts.requests); i++ {
			if !parts.requests[i].Incomplete() {
				joined = append(joined, parts.requests[i])
			} else if b.shouldKeep(parts.requests[i], nowUnix) {
				if _, ok := b.data[key]; !ok {
					b.data[key] = newTXParts(defaultArraySize)
				}
				b.data[key].requests = append(b.data[key].requests, parts.requests[i])
			}
		}
	}

	return joined
}

func (b *incompleteBuffer) shouldKeep(tx http.Transaction, now int64) bool {
	then := int64(tx.RequestStarted())
	return (now - then) < b.minAgeNano
}
