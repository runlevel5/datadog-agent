// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017-present Datadog, Inc.

//nolint:revive // TODO(AML) Fix revive linter
package corechecks

import (
	"fmt"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/aggregator/sender"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
)

type LongRunningCheck interface {
	check.Check

	GetSender() (sender.Sender, error)
}

// LongRunningCheckWrapper provides a wrapper for long running checks
// that will be used by the collector to handle the check lifecycle.
type LongRunningCheckWrapper struct {
	LongRunningCheck
	running bool
	mutex   sync.Mutex
}

// NewLongRunningCheckWrapper returns a new LongRunningCheckWrapper
func NewLongRunningCheckWrapper(check LongRunningCheck) *LongRunningCheckWrapper {
	return &LongRunningCheckWrapper{LongRunningCheck: check, mutex: sync.Mutex{}}
}

// Run runs the check in a goroutine if it is not already running.
// If the check is already running, it will commit the sender.
func (cw *LongRunningCheckWrapper) Run() error {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.running {
		s, err := cw.LongRunningCheck.GetSender()
		if err != nil {
			return fmt.Errorf("error getting sender: %w", err)
		}
		s.Commit()
		return nil
	}

	cw.running = true
	go func() {
		if err := cw.LongRunningCheck.Run(); err != nil {
			fmt.Printf("Error running check: %v\n", err)
		}
		cw.mutex.Lock()
		cw.running = false
		cw.mutex.Unlock()
	}()

	return nil
}
