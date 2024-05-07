// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build !windows

// Package service provides a way to interact with os services
package service

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const (
	ldSoPreloadPath = "/etc/ld.so.preload"
	injectorPath    = "/opt/datadog-packages/datadog-apm-inject/stable"
	oldLDPath       = "/opt/datadog/apm/inject/launcher.preload.so"
)

// SetupAPMInjector sets up the injector at bootstrap
func SetupAPMInjector(ctx context.Context) error {
	var err error
	span, ctx := tracer.StartSpanFromContext(ctx, "setup_injector")
	defer span.Finish(tracer.WithError(err))
	installer := newAPMInjectorInstaller(injectorPath)
	return installer.Setup(ctx)
}

// RemoveAPMInjector removes the APM injector
func RemoveAPMInjector(ctx context.Context) {
	span, ctx := tracer.StartSpanFromContext(ctx, "remove_injector")
	defer span.Finish()
	installer := newAPMInjectorInstaller(injectorPath)
	installer.Remove(ctx)
}

func newAPMInjectorInstaller(path string) *apmInjectorInstaller {
	a := &apmInjectorInstaller{
		installPath: path,
	}
	a.ldPreloadFileInstrument = newFileMutator(ldSoPreloadPath, a.setLDPreloadConfigContent, nil, nil)
	a.ldPreloadFileUninstrument = newFileMutator(ldSoPreloadPath, a.deleteLDPreloadConfigContent, nil, nil)
	a.dockerConfigInstrument = newFileMutator(dockerDaemonPath, a.setDockerConfigContent, nil, nil)
	a.dockerConfigUninstrument = newFileMutator(dockerDaemonPath, a.deleteDockerConfigContent, nil, nil)
	return a
}

type apmInjectorInstaller struct {
	installPath               string
	ldPreloadFileInstrument   *fileMutator
	ldPreloadFileUninstrument *fileMutator
	dockerConfigInstrument    *fileMutator
	dockerConfigUninstrument  *fileMutator
}

// Setup sets up the APM injector
func (a *apmInjectorInstaller) Setup(ctx context.Context) (err error) {
	var rollbackLDPreload, rollbackDockerConfig func() error
	defer func() {
		if err != nil {
			// todo propagate rollbacks until success of package installation
			if rollbackLDPreload != nil {
				if err := rollbackLDPreload(); err != nil {
					log.Warnf("Failed to rollback ld preload: %v", err)
				}
			}
			if rollbackDockerConfig != nil {
				if err := rollbackDockerConfig(); err != nil {
					log.Warnf("Failed to rollback docker config: %v", err)
				}
			}
		}
	}()

	rollbackLDPreload, err = a.ldPreloadFileInstrument.mutate()
	if err != nil {
		return err
	}

	// TODO only instrument docker if DD_APM_INSTRUMENTATION_ENABLED=docker is set
	rollbackDockerConfig, err = a.setupDocker(ctx)
	return err
}

func (a *apmInjectorInstaller) Remove(ctx context.Context) {
	if _, err := a.ldPreloadFileUninstrument.mutate(); err != nil {
		log.Warnf("Failed to remove ld preload config: %v", err)
	}
	// TODO docker only on DD_APM_INSTRUMENTATION_ENABLED=docker
	if err := a.uninstallDocker(ctx); err != nil {
		log.Warnf("Failed to remove docker config: %v", err)
	}
}

// setLDPreloadConfigContent sets the content of the LD preload configuration
func (a *apmInjectorInstaller) setLDPreloadConfigContent(ldSoPreload []byte) ([]byte, error) {
	launcherPreloadPath := path.Join(a.installPath, "inject", "launcher.preload.so")

	if strings.Contains(string(ldSoPreload), launcherPreloadPath) {
		// If the line of interest is already in /etc/ld.so.preload, return fast
		return ldSoPreload, nil
	}

	if bytes.Contains(ldSoPreload, []byte(oldLDPath)) {
		return bytes.ReplaceAll(ldSoPreload, []byte(oldLDPath), []byte(launcherPreloadPath)), nil
	}

	var buf bytes.Buffer
	buf.Write(ldSoPreload)
	// Append the launcher preload path to the file
	if len(ldSoPreload) > 0 && ldSoPreload[len(ldSoPreload)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString(launcherPreloadPath)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// deleteLDPreloadConfigContent deletes the content of the LD preload configuration
func (a *apmInjectorInstaller) deleteLDPreloadConfigContent(ldSoPreload []byte) ([]byte, error) {
	launcherPreloadPath := path.Join(a.installPath, "inject", "launcher.preload.so")

	if !strings.Contains(string(ldSoPreload), launcherPreloadPath) {
		// If the line of interest isn't there, return fast
		return ldSoPreload, nil
	}

	// Possible configurations of the preload path, order matters
	replacementsToTest := [][]byte{
		[]byte(launcherPreloadPath + "\n"),
		[]byte("\n" + launcherPreloadPath),
		[]byte(launcherPreloadPath + " "),
		[]byte(" " + launcherPreloadPath),
	}
	for _, replacement := range replacementsToTest {
		ldSoPreloadNew := bytes.Replace(ldSoPreload, replacement, []byte{}, 1)
		if !bytes.Equal(ldSoPreloadNew, ldSoPreload) {
			return ldSoPreloadNew, nil
		}
	}
	if bytes.Equal(ldSoPreload, []byte(launcherPreloadPath)) {
		// If the line is the only one in the file without newlines, return an empty file
		return []byte{}, nil
	}

	return nil, fmt.Errorf("failed to remove %s from %s", launcherPreloadPath, ldSoPreloadPath)
}
