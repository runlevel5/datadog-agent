// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package tracer

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/ebpf/ebpftest"
	"github.com/DataDog/datadog-agent/pkg/network/config"
)

func timeSyscall(who int) *syscall.Rusage {
	var rusage syscall.Rusage

	err := syscall.Getrusage(who, &rusage)
	if err != nil {
		return nil
	}

	return &rusage
}

func TestConntrackCompile(t *testing.T) {
	ebpftest.TestBuildMode(t, ebpftest.RuntimeCompiled, "", func(t *testing.T) {
		cfg := config.New()
		cfg.BPFDebug = true

		startSelf := timeSyscall(syscall.RUSAGE_SELF)
		startThread := timeSyscall(syscall.RUSAGE_THREAD)
		startChildren := timeSyscall(syscall.RUSAGE_CHILDREN)
		realStart := time.Now().UnixMicro()

		out, err := getRuntimeCompiledConntracker(cfg)
		realEnd := time.Now().UnixMicro()
		endSelf := timeSyscall(syscall.RUSAGE_SELF)
		endThread := timeSyscall(syscall.RUSAGE_THREAD)
		endChildren := timeSyscall(syscall.RUSAGE_CHILDREN)
		if startSelf != nil && startThread != nil && startChildren != nil && endSelf != nil && endThread != nil && endChildren != nil {
			t.Logf("[Self] User time (sec): %d User time (usec): %d\n", endSelf.Utime.Sec-startSelf.Utime.Sec, endSelf.Utime.Usec-startSelf.Utime.Usec)
			t.Logf("[Self] Sys time (sec): %d Sys time (usec): %d\n", endSelf.Stime.Sec-startSelf.Stime.Sec, endSelf.Stime.Usec-startSelf.Stime.Usec)
			t.Logf("[Thread] User time (sec): %d User time (usec): %d\n", endThread.Utime.Sec-startThread.Utime.Sec, endThread.Utime.Usec-startThread.Utime.Usec)
			t.Logf("[Thread] Sys time (sec): %d Sys time (usec): %d\n", endThread.Stime.Sec-startThread.Stime.Sec, endThread.Stime.Usec-startThread.Stime.Usec)
			t.Logf("[Children] User time (sec): %d User time (usec): %d\n", endChildren.Utime.Sec-startChildren.Utime.Sec, endChildren.Utime.Usec-startChildren.Utime.Usec)
			t.Logf("[Children] Sys time (sec): %d Sys time (usec): %d\n", endChildren.Stime.Sec-startChildren.Stime.Sec, endChildren.Stime.Usec-startChildren.Stime.Usec)
			t.Logf("Real time: %d\n", realEnd-realStart)
		}
		require.NoError(t, err)
		_ = out.Close()
	})
}
