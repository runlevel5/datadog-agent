// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package packet holds packet related files
package packet

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"go.uber.org/atomic"
	"golang.org/x/net/bpf"

	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
)

const (
	captureLen = 65536
	netnsIno   = 4026531840
)

type state int

const (
	stateInit state = iota
	stateRunning
	stateStopped
)

// Manager is a manager for packet capture
type Manager struct {
	mu sync.Mutex

	ctx                           context.Context
	flowPidMap                    *ebpf.Map
	state                         state
	stateChan                     chan state
	stopChan                      chan struct{}
	currentPacketFilterExpression string
	eventStub                     *model.Event
	onPacketEvent                 func(*model.Event)

	// stats
	pktCaptured      *atomic.Uint64
	pktBytesCaptured *atomic.Uint64
}

type flowPidKey struct {
	netns uint32
	addr  []byte
	port  []byte
}

func (k *flowPidKey) write(buffer []byte) {
	copy(buffer[0:], k.addr)
	model.ByteOrder.PutUint32(buffer[16:20], k.netns)
	copy(buffer[20:], k.port)
}

func (k *flowPidKey) MarshalBinary() ([]byte, error) {
	if len(k.addr) != 4 && len(k.addr) != 16 {
		return nil, fmt.Errorf("invalid address length: %d", len(k.addr))
	}
	bytes := make([]byte, 24)
	k.write(bytes)
	return bytes, nil
}

// NewManager returns a new Manager
func NewManager(ctx context.Context, flowPidMap *ebpf.Map, eventStub *model.Event, onPacketEvent func(*model.Event)) *Manager {
	eventStub.Type = uint32(model.PacketEventType)
	return &Manager{
		ctx:              ctx,
		flowPidMap:       flowPidMap,
		state:            stateInit,
		stateChan:        make(chan state),
		stopChan:         make(chan struct{}),
		onPacketEvent:    onPacketEvent,
		eventStub:        eventStub,
		pktCaptured:      atomic.NewUint64(0),
		pktBytesCaptured: atomic.NewUint64(0),
	}
}

// UpdatePacketFilter updates the packet filter used for packet capture
func (m *Manager) UpdatePacketFilter(filters []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newFilterExpression := computeGlobalFilter(filters)
	if newFilterExpression == m.currentPacketFilterExpression {
		return nil
	}

	m.stop()

	if newFilterExpression == "" {
		m.currentPacketFilterExpression = newFilterExpression
		return nil
	}

	tpacket, err := afpacket.NewTPacket(afpacket.OptFrameSize(captureLen))
	if err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	filter, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, captureLen, newFilterExpression)
	if err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	if err := tpacket.SetBPF(pcapFilterToBpfFilter(filter)); err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	m.currentPacketFilterExpression = newFilterExpression
	m.start(tpacket)
	return nil
}

// Stop stops the packet capture
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stop()
}

func (m *Manager) resolvePid(netns uint32, addr []byte, port []byte) uint32 {
	key := flowPidKey{
		netns: netns,
		addr:  addr,
		port:  port,
	}
	var pid uint32
	if err := m.flowPidMap.Lookup(&key, &pid); err == nil {
		return pid
	}
	key.addr = make([]byte, len(addr))
	if err := m.flowPidMap.Lookup(&key, &pid); err == nil {
		return pid
	}
	return 0
}

func (m *Manager) start(tpacket *afpacket.TPacket) {
	packetSource := gopacket.NewPacketSource(tpacket, layers.LayerTypeEthernet)
	packetSource.NoCopy = true
	packetChan := packetSource.Packets()
	go func() {
		defer func() {
			m.stateChan <- stateStopped
		}()
		defer tpacket.Close()
		m.stateChan <- stateRunning

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-m.stopChan:
				return
			case packet, ok := <-packetChan:
				if !ok {
					return
				}

				metadata := packet.Metadata()
				if metadata == nil {
					continue
				}

				m.pktCaptured.Inc()
				m.pktBytesCaptured.Add(uint64(metadata.CaptureInfo.Length))

				event := m.eventStub
				event.Timestamp = metadata.CaptureInfo.Timestamp
				event.Packet.Packet = packet

				// TODO: ipv6 pid resolution
				var pid uint32
				networkFlowType := packet.NetworkLayer().NetworkFlow().EndpointType()
				transportFlowType := packet.TransportLayer().TransportFlow().EndpointType()
				if networkFlowType == layers.EndpointIPv4 && (transportFlowType == layers.EndpointUDPPort || transportFlowType == layers.EndpointTCPPort) {
					ipv4Src, ipv4Dst := packet.NetworkLayer().NetworkFlow().Endpoints()
					transportSrc, transportDst := packet.TransportLayer().TransportFlow().Endpoints()
					pid = m.resolvePid(netnsIno, ipv4Src.Raw(), transportSrc.Raw())
					if pid == 0 {
						pid = m.resolvePid(netnsIno, ipv4Dst.Raw(), transportDst.Raw())
					}
				}

				event.PIDContext.Pid = pid
				event.PIDContext.Tid = pid
				entry, _ := event.FieldHandlers.ResolveProcessCacheEntry(event)
				event.ProcessCacheEntry = entry
				event.ProcessContext = &event.ProcessCacheEntry.ProcessContext

				m.onPacketEvent(event)
			}
		}
	}()
	m.state = <-m.stateChan
}

func (m *Manager) stop() {
	if m.state == stateRunning {
		m.stopChan <- struct{}{}
		m.state = <-m.stateChan
	}
}

func computeGlobalFilter(filters []string) string {
	var sb strings.Builder
	for i, filter := range filters {
		sb.WriteRune('(')
		sb.WriteString(filter)
		sb.WriteRune(')')
		if i < len(filters)-1 {
			sb.WriteString(" || ")
		}
	}
	return sb.String()
}

func pcapFilterToBpfFilter(pcapFiler []pcap.BPFInstruction) []bpf.RawInstruction {
	bpfFilter := make([]bpf.RawInstruction, len(pcapFiler))
	for i, p := range pcapFiler {
		bpfFilter[i] = bpf.RawInstruction{
			Op: p.Code,
			Jt: p.Jt,
			Jf: p.Jf,
			K:  p.K,
		}
	}
	return bpfFilter
}
