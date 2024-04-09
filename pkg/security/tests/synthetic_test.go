// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux && functionaltests

// Package tests holds tests related files
package tests

import (
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/secl/rules"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestSynthetic(t *testing.T) {
	synthetics := []rules.SyntheticHookPoint{
		{
			Name: "do_sys_openat2",
			Args: []rules.HookPointArg{
				{
					N:    1,
					Kind: "int",
				},
				{
					N:    2,
					Kind: "null-terminated-string",
				},
			},
		},
	}
	rule := &rules.RuleDefinition{
		ID:         "test_rule",
		Expression: `synthetic.name == "do_sys_openat2" && synthetic.arg2.str =~ ~"*/test-open" && process.file.name == "testsuite"`,
	}

	test, err := newTestModuleWithSynthetics(t, synthetics, nil, []*rules.RuleDefinition{rule})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	fileMode := 0o447
	expectedMode := uint64(applyUmask(fileMode))
	testFile, testFilePtr, err := test.CreateWithOptions("test-open", 98, 99, fileMode)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	t.Run("open", func(t *testing.T) {
		test.WaitSignal(t, func() error {
			openHow := unix.OpenHow{
				Flags: unix.O_RDONLY,
				Mode:  expectedMode,
			}
			fd, _, errno := syscall.Syscall6(unix.SYS_OPENAT2, 0, uintptr(testFilePtr), uintptr(unsafe.Pointer(&openHow)), unix.SizeofOpenHow, 0, 0)
			if errno != 0 {
				return err
			}
			return syscall.Close(int(fd))
		}, func(event *model.Event, r *rules.Rule) {
			assert.Equal(t, "synthetic", event.GetType(), "wrong event type")

			value, _ := event.GetFieldValue("synthetic.arg2.str")
			assert.Equal(t, value.(string), testFile)
		})
	})
}
