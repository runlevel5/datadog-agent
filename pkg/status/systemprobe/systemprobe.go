// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build process

// Package systemprobe fetch information about the system probe
package systemprobe

import (
	"embed"
	"fmt"
	"io"
	"path"

	textTemplate "text/template"

	"github.com/DataDog/datadog-agent/comp/core/status"
	"github.com/DataDog/datadog-agent/pkg/process/net"
)

// GetStatus returns the expvar stats of the system probe
func GetStatus(stats map[string]interface{}, socketPath string) {
	probeUtil, err := net.GetRemoteSystemProbeUtil(socketPath)

	if err != nil {
		stats["systemProbeStats"] = map[string]interface{}{
			"Errors": fmt.Sprintf("%v", err),
		}
		return
	}

	systemProbeDetails, err := probeUtil.GetStats()
	if err != nil {
		stats["systemProbeStats"] = map[string]interface{}{
			"Errors": fmt.Sprintf("issue querying stats from system probe: %v", err),
		}
		return
	}
	stats["systemProbeStats"] = systemProbeDetails
}

// Provider provides the functionality to populate the status output
type Provider struct {
	SocketPath string
}

//go:embed status_templates
var templatesFS embed.FS

// Name returns the name
func (Provider) Name() string {
	return "System Probe"
}

// Section return the section
func (Provider) Section() string {
	return "System Probe"
}

// JSON populates the status map
func (p Provider) JSON(verbose bool, stats map[string]interface{}) error {
	GetStatus(stats, p.SocketPath)

	return nil
}

func (p Provider) Text(_ bool, buffer io.Writer) error {
	return renderText(buffer, p.getStatusInfo())
}

func (p Provider) HTML(_ bool, buffer io.Writer) error {
	return nil
}

func renderText(buffer io.Writer, data any) error {
	tmpl, tmplErr := templatesFS.ReadFile(path.Join("status_templates", "clusteragent.tmpl"))
	if tmplErr != nil {
		return tmplErr
	}
	t := textTemplate.Must(textTemplate.New("clusteragent").Funcs(status.TextFmap()).Parse(string(tmpl)))
	return t.Execute(buffer, data)
}

func (p Provider) getStatusInfo() map[string]interface{} {
	stats := make(map[string]interface{})

	GetStatus(stats, p.SocketPath)

	return stats
}
