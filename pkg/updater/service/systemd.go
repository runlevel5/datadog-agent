// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows

// Package service provides a way to interact with os services
package service

import (
	"fmt"
	"path/filepath"

	"github.com/DataDog/datadog-agent/pkg/config/setup"
)

const (
	adminExecutor = "datadog-updater-admin.service"
	libSystemdDir = "/lib/systemd/system"
)

var unitPath = filepath.Join(setup.InstallPath, "systemd")

func stopUnit(unit string) error {
	return RootExec(wrapUnitCommand("stop", unit))
}

func startUnit(unit string) error {
	return RootExec(wrapUnitCommand("start", unit))
}

func enableUnit(unit string) error {
	return RootExec(wrapUnitCommand("enable", unit))
}

func disableUnit(unit string) error {
	return RootExec(wrapUnitCommand("disable", unit))
}

func loadUnit(unit string) error {
	// todo non debian
	return RootExec(fmt.Sprintf("cp %s %s" + filepath.Join(unitPath, unit) + filepath.Join(libSystemdDir, unit)))
}

func removeUnit(unit string) error {
	// todo non debian
	return RootExec(fmt.Sprintf("rm %s" + filepath.Join(unitPath, unit)))
}

func systemdReload() error {
	return RootExec("systemctl daemon-reload")
}

func wrapUnitCommand(command, unit string) string {
	return fmt.Sprintf("systemctl %s %s", string(command), unit)
}
