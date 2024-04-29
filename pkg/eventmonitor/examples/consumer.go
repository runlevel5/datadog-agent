// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package examples

import (
	"sync"

	"github.com/DataDog/datadog-agent/pkg/eventmonitor"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// SimpleEvent defines a simple event
type SimpleEvent struct {
	Type model.EventType
}

// SimpleEventConsumer defines a simple event consumer
type SimpleEventConsumer struct {
	sync.RWMutex
	exec int
	fork int
	exit int
}

// NewSimpleEventConsumer returns a new simple event consumer
func NewSimpleEventConsumer(em *eventmonitor.EventMonitor) *SimpleEventConsumer {
	fc := &SimpleEventConsumer{}
	_ = em.AddEventConsumer(fc)
	return fc
}

// ID returns the ID of this consumer
// Implement the consumer interface
func (fc *SimpleEventConsumer) ID() string {
	return "SIMPLE_CONSUMER"
}

// Start the consumer
// Implement the consumer interface
func (fc *SimpleEventConsumer) Start() error {
	return nil
}

// Stop the consumer
// Implement the consumer interface
func (fc *SimpleEventConsumer) Stop() {
}

// EventTypes returns the event types handled by this consumer
// Implement the consumer interface
func (fc *SimpleEventConsumer) EventTypes() []model.EventType {
	return []model.EventType{
		model.ForkEventType,
		model.ExecEventType,
		model.ExitEventType,
	}
}

// IsReady specifies is the consumer is ready to consume event
func (fc *SimpleEventConsumer) IsReady() bool {
	return true
}

// ChanSize returns the chan size used by the consumer
func (fc *SimpleEventConsumer) ChanSize() int {
	return 50
}

// HandleEvent handles this event
// Implement the consumer interface
func (fc *SimpleEventConsumer) HandleEvent(event any) {
	sevent, ok := event.(*SimpleEvent)
	if !ok {
		log.Error("Event is not a security model event")
		return
	}

	fc.Lock()
	defer fc.Unlock()

	switch sevent.Type {
	case model.ExecEventType:
		fc.exec++
	case model.ForkEventType:
		fc.fork++
	case model.ExitEventType:
		fc.exit++
	}
}

// Copy should copy the given event or return nil to discard it
// Implement the consumer interface
func (fc *SimpleEventConsumer) Copy(event *model.Event) any {
	return &SimpleEvent{
		Type: event.GetEventType(),
	}
}

// ForkCount returns the number of fork handled
func (fc *SimpleEventConsumer) ForkCount() int {
	fc.RLock()
	defer fc.RUnlock()
	return fc.fork
}

// ExitCount returns the number of exit handled
func (fc *SimpleEventConsumer) ExitCount() int {
	fc.RLock()
	defer fc.RUnlock()
	return fc.exit
}

// ExecCount returns the number of exec handled
func (fc *SimpleEventConsumer) ExecCount() int {
	fc.RLock()
	defer fc.RUnlock()
	return fc.exec
}
