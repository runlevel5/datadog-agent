// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-agent/comp/snmpwalk/fetch/valuestore"
	"github.com/DataDog/datadog-agent/pkg/aggregator/sender"
	"github.com/DataDog/datadog-agent/pkg/epforwarder"
	"github.com/DataDog/datadog-agent/pkg/networkdevice/metadata"
	parse "github.com/DataDog/datadog-agent/pkg/snmp/snmpparse"
	"github.com/DataDog/datadog-agent/pkg/util/hostname"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/gosnmp/gosnmp"
	"os"
	"strings"
	"time"
)

// SnmpwalkRunner receives configuration from remote-config
type SnmpwalkRunner struct {
	upToDate bool
	sender   sender.Sender
}

// NewSnmpwalkRunner creates a new SnmpwalkRunner.
func NewSnmpwalkRunner(sender sender.Sender) *SnmpwalkRunner {
	return &SnmpwalkRunner{
		upToDate: false,
		sender:   sender,
	}
}

// Callback is when profiles updates are available (rc product NDM_DEVICE_PROFILES_CUSTOM)
func (rc *SnmpwalkRunner) Callback() {
	globalStart := time.Now()
	log.Info("[SNMP RUNNER] SNMP RUNNER")

	// TODO: Do not collect snmp-listener configs
	snmpConfigList, err := parse.GetConfigCheckSnmp()
	if err != nil {
		log.Infof("[SNMP RUNNER] Couldn't parse the SNMP config: %v", err)
		return
	}
	log.Infof("[SNMP RUNNER] snmpConfigList len=%d", len(snmpConfigList))

	for _, config := range snmpConfigList {
		log.Infof("[SNMP RUNNER] SNMP config: %+v", config)
	}

	hname, err := hostname.Get(context.TODO())
	if err != nil {
		log.Warnf("Error getting the hostname: %v", err)
	}
	commonTags := []string{
		"agent_host:" + hname,
	}
	namespace := "default"
	for _, config := range snmpConfigList {
		ipaddr := config.IPAddress
		if ipaddr == "" {
			log.Infof("[SNMP RUNNER] Missing IP Addr: %v", config)
			continue
		}
		devTags := []string{
			"namespace:" + namespace, // TODO: FIX ME
			"device_ip:" + ipaddr,
			"device_id:" + namespace + ":" + ipaddr,
		}
		for _, tag := range commonTags {
			devTags = append(devTags, tag)
		}
		log.Infof("[SNMP RUNNER] Run Device OID Scan for: %s", ipaddr)

		localStart := time.Now()
		oidsCollectedCount := rc.collectDeviceOIDs(config)
		duration := time.Since(localStart)
		rc.sender.Gauge("datadog.snmpwalk.device.duration", duration.Seconds(), "", devTags)
		rc.sender.Gauge("datadog.snmpwalk.device.oids", float64(oidsCollectedCount), "", devTags)
	}
	rc.sender.Gauge("datadog.snmpwalk.total.duration", time.Since(globalStart).Seconds(), "", commonTags)
	rc.sender.Commit()
}

func (rc *SnmpwalkRunner) collectDeviceOIDs(config parse.SNMPConfig) int {
	prefix := fmt.Sprintf("(%s)", config.CommunityString)
	namespace := "default" // TODO: CHANGE PLACEHOLDER
	deviceId := namespace + ":" + config.IPAddress

	session := createSession(config)
	log.Infof("[SNMP RUNNER]%s session: %+v", prefix, session)

	// Establish connection
	err := session.Connect()
	if err != nil {
		log.Errorf("[SNMP RUNNER]%s Connect err: %v\n", prefix, err)
		os.Exit(1)
		return 0
	}
	defer session.Conn.Close()

	variables := FetchAllFirstRowOIDsVariables(session)
	log.Infof("[SNMP RUNNER]%s Variables: %d", prefix, len(variables))

	for idx, variable := range variables {
		log.Infof("[SNMP RUNNER]%s Variable Name (%d): %s", prefix, idx+1, variable.Name)
	}

	deviceOIDs := buildDeviceScanMetadata(deviceId, variables)

	metadataPayloads := metadata.BatchPayloads(namespace,
		"",
		time.Now(),
		metadata.PayloadMetadataBatchSize,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		deviceOIDs,
	)
	for _, payload := range metadataPayloads {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Errorf("[SNMP RUNNER]%s Error marshalling device metadata: %s", prefix, err)
			continue
		}
		log.Debugf("[SNMP RUNNER]%s Device OID metadata payload: %s", prefix, string(payloadBytes))
		rc.sender.EventPlatformEvent(payloadBytes, epforwarder.EventTypeNetworkDevicesMetadata)
		if err != nil {
			log.Errorf("[SNMP RUNNER]%s Error sending event platform event for Device OID metadata: %s", prefix, err)
		}
	}
	return len(deviceOIDs)
}

func buildDeviceScanMetadata(deviceId string, oidsValues []gosnmp.SnmpPDU) []metadata.DeviceOid {
	var deviceOids []metadata.DeviceOid
	if oidsValues == nil {
		return deviceOids
	}
	for _, variablePdu := range oidsValues {
		_, value, err := valuestore.GetResultValueFromPDU(variablePdu)
		if err != nil {
			log.Debugf("GetValueFromPDU error: %s", err)
			continue
		}

		// TODO: How to store different types? Use Base64?
		strValue, err := value.ToString()
		if err != nil {
			log.Debugf("ToString error: %s", err)
			continue
		}

		deviceOids = append(deviceOids, metadata.DeviceOid{
			DeviceID:    deviceId,
			Oid:         strings.TrimLeft(variablePdu.Name, "."),
			Type:        variablePdu.Type.String(), // TODO: Map to internal types?
			ValueString: strValue,
		})
	}
	return deviceOids
}
