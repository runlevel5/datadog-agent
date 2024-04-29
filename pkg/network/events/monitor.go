// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:generate go run github.com/DataDog/datadog-agent/pkg/security/generators/event_copy -scope "(h *eventConsumerWrapper)" -pkg events -output event_copy_linux.go Process .

//go:build linux

// Package events handles process events
package events

import (
	"slices"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go4.org/intern"

	sprobe "github.com/DataDog/datadog-agent/pkg/security/probe"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	chanSize = 100
)

var theMonitor atomic.Value
var once sync.Once
var initErr error

// Process is a process
type Process struct {
	Pid         uint32        `copy:"GetProcessPid;event:*"`
	Envs        []string      `copy:"FilterEnvs;event:*;envs:DD_SERVICE,DD_VERSION,DD_ENV"`
	ContainerID *intern.Value `copy:"GetContainerId;event:*;intern:true"`
	ExecTime    time.Time     `copy:"GetProcessExecTime;event:ExecEventType"`
	ForkTime    time.Time     `copy:"GetProcessExecTime;event:ForkEventType"`
	StartTime   int64
	Expiry      int64
}

// Env returns the value of a environment variable
func (p *Process) Env(key string) string {
	for _, e := range p.Envs {
		k, v, _ := strings.Cut(e, "=")
		if k == key {
			return v
		}
	}

	return ""
}

// Init initializes the events package
func Init() error {
	once.Do(func() {
		var m *eventMonitor
		m, initErr = newEventMonitor()
		if initErr == nil {
			theMonitor.Store(m)
		}
	})

	return initErr
}

// Initialized returns true if Init() has been called successfully
func Initialized() bool {
	return theMonitor.Load() != nil
}

//nolint:revive // TODO(NET) Fix revive linter
type ProcessEventHandler interface {
	HandleProcessEvent(*Process)
}

// RegisterHandler registers a handler function for getting process events
func RegisterHandler(handler ProcessEventHandler) {
	m := theMonitor.Load().(*eventMonitor)
	m.RegisterHandler(handler)
}

// UnregisterHandler unregisters a handler function for getting process events
func UnregisterHandler(handler ProcessEventHandler) {
	m := theMonitor.Load().(*eventMonitor)
	m.UnregisterHandler(handler)
}

type eventConsumerWrapper struct{}

func (h *eventConsumerWrapper) HandleEvent(ev any) {
	if ev == nil {
		log.Errorf("Received nil event")
		return
	}

	evProcess, ok := ev.(*Process)
	if !ok {
		log.Errorf("Event is not a process")
		return
	}

	m := theMonitor.Load()
	if m != nil {
		if !evProcess.ExecTime.IsZero() {
			evProcess.StartTime = evProcess.ExecTime.UnixNano()
		} else if !evProcess.ForkTime.IsZero() {
			evProcess.StartTime = evProcess.ForkTime.UnixNano()
		}
		m.(*eventMonitor).HandleEvent(evProcess)
	}
}

// monitor is not initialized, no need to copy the event
// since it will get dropped by the handler anyway
func (h *eventConsumerWrapper) IsReady() bool {
	return theMonitor.Load() != nil
}

// EventTypes returns the event types handled by this consumer
func (h *eventConsumerWrapper) EventTypes() []model.EventType {
	return []model.EventType{
		model.ForkEventType,
		model.ExecEventType,
	}
}

// ChanSize returns the chan size used by this consumer
func (h *eventConsumerWrapper) ChanSize() int {
	return chanSize
}

var _eventConsumerWrapper = &eventConsumerWrapper{}

// Consumer returns an event consumer to handle events from the runtime security module
func Consumer() sprobe.EventConsumerInterface {
	return _eventConsumerWrapper
}

type eventMonitor struct {
	sync.Mutex

	handlers []ProcessEventHandler
}

func newEventMonitor() (*eventMonitor, error) {
	return &eventMonitor{}, nil
}

func (e *eventMonitor) HandleEvent(ev *Process) {
	e.Lock()
	defer e.Unlock()

	for _, h := range e.handlers {
		h.HandleProcessEvent(ev)
	}
}

func (e *eventMonitor) RegisterHandler(handler ProcessEventHandler) {
	if handler == nil {
		return
	}

	e.Lock()
	defer e.Unlock()

	e.handlers = append(e.handlers, handler)
}

func (e *eventMonitor) UnregisterHandler(handler ProcessEventHandler) {
	if handler == nil {
		return
	}

	e.Lock()
	defer e.Unlock()

	if idx := slices.Index(e.handlers, handler); idx >= 0 {
		e.handlers = slices.Delete(e.handlers, idx, idx+1)
	}
}
