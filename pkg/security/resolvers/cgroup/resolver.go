// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package cgroup holds cgroup related files
package cgroup

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/simplelru"

	"github.com/DataDog/datadog-agent/pkg/security/common/containerutils"
	cgroupModel "github.com/DataDog/datadog-agent/pkg/security/resolvers/cgroup/model"
	"github.com/DataDog/datadog-agent/pkg/security/resolvers/sbom"
	"github.com/DataDog/datadog-agent/pkg/security/resolvers/tags"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/seclog"
)

// Event defines the cgroup event type
type Event int

const (
	// WorkloadSelectorResolved is used to notify that a new cgroup with a resolved workload selector is ready
	WorkloadSelectorResolved Event = iota
	// CGroupDeleted is used to notify that a cgroup was deleted
	CGroupDeleted
	// CGroupMaxEvent is used cap the event ID
	CGroupMaxEvent
)

const systemdSystemDir = "/lib/systemd/system"

// Listener is used to propagate CGroup events
type Listener func(workload *cgroupModel.CacheEntry)

// Resolver defines a cgroup monitor
type Resolver struct {
	sync.RWMutex
	workloads            *simplelru.LRU[string, *cgroupModel.CacheEntry]
	tagsResolver         tags.Resolver
	sbomResolver         *sbom.Resolver
	workloadsWithoutTags chan *cgroupModel.CacheEntry

	listenersLock sync.Mutex
	listeners     map[Event][]Listener
}

// NewResolver returns a new cgroups monitor
func NewResolver(tagsResolver tags.Resolver, sbomResolver *sbom.Resolver) (*Resolver, error) {
	cr := &Resolver{
		tagsResolver:         tagsResolver,
		sbomResolver:         sbomResolver,
		workloadsWithoutTags: make(chan *cgroupModel.CacheEntry, 100),
		listeners:            make(map[Event][]Listener),
	}
	workloads, err := simplelru.NewLRU(1024, func(key string, value *cgroupModel.CacheEntry) {
		value.CallReleaseCallback()
		value.Deleted.Store(true)

		cr.listenersLock.Lock()
		defer cr.listenersLock.Unlock()
		for _, l := range cr.listeners[CGroupDeleted] {
			l(value)
		}
	})
	if err != nil {
		return nil, err
	}
	cr.workloads = workloads
	return cr, nil
}

// Start starts the goroutine of the SBOM resolver
func (cr *Resolver) Start(ctx context.Context) {
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		delayerTick := time.NewTicker(10 * time.Second)
		defer delayerTick.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-delayerTick.C:
				select {
				case workload := <-cr.workloadsWithoutTags:
					cr.checkTags(workload)
				default:
				}

			}
		}
	}()
}

// RegisterListener registers a CGroup event listener
func (cr *Resolver) RegisterListener(event Event, listener Listener) error {
	if event >= CGroupMaxEvent || event < 0 {
		return fmt.Errorf("invalid Event: %v", event)
	}

	cr.listenersLock.Lock()
	defer cr.listenersLock.Unlock()

	if cr.listeners != nil {
		cr.listeners[event] = append(cr.listeners[event], listener)
	} else {
		return fmt.Errorf("a Listener was inserted before initialization")
	}
	return nil
}

// AddPID associates a container id and a pid which is expected to be the pid 1
func (cr *Resolver) AddPID(process *model.ProcessCacheEntry) {
	cr.Lock()
	defer cr.Unlock()

	entry, exists := cr.workloads.Get(process.ContainerID)
	if exists {
		entry.AddPID(process.Pid)
		return
	}

	var err error
	// create new entry now
	newCGroup, err := cgroupModel.NewCacheEntry(process.ContainerID, process.Pid)
	if err != nil {
		seclog.Errorf("couldn't create new cgroup_resolver cache entry: %v", err)
		return
	}
	newCGroup.Flags = uint64(process.ContainerFlags)
	newCGroup.CreatedAt = uint64(process.ProcessContext.ExecTime.UnixNano())

	// add the new CGroup to the cache
	cr.workloads.Add(process.ContainerID, newCGroup)

	// check the tags of this workload
	cr.checkTags(newCGroup)
}

// checkTags checks if the tags of a workload were properly set
func (cr *Resolver) checkTags(workload *cgroupModel.CacheEntry) {
	// check if the workload tags were found
	if workload.NeedsTagsResolution() {
		// this is a container, try to resolve its tags now
		if err := cr.fetchTags(workload); err != nil || workload.NeedsTagsResolution() {
			// push to the workloadsWithoutTags chan so that its tags can be resolved later
			select {
			case cr.workloadsWithoutTags <- workload:
			default:
			}
			return
		}
	}

	// notify listeners
	cr.listenersLock.Lock()
	defer cr.listenersLock.Unlock()
	for _, l := range cr.listeners[WorkloadSelectorResolved] {
		l(workload)
	}
}

// ResolveTags resolves the tags for the specified cgroup
func (cr *Resolver) ResolveTags(cgroupID string, cgroupFlags uint64) (newTags []string, selector cgroupModel.WorkloadSelector, err error) {
	if cr.sbomResolver != nil && (cgroupFlags&containerutils.CGroupManagerSystemd != 0) {
		systemdService := cgroupID
		serviceVersion := "latest"
		servicePath := filepath.Join(systemdSystemDir, systemdService+".service")
		if pkg := cr.sbomResolver.ResolvePackage("", &model.FileEvent{PathnameStr: servicePath}); pkg != nil {
			serviceVersion = pkg.Version
		}
		selector, err = cgroupModel.NewWorkloadSelector(systemdService, serviceVersion)
		newTags = []string{
			"service:" + cgroupID,
			"version:" + serviceVersion,
		}
	} else {
		if newTags, err = cr.tagsResolver.ResolveWithErr(cgroupID); err != nil {
			return nil, selector, fmt.Errorf("failed to get container tags %s: %w", cgroupID, err)
		}
	}
	return
}

// fetchTags fetches tags for the provided workload
func (cr *Resolver) fetchTags(workload *cgroupModel.CacheEntry) (err error) {
	newTags, selector, err := cr.ResolveTags(workload.ID, workload.Flags)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", workload.ID, err)
	}
	if selector.IsReady() {
		workload.WorkloadSelector = selector
	}
	workload.SetTags(newTags)
	return nil
}

// GetWorkload returns the workload referenced by the provided ID
func (cr *Resolver) GetWorkload(id string) (*cgroupModel.CacheEntry, bool) {
	cr.RLock()
	defer cr.RUnlock()

	return cr.workloads.Get(id)
}

// DelPID removes a PID from the cgroup resolver
func (cr *Resolver) DelPID(pid uint32) {
	cr.Lock()
	defer cr.Unlock()

	for _, id := range cr.workloads.Keys() {
		entry, exists := cr.workloads.Get(id)
		if exists {
			cr.deleteWorkloadPID(pid, entry)
		}
	}
}

// DelPIDWithID removes a PID from the cgroup cache entry referenced by the provided ID
func (cr *Resolver) DelPIDWithID(id string, pid uint32) {
	cr.Lock()
	defer cr.Unlock()

	entry, exists := cr.workloads.Get(id)
	if exists {
		cr.deleteWorkloadPID(pid, entry)
	}
}

// deleteWorkloadPID removes a PID from a workload
func (cr *Resolver) deleteWorkloadPID(pid uint32, workload *cgroupModel.CacheEntry) {
	workload.Lock()
	defer workload.Unlock()

	delete(workload.PIDs, pid)

	// check if the workload should be deleted
	if len(workload.PIDs) <= 0 {
		cr.workloads.Remove(workload.ID)
	}
}

// Len return the number of entries
func (cr *Resolver) Len() int {
	cr.RLock()
	defer cr.RUnlock()

	return cr.workloads.Len()
}
