// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	_ "embed"
	"html/template"
	"io"
	"os"
	"runtime"
	"strings"
)

// templateContext contains the templateContext used to render the config file template
type templateContext struct {
	OS                               string
	Common                           bool
	Agent                            bool
	Python                           bool // Sub-option of Agent
	BothPythonPresent                bool // Sub-option of Agent - Python
	Metadata                         bool
	InternalProfiling                bool
	Dogstatsd                        bool
	LogsAgent                        bool
	JMX                              bool
	Autoconfig                       bool
	Logging                          bool
	Autodiscovery                    bool
	DockerTagging                    bool
	Kubelet                          bool
	KubernetesTagging                bool
	ECS                              bool
	Containerd                       bool
	CRI                              bool
	ProcessAgent                     bool
	SystemProbe                      bool
	KubeApiServer                    bool
	TraceAgent                       bool
	ClusterAgent                     bool
	ClusterChecks                    bool
	AdmissionController              bool
	CloudFoundryBBS                  bool
	CloudFoundryCC                   bool
	Compliance                       bool
	SNMP                             bool
	SecurityModule                   bool
	SecurityAgent                    bool
	NetworkModule                    bool // Sub-module of System Probe
	UniversalServiceMonitoringModule bool // Sub-module of System Probe
	DataStreamsModule                bool // Sub-module of System Probe
	PrometheusScrape                 bool
	OTLP                             bool
	APMInjection                     bool
}

func mkContext(buildType string) templateContext {
	buildType = strings.ToLower(buildType)

	agentContext := templateContext{
		OS:                runtime.GOOS,
		Common:            true,
		Agent:             true,
		Python:            true,
		Metadata:          true,
		InternalProfiling: false, // NOTE: hidden for now
		Dogstatsd:         true,
		LogsAgent:         true,
		JMX:               true,
		Autoconfig:        true,
		Logging:           true,
		Autodiscovery:     true,
		DockerTagging:     true,
		KubernetesTagging: true,
		ECS:               true,
		Containerd:        true,
		CRI:               true,
		ProcessAgent:      true,
		TraceAgent:        true,
		Kubelet:           true,
		KubeApiServer:     true, // TODO: remove when phasing out from node-agent
		Compliance:        true,
		SNMP:              true,
		PrometheusScrape:  true,
		OTLP:              true,
	}

	switch buildType {
	case "agent-py3":
		return agentContext
	case "agent-py2py3":
		agentContext.BothPythonPresent = true
		return agentContext
	case "iot-agent":
		return templateContext{
			OS:        runtime.GOOS,
			Common:    true,
			Agent:     true,
			Metadata:  true,
			Dogstatsd: true,
			LogsAgent: true,
			Logging:   true,
		}
	case "system-probe":
		return templateContext{
			OS:                               runtime.GOOS,
			SystemProbe:                      true,
			NetworkModule:                    true,
			UniversalServiceMonitoringModule: true,
			DataStreamsModule:                true,
			SecurityModule:                   true,
		}
	case "dogstatsd":
		return templateContext{
			OS:                runtime.GOOS,
			Common:            true,
			Dogstatsd:         true,
			DockerTagging:     true,
			Logging:           true,
			KubernetesTagging: true,
			ECS:               true,
			TraceAgent:        true,
			Kubelet:           true,
		}
	case "dca":
		return templateContext{
			OS:                  runtime.GOOS,
			ClusterAgent:        true,
			Common:              true,
			Logging:             true,
			KubeApiServer:       true,
			ClusterChecks:       true,
			AdmissionController: true,
		}
	case "dcacf":
		return templateContext{
			OS:              runtime.GOOS,
			ClusterAgent:    true,
			Common:          true,
			Logging:         true,
			ClusterChecks:   true,
			CloudFoundryBBS: true,
			CloudFoundryCC:  true,
		}
	case "security-agent":
		return templateContext{
			OS:            runtime.GOOS,
			SecurityAgent: true,
		}
	case "apm-injection":
		return templateContext{
			OS:           runtime.GOOS,
			APMInjection: true,
		}
	}

	return templateContext{}
}

// WriteConfigFromString renders the given template and writes it to the writer
func WriteConfigFromString(writer io.Writer, buildType, tplFilename, tmpl string) error {
	t := template.Must(template.New(tplFilename).Parse(tmpl))
	return t.Execute(writer, mkContext(buildType))
}

//go:embed config_template.yaml
var configTemplate string

// WriteConfigFromTemplate renders the config template and writes it to the given writer
func WriteConfigFromTemplate(writer io.Writer, buildType string) error {
	return WriteConfigFromString(writer, buildType, "config_template.yaml", configTemplate)
}

// WriteConfigFromFile renders the given file and writes it to the writer
func WriteConfigFromFile(writer io.Writer, buildType, tplFilename, tplFile string) error {
	file, err := os.Open(tplFile)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return WriteConfigFromString(writer, buildType, tplFilename, string(data))
}
