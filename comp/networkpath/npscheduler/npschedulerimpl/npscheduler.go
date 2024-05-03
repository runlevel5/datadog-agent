// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024-present Datadog, Inc.

// Package npschedulerimpl implements the scheduler for network path
package npschedulerimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/comp/core/config"
	"github.com/DataDog/datadog-agent/comp/core/log"
	"github.com/DataDog/datadog-agent/comp/forwarder/eventplatform"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/DataDog/datadog-agent/pkg/network/bogon"
	"github.com/DataDog/datadog-agent/pkg/networkdevice/utils"
	"github.com/DataDog/datadog-agent/pkg/networkpath/traceroute"
	"github.com/DataDog/datadog-agent/pkg/process/statsd"
	"github.com/DataDog/datadog-agent/pkg/util/hostname"
	"go.uber.org/atomic"
)

type npSchedulerImpl struct {
	epForwarder eventplatform.Component
	logger      log.Component

	workers int

	excludeIPManager *bogon.Bogon

	receivedPathtestConfigCount *atomic.Uint64
	pathtestStore               *pathtestStore
	pathtestInputChan           chan *pathtest
	pathtestProcessChan         chan *pathtestContext
	stopChan                    chan struct{}
	flushLoopDone               chan struct{}
	runDone                     chan struct{}

	TimeNowFunction  func() time.Time // Allows to mock time in tests
	enabled          bool
	tracerouteRunner *traceroute.Runner
}

func newNoopNpSchedulerImpl() *npSchedulerImpl {
	return &npSchedulerImpl{}
}

func newNpSchedulerImpl(epForwarder eventplatform.Component, logger log.Component, sysprobeYamlConfig config.Reader, params Params) *npSchedulerImpl {
	workers := sysprobeYamlConfig.GetInt("network_path.workers")
	pathtestInputChanSize := sysprobeYamlConfig.GetInt("network_path.input_chan_size")
	pathtestProcessChanSize := sysprobeYamlConfig.GetInt("network_path.process_chan_size")
	pathtestTTL := sysprobeYamlConfig.GetDuration("network_path.pathtest_ttl")
	pathtestInterval := sysprobeYamlConfig.GetDuration("network_path.pathtest_interval")
	excludeCIDR := sysprobeYamlConfig.GetStringSlice("network_path.exclude_cidr")

	logger.Infof("New NpScheduler (workers=%d input_chan_size=%d pathtest_ttl=%s pathtest_interval=%s exclude_cidr=%v)",
		workers,
		pathtestInputChanSize,
		pathtestTTL.String(),
		pathtestInterval.String(),
		excludeCIDR)

	var tracerouteRunner *traceroute.Runner
	if params.TracerouteRunner == SimpleTraceroute {
		runner, err := traceroute.NewRunner()
		if err != nil {
			logger.Errorf("Unable to create traceroute Runner: %s", err)
		} else {
			tracerouteRunner = runner
		}
	}

	var excludeIPManager *bogon.Bogon
	if len(excludeCIDR) >= 1 {
		newExcludeIPManager, err := bogon.New(excludeCIDR)
		if err != nil {
			logger.Errorf("Invalid network_path.exclude_cidr: %s", err)
		} else {
			excludeIPManager = newExcludeIPManager
		}
	}

	if excludeIPManager == nil {
		excludeIPManager = &bogon.Bogon{}
	}

	return &npSchedulerImpl{
		enabled:          true,
		tracerouteRunner: tracerouteRunner,
		epForwarder:      epForwarder,
		logger:           logger,

		pathtestStore:       newPathtestStore(DefaultFlushTickerInterval, pathtestTTL, pathtestInterval, logger),
		pathtestInputChan:   make(chan *pathtest, pathtestInputChanSize),
		pathtestProcessChan: make(chan *pathtestContext, pathtestProcessChanSize),
		workers:             workers,
		excludeIPManager:    excludeIPManager,

		receivedPathtestConfigCount: atomic.NewUint64(0),
		TimeNowFunction:             time.Now,

		stopChan:      make(chan struct{}),
		runDone:       make(chan struct{}),
		flushLoopDone: make(chan struct{}),
	}
}

func (s *npSchedulerImpl) listenPathtestConfigs() {
	for {
		select {
		case <-s.stopChan:
			// TODO: TESTME
			s.logger.Info("Stop listening to traceroute commands")
			s.runDone <- struct{}{}
			return
		case ptest := <-s.pathtestInputChan:
			// TODO: TESTME
			s.logger.Debugf("Pathtest received: %+v", ptest)
			s.receivedPathtestConfigCount.Inc()
			s.pathtestStore.add(ptest)
		}
	}
}

// Schedule schedules pathtests.
// It shouldn't block, if the input channel is full, an error is returned.
func (s *npSchedulerImpl) Schedule(hostname string, port uint16) error {
	if s.pathtestInputChan == nil {
		return errors.New("no input channel, please check that network path is enabled")
	}
	s.logger.Debugf("Schedule traceroute for: hostname=%s port=%d", hostname, port)

	if net.ParseIP(hostname).To4() == nil {
		// TODO: IPv6 not supported yet
		s.logger.Debugf("Only IPv4 is currently supported. Address not supported: %s", hostname)
		return nil
	}

	if s.excludeIPManager != nil {
		isExcluded, _ := s.excludeIPManager.Is(hostname)
		if isExcluded {
			s.logger.Debugf("Excluded IP hostname=%s", hostname)
			return nil
		}
	}

	ptest := &pathtest{
		hostname: hostname,
		port:     port,
	}
	select {
	case s.pathtestInputChan <- ptest:
		return nil
	default:
		return fmt.Errorf("scheduler input channel is full (channel capacity is %d)", cap(s.pathtestInputChan))
	}
}

func (s *npSchedulerImpl) Enabled() bool {
	return s.enabled
}

func (s *npSchedulerImpl) runTraceroute(ptest *pathtestContext) {
	s.logger.Debugf("Run Traceroute for ptest: %+v", ptest)
	s.pathForConn(ptest)
}

func (s *npSchedulerImpl) pathForConn(ptest *pathtestContext) {
	startTime := time.Now()
	cfg := traceroute.Config{
		DestHostname: ptest.pathtest.hostname,
		DestPort:     uint16(ptest.pathtest.port),
		MaxTTL:       24,
		TimeoutMs:    1000,
	}

	var path traceroute.NetworkPath
	if s.tracerouteRunner != nil {
		s.logger.Debugf("Run Simple Traceroute for %v", cfg)
		newPath, err := s.tracerouteRunner.RunTraceroute(context.TODO(), cfg)
		if err != nil {
			s.logger.Warnf("traceroute error: %+v", err)
			return
		}
		path = newPath
	} else {
		s.logger.Debugf("Run Classic Traceroute for %v", cfg)
		tr, err := traceroute.New(cfg)
		if err != nil {
			s.logger.Warnf("traceroute error: %+v", err)
			return
		}
		newPath, err := tr.Run(context.TODO())
		if err != nil {
			s.logger.Warnf("traceroute error: %+v", err)
			return
		}
		path = newPath
	}
	s.logger.Debugf("Network Path: %+v", path)

	s.sendTelemetry(path, startTime, ptest)

	epForwarder, ok := s.epForwarder.Get()
	if ok {
		payloadBytes, err := json.Marshal(path)
		if err != nil {
			s.logger.Errorf("json marshall error: %s", err)
		} else {

			s.logger.Debugf("network path event: %s", string(payloadBytes))
			m := message.NewMessage(payloadBytes, nil, "", 0)
			err = epForwarder.SendEventPlatformEvent(m, eventplatform.EventTypeNetworkPath)
			if err != nil {
				s.logger.Errorf("SendEventPlatformEvent error: %s", err)
			}
		}
	}
}

func (s *npSchedulerImpl) Start() {
	s.logger.Info("Start NpScheduler")
	go s.listenPathtestConfigs()
	go s.flushLoop()
	s.startWorkers()
}

func (s *npSchedulerImpl) Stop() {
	s.logger.Infof("Stop NpScheduler")
	close(s.stopChan)
	<-s.flushLoopDone
	<-s.runDone
}

func (s *npSchedulerImpl) flushLoop() {
	flushTicker := time.NewTicker(10 * time.Second)

	var lastFlushTime time.Time
	for {
		select {
		// stop sequence
		case <-s.stopChan:
			s.flushLoopDone <- struct{}{}
			flushTicker.Stop()
			return
		// automatic flush sequence
		case <-flushTicker.C:
			now := time.Now()
			if !lastFlushTime.IsZero() {
				flushInterval := now.Sub(lastFlushTime)
				statsd.Client.Gauge("datadog.network_path.scheduler.flush_interval", flushInterval.Seconds(), []string{}, 1) //nolint:errcheck
			}
			lastFlushTime = now

			flushStartTime := time.Now()
			s.flush()
			statsd.Client.Gauge("datadog.network_path.scheduler.flush_duration", time.Since(flushStartTime).Seconds(), []string{}, 1) //nolint:errcheck
		}
	}
}

func (s *npSchedulerImpl) flush() {
	// TODO: Remove workers metric?
	statsd.Client.Gauge("datadog.network_path.scheduler.workers", float64(s.workers), []string{}, 1) //nolint:errcheck

	flowsContexts := s.pathtestStore.getPathtestContextCount()
	statsd.Client.Gauge("datadog.network_path.scheduler.pathtest_store_size", float64(flowsContexts), []string{}, 1) //nolint:errcheck
	flushTime := s.TimeNowFunction()
	flowsToFlush := s.pathtestStore.flush()
	statsd.Client.Gauge("datadog.network_path.scheduler.pathtest_flushed_count", float64(len(flowsToFlush)), []string{}, 1) //nolint:errcheck
	s.logger.Debugf("Flushing %d flows to the forwarder (flush_duration=%d, flow_contexts_before_flush=%d)", len(flowsToFlush), time.Since(flushTime).Milliseconds(), flowsContexts)

	for _, ptConf := range flowsToFlush {
		s.logger.Tracef("flushed ptConf %s:%d", ptConf.pathtest.hostname, ptConf.pathtest.port)
		// TODO: FLUSH TO CHANNEL + WORKERS EXECUTE
		s.pathtestProcessChan <- ptConf
	}
}

func (s *npSchedulerImpl) sendTelemetry(path traceroute.NetworkPath, startTime time.Time, ptest *pathtestContext) {
	// TODO: Factor Network Path telemetry from Network Path Integration and use the code
	// TODO: Factor Network Path telemetry from Network Path Integration and use the code
	// TODO: Factor Network Path telemetry from Network Path Integration and use the code
	// TODO: Factor Network Path telemetry from Network Path Integration and use the code

	// TODO: Add collector type tag (np_scheduler | network_path_integration)
	tags := s.getTelemetryTags(path, ptest)
	tags = append(tags, "pathtest_source:network_path_scheduler")

	checkDuration := time.Since(startTime)
	statsd.Client.Gauge("datadog.network_path.check_duration", checkDuration.Seconds(), tags, 1) //nolint:errcheck

	if ptest.lastFlushInterval > 0 {
		statsd.Client.Gauge("datadog.network_path.check_interval", ptest.lastFlushInterval.Seconds(), tags, 1) //nolint:errcheck
	}

	statsd.Client.Gauge("datadog.network_path.path.monitored", float64(1), tags, 1) //nolint:errcheck
	if len(path.Hops) > 0 {
		lastHop := path.Hops[len(path.Hops)-1]
		if lastHop.Success {
			statsd.Client.Gauge("datadog.network_path.path.hops", float64(len(path.Hops)), tags, 1) //nolint:errcheck
		}
		statsd.Client.Gauge("datadog.network_path.path.reachable", float64(utils.BoolToFloat64(lastHop.Success)), tags, 1) //nolint:errcheck
		statsd.Client.Gauge("datadog.network_path.path.unreachable", float64(utils.BoolToFloat64(!lastHop.Success)), tags, 1)
	}
}

func (s *npSchedulerImpl) getTelemetryTags(path traceroute.NetworkPath, ptest *pathtestContext) []string {
	var tags []string
	agentHost, err := hostname.Get(context.TODO())
	if err != nil {
		s.logger.Warnf("Error getting the hostname: %v", err)
	} else {
		tags = append(tags, "agent_host:"+agentHost)
	}
	tags = append(tags, utils.GetAgentVersionTag())

	destPortTag := "unspecified"
	if ptest.pathtest.port > 0 {
		destPortTag = strconv.Itoa(int(ptest.pathtest.port))
	}
	tags = append(tags, []string{
		"protocol:udp", // TODO: Update to protocol from config when we support tcp/icmp
		"destination_hostname:" + path.Destination.Hostname,
		"destination_ip:" + path.Destination.IPAddress,
		"destination_port:" + destPortTag,
	}...)
	return tags
}

func (s *npSchedulerImpl) startWorkers() {
	// TODO: TESTME
	for w := 0; w < s.workers; w++ {
		go s.startWorker(w)
	}
}

func (s *npSchedulerImpl) startWorker(workerID int) {
	// TODO: TESTME
	s.logger.Debugf("Starting worker #%d", workerID)
	for {
		select {
		case <-s.stopChan:
			s.logger.Debugf("[worker%d] Stopping worker", workerID)
			return
		case pathtestCtx := <-s.pathtestProcessChan:
			s.logger.Debugf("[worker%d] Handling pathtest hostname=%s, port=%d", workerID, pathtestCtx.pathtest.hostname, pathtestCtx.pathtest.port)
			s.runTraceroute(pathtestCtx)
		}
	}
}
