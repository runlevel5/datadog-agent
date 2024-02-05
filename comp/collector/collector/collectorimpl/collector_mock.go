// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

//go:build test

// Package collectorimpl provides the implementation of the collector component.
package collectorimpl

import (
	"go.uber.org/fx"

	"github.com/stretchr/testify/mock"

	"github.com/DataDog/datadog-agent/comp/collector/collector"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	checkid "github.com/DataDog/datadog-agent/pkg/collector/check/id"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"github.com/DataDog/datadog-agent/pkg/util/optional"
)

// MockModule defines the fx options for the mock component.
func MockModule() fxutil.Module {
	return fxutil.Component(
		fx.Provide(newMock),
		fx.Provide(func(collector collector.Component) optional.Option[collector.Component] {
			return optional.NewOption(collector)
		}),
	)
}

type mockimpl struct {
	collector.Component

	mock.Mock
	checksInfo []check.Info
}

func newMock() collector.Component {
	return &mockimpl{}
}

// Start begins the collector's operation.  The scheduler will not run any checks until this has been called.
func (c *mockimpl) Start() {
	c.Called()
}

// Stop halts any component involved in running a Check
func (c *mockimpl) Stop() {
	c.Called()
}

// RunCheck sends a Check in the execution queue
func (c *mockimpl) RunCheck(inner check.Check) (checkid.ID, error) {
	args := c.Called(inner)
	return args.Get(0).(checkid.ID), args.Error(1)
}

// StopCheck halts a check and remove the instance
func (c *mockimpl) StopCheck(id checkid.ID) error {
	args := c.Called(id)
	return args.Error(0)
}

// MapOverChecks call the callback with the list of checks locked.
func (c *mockimpl) MapOverChecks(cb func([]check.Info)) {
	c.Called(cb)
	cb(c.checksInfo)
}

// GetChecks copies checks
func (c *mockimpl) GetChecks() []check.Check {
	args := c.Called()
	return args.Get(0).([]check.Check)
}

// GetAllInstanceIDs returns the ID's of all instances of a check
func (c *mockimpl) GetAllInstanceIDs(checkName string) []checkid.ID {
	args := c.Called(checkName)
	return args.Get(0).([]checkid.ID)
}

// ReloadAllCheckInstances completely restarts a check with a new configuration
func (c *mockimpl) ReloadAllCheckInstances(name string, newInstances []check.Check) ([]checkid.ID, error) {
	args := c.Called(name, newInstances)
	return args.Get(0).([]checkid.ID), args.Error(1)
}

// AddEventReceiver adds a callback to the collector to be called each time a check is added or removed.
func (c *mockimpl) AddEventReceiver(cb collector.EventReceiver) {
	c.Called(cb)
}
