// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package client

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/components"
	"github.com/DataDog/test-infra-definitions/components/os"
)

// EC2Metadata contains a pointer to a VM and its AWS token
type EC2Metadata struct {
	h     *components.RemoteHost
	token string
}

const metadataEndPoint = "http://169.254.169.254"

// NewEC2Metadata creates a new [EC2Metadata] given an EC2 [VM]
func NewEC2Metadata(h *components.RemoteHost) *EC2Metadata {
	var cmd string

	switch h.OSFamily {
	case os.WindowsFamily:
		cmd = fmt.Sprintf(`Invoke-RestMethod -Uri "%v/latest/api/token" -Method Put -Headers @{ "X-aws-ec2-metadata-token-ttl-seconds" = "21600" }`, metadataEndPoint)
	case os.LinuxFamily:
		cmd = fmt.Sprintf(`curl -s -X PUT "%v/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600"`, metadataEndPoint)
	default:
		panic(fmt.Sprintf("unsupported OS family: %v", h.OSFamily))
	}

	output := h.MustExecute(cmd)
	return &EC2Metadata{h: h, token: output}
}

// Get returns EC2 instance name
func (m *EC2Metadata) Get(name string) string {

	var cmd string
	switch m.h.OSFamily {
	case os.WindowsFamily:
		cmd = fmt.Sprintf(`Invoke-RestMethod  -Headers @{"X-aws-ec2-metadata-token"="%v"} -Uri "%v/latest/meta-data/%v"`, m.token, metadataEndPoint, name)
	case os.LinuxFamily:
		cmd = fmt.Sprintf(`curl -s -H "X-aws-ec2-metadata-token: %v" "%v/latest/meta-data/%v"`, m.token, metadataEndPoint, name)
	default:
		panic(fmt.Sprintf("unsupported OS family: %v", m.h.OSFamily))
	}

	return strings.TrimRight(m.h.MustExecute(cmd), "\r\n")
}
