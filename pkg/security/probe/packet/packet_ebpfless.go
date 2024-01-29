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

	"go.uber.org/atomic"
	"golang.org/x/net/bpf"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/DataDog/datadog-agent/pkg/security/seclog"
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

const dnsPktFilter = "udp and dst port 53"

type FilterOptions struct {
	DNSEnabled bool
}

func (fo *FilterOptions) Equals(other *FilterOptions) bool {
	return fo.DNSEnabled == other.DNSEnabled
}

type FilterOption func(*FilterOptions)

func WithDNSFilter() FilterOption {
	return func(fo *FilterOptions) {
		fo.DNSEnabled = true
	}
}

// Manager is a manager for packet capture
type Manager struct {
	mu sync.Mutex

	fo        *FilterOptions
	ctx       context.Context
	state     state
	stateChan chan state
	stopChan  chan struct{}

	// stats
	pktCaptured      *atomic.Uint64
	pktBytesCaptured *atomic.Uint64
}

// NewManager returns a new Manager
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		fo:               &FilterOptions{},
		ctx:              ctx,
		state:            stateInit,
		stateChan:        make(chan state),
		stopChan:         make(chan struct{}),
		pktCaptured:      atomic.NewUint64(0),
		pktBytesCaptured: atomic.NewUint64(0),
	}
}

// UpdateFilters updates the packet filter used to packet capture
func (m *Manager) UpdateFilters(opts ...FilterOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newFilterOptions := &FilterOptions{}
	for _, opt := range opts {
		opt(newFilterOptions)
	}

	if m.fo.Equals(newFilterOptions) {
		return nil
	}
	m.fo = newFilterOptions

	m.stop()

	pktFilter := buildPktFilter(m.fo)
	if pktFilter == "" {
		return nil
	}

	tpacket, err := afpacket.NewTPacket(afpacket.OptFrameSize(captureLen))
	if err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	filter, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, captureLen, pktFilter)
	if err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	if err := tpacket.SetBPF(pcapFilterToBpfFilter(filter)); err != nil {
		return fmt.Errorf("failed to update packet filter: %w", err)
	}

	m.start(tpacket)
	return nil
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

				if packet.ApplicationLayer().LayerType().Contains(layers.LayerTypeDNS) {
					dns := packet.ApplicationLayer().(*layers.DNS)
					var questionsSb strings.Builder
					for _, question := range dns.Questions {
						_, _ = questionsSb.WriteString(string(question.Name))
						_, _ = questionsSb.WriteRune(' ')
						_, _ = questionsSb.WriteString(question.Type.String())
						_, _ = questionsSb.WriteRune(' ')
						_, _ = questionsSb.WriteString(question.Class.String())
						_, _ = questionsSb.WriteRune('/')
					}
					seclog.Infof("DNS packet: questions(%s) id(%d)", questionsSb.String(), dns.ID)
				}
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

func buildPktFilter(fo *FilterOptions) string {
	var filters []string
	if fo.DNSEnabled {
		filters = append(filters, dnsPktFilter)
	}
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
