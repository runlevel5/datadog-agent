// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

// Package manager contains the manager wrapper
package manager

import (
	"fmt"
	"io"

	manager "github.com/DataDog/ebpf-manager"
	"golang.org/x/exp/slices"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Manager wraps ebpf-manager.Manager
// TODO: Do we want to retain this? Doesn't do anything for now, but how likely it is
// that we're going to add some fields here in the future?
type Manager struct {
	*manager.Manager
}

// Options wraps ebpf-manager.Options to add a list of disabled modifiers
type Options struct {
	manager.Options
	DisabledModifiers []string
}

// NewManager creates a Manager
func NewManager(mgr *manager.Manager) *Manager {
	return &Manager{
		Manager: mgr,
	}
}

// Modifier is an interface that can be implemented by a package to
// add functionality to the ebpf.Manager. It exposes a name to identify the modifier,
// and two functions that will be called before and after the ebpf.Manager.InitWithOptions
// call
type Modifier interface {
	// Name returns the name of the modifier. Should be unique, although it's not enforced for now.
	Name() string

	// BeforeInit is called before the ebpf.Manager.InitWithOptions call
	BeforeInit(*Manager, *Options) error

	// AfterInit is called after the ebpf.Manager.InitWithOptions call
	AfterInit(*Manager, *Options) error

	// OnStop is called when the manager is stopped
	OnStop(*Manager) error
}

var modifiers []Modifier

// RegisterModifier registers a Modifier to be run whenever a new manager is
// initialized. This is used to add functionality to the manager, such as telemetry or
// the newline patching
// This should be called on init() functions of packages that want to add a modifier.
func RegisterModifier(mod Modifier) {
	modifiers = append(modifiers, mod)
}

// InitWithOptions is a wrapper around ebpf-manager.Manager.InitWithOptions
func (m *Manager) InitWithOptions(bytecode io.ReaderAt, opts *Options) error {
	for _, mod := range modifiers {
		if slices.Contains(opts.DisabledModifiers, mod.Name()) {
			log.Debugf("Skipping disabled %s manager modifier", mod.Name())
		} else {
			log.Debugf("Running %s manager modifier", mod.Name())
			if err := mod.BeforeInit(m, opts); err != nil {
				return fmt.Errorf("error running %s manager modifier: %w", mod.Name(), err)
			}
		}
	}

	if err := m.Manager.InitWithOptions(bytecode, opts.Options); err != nil {
		return err
	}

	for _, mod := range modifiers {
		if slices.Contains(opts.DisabledModifiers, mod.Name()) {
			log.Debugf("Skipping disabled %s manager modifier", mod.Name())
		} else {
			log.Debugf("Running %s manager modifier", mod.Name())
			if err := mod.AfterInit(m, opts); err != nil {
				return fmt.Errorf("error running %s manager modifier: %w", mod.Name(), err)
			}
		}
	}
	return nil
}
