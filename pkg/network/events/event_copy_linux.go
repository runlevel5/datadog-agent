// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.
// Code generated - DO NOT EDIT.

package events

import (
	"go4.org/intern"

	smodel "github.com/DataDog/datadog-agent/pkg/security/secl/model"
)

var _ = intern.Value{}

func (h *eventConsumerWrapper) Copy(event *smodel.Event) any {
	var result Process

	valuePid := event.GetProcessPid()
	result.Pid = valuePid

	valueEnvs := smodel.FilterEnvs(event.GetProcessEnvp(), map[string]bool{"DD_SERVICE": true, "DD_VERSION": true, "DD_ENV": true})
	result.Envs = valueEnvs

	valueContainerID := intern.GetByString(event.GetContainerId())
	result.ContainerID = valueContainerID

	if event.GetEventType() == smodel.ExecEventType {
		valueExecTime := event.GetProcessExecTime()
		result.ExecTime = valueExecTime
	}

	if event.GetEventType() == smodel.ForkEventType {
		valueForkTime := event.GetProcessForkTime()
		result.ForkTime = valueForkTime
	}
	return &result
}
