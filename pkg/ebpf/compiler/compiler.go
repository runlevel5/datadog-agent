// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package compiler

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/kernel"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var (
	datadogAgentEmbeddedPath = "/opt/datadog-agent/embedded"
	clangBinPath             = filepath.Join(datadogAgentEmbeddedPath, "bin/clang-bpf")
	llcBinPath               = filepath.Join(datadogAgentEmbeddedPath, "bin/llc-bpf")

	//go:embed stdarg.h
	stdargHData []byte
)

const compilationStepTimeout = 60 * time.Second

func writeStdarg() (string, error) {
	tmpIncludeDir, err := os.MkdirTemp(os.TempDir(), "include-")
	if err != nil {
		return "", fmt.Errorf("error creating temporary include directory: %s", err.Error())
	}

	if err = os.WriteFile(filepath.Join(tmpIncludeDir, "stdarg.h"), stdargHData, 0644); err != nil {
		return "", fmt.Errorf("error writing data to stdarg.h: %s", err.Error())
	}
	return tmpIncludeDir, nil
}

func kernelHeaderPaths(headerDirs []string) []string {
	arch := kernel.Arch()
	var paths []string
	for _, d := range headerDirs {
		paths = append(paths,
			fmt.Sprintf("%s/arch/%s/include", d, arch),
			fmt.Sprintf("%s/arch/%s/include/generated", d, arch),
			fmt.Sprintf("%s/include", d),
			fmt.Sprintf("%s/arch/%s/include/uapi", d, arch),
			fmt.Sprintf("%s/arch/%s/include/generated/uapi", d, arch),
			fmt.Sprintf("%s/include/uapi", d),
			fmt.Sprintf("%s/include/generated/uapi", d),
		)
	}
	return paths
}

// CompileToObjectFile compiles an eBPF program
func CompileToObjectFile(inFile, outputFile string, cflags []string, headerDirs []string) error {
	if len(headerDirs) == 0 {
		return fmt.Errorf("unable to find kernel headers")
	}

	tmpIncludeDir, err := writeStdarg()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpIncludeDir)
	cflags = append(cflags, fmt.Sprintf("-isystem%s", tmpIncludeDir))

	kps := kernelHeaderPaths(headerDirs)
	for _, p := range kps {
		cflags = append(cflags, fmt.Sprintf("-isystem%s", p))
	}

	cflags = append(cflags, "-c", "-x", "c", "-o", "-", inFile)
	clangOut := &bytes.Buffer{}
	if err := clang(cflags, WithStdout(clangOut)); err != nil {
		return fmt.Errorf("compiling asset to bytecode: %w", err)
	}
	if err := llc(clangOut, outputFile); err != nil {
		return fmt.Errorf("error compiling bytecode to object file: %w", err)
	}
	return nil
}

func WithStdin(in io.Reader) func(*exec.Cmd) {
	return func(c *exec.Cmd) {
		c.Stdin = in
	}
}

func WithStdout(out io.Writer) func(*exec.Cmd) {
	return func(c *exec.Cmd) {
		c.Stdout = out
	}
}

func timeSyscall(who int) *syscall.Rusage {
	var rusage syscall.Rusage

	err := syscall.Getrusage(who, &rusage)
	if err != nil {
		log.Errorf("syscall rusage failed: %v\n", err)
		return nil
	}

	return &rusage
}

func clang(cflags []string, options ...func(*exec.Cmd)) error {
	var clangErr bytes.Buffer

	clangCtx, clangCancel := context.WithTimeout(context.Background(), compilationStepTimeout)
	defer clangCancel()

	compileToBC := exec.CommandContext(clangCtx, clangBinPath, cflags...)
	for _, opt := range options {
		opt(compileToBC)
	}
	compileToBC.Stderr = &clangErr

	log.Debugf("running clang: %v", compileToBC.Args)

	startSelf := timeSyscall(syscall.RUSAGE_SELF)
	startThread := timeSyscall(syscall.RUSAGE_THREAD)
	startChildren := timeSyscall(syscall.RUSAGE_CHILDREN)
	realStart := time.Now().UnixMicro()
	err := compileToBC.Run()
	realEnd := time.Now().UnixMicro()
	endSelf := timeSyscall(syscall.RUSAGE_SELF)
	endThread := timeSyscall(syscall.RUSAGE_THREAD)
	endChildren := timeSyscall(syscall.RUSAGE_CHILDREN)
	if startSelf != nil && startThread != nil && startChildren != nil && endSelf != nil && endThread != nil && endChildren != nil {
		log.Infof("[Self] User time (sec): %d User time (usec): %d\n", endSelf.Utime.Sec-startSelf.Utime.Sec, endSelf.Utime.Usec-startSelf.Utime.Usec)
		log.Infof("[Self] Sys time (sec): %d Sys time (usec): %d\n", endSelf.Stime.Sec-startSelf.Stime.Sec, endSelf.Stime.Usec-startSelf.Stime.Usec)
		log.Infof("[Thread] User time (sec): %d User time (usec): %d\n", endThread.Utime.Sec-startThread.Utime.Sec, endThread.Utime.Usec-startThread.Utime.Usec)
		log.Infof("[Thread] Sys time (sec): %d Sys time (usec): %d\n", endThread.Stime.Sec-startThread.Stime.Sec, endThread.Stime.Usec-startThread.Stime.Usec)
		log.Infof("[Children] User time (sec): %d User time (usec): %d\n", endChildren.Utime.Sec-startChildren.Utime.Sec, endChildren.Utime.Usec-startChildren.Utime.Usec)
		log.Infof("[Children] Sys time (sec): %d Sys time (usec): %d\n", endChildren.Stime.Sec-startChildren.Stime.Sec, endChildren.Stime.Usec-startChildren.Stime.Usec)
		log.Infof("Real time: %d\n", realEnd-realStart)
	}
	if err != nil {
		var errMsg string
		if clangCtx.Err() == context.DeadlineExceeded {
			errMsg = "operation timed out"
		} else if len(clangErr.String()) > 0 {
			errMsg = clangErr.String()
		} else {
			errMsg = err.Error()
		}
		return fmt.Errorf("clang: %s", errMsg)
	}

	if len(clangErr.String()) > 0 {
		log.Debugf("%s", clangErr.String())
	}
	return nil
}

func llc(in io.Reader, outputFile string) error {
	var llcErr bytes.Buffer
	llcCtx, llcCancel := context.WithTimeout(context.Background(), compilationStepTimeout)
	defer llcCancel()

	bcToObj := exec.CommandContext(llcCtx, llcBinPath, "-march=bpf", "-filetype=obj", "-o", outputFile, "-")
	bcToObj.Stdin = in
	bcToObj.Stdout = nil
	bcToObj.Stderr = &llcErr

	log.Debugf("running llc: %v", bcToObj.Args)

	err := bcToObj.Run()
	if err != nil {
		var errMsg string
		if llcCtx.Err() == context.DeadlineExceeded {
			errMsg = "operation timed out"
		} else if len(llcErr.String()) > 0 {
			errMsg = llcErr.String()
		} else {
			errMsg = err.Error()
		}
		return fmt.Errorf("llc: %s", errMsg)
	}

	if len(llcErr.String()) > 0 {
		log.Debugf("%s", llcErr.String())
	}
	return nil
}

// Preprocess runs the clang preprocessor on `in` and writes the output to `out`
func Preprocess(in io.Reader, out io.Writer, cflags []string, headerDirs []string) error {
	tmpIncludeDir, err := writeStdarg()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpIncludeDir)
	cflags = append(cflags, fmt.Sprintf("-isystem%s", tmpIncludeDir))

	kps := kernelHeaderPaths(headerDirs)
	for _, p := range kps {
		cflags = append(cflags, fmt.Sprintf("-isystem%s", p))
	}

	cflags = append(cflags, "-E", "-x", "c", "-o", "-", "-")
	return clang(cflags, WithStdin(in), WithStdout(out))
}
