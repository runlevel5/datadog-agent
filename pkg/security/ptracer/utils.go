// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package ptracer holds the start command of CWS injector
package ptracer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"github.com/DataDog/datadog-agent/pkg/security/common/containerutils"
	"github.com/DataDog/datadog-agent/pkg/security/proto/ebpfless"
)

// Funcs mainly copied from github.com/DataDog/datadog-agent/pkg/security/utils/cgroup.go
// in order to reduce the binary size of cws-instrumentation

type controlGroup struct {
	// id unique hierarchy ID
	id int

	// controllers are the list of cgroup controllers bound to the hierarchy
	controllers []string

	// path is the pathname of the control group to which the process
	// belongs. It is relative to the mountpoint of the hierarchy.
	path string
}

func getProcControlGroupsFromFile(path string) ([]controlGroup, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cgroups []controlGroup
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		t := scanner.Text()
		parts := strings.Split(t, ":")
		var ID int
		ID, err = strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		c := controlGroup{
			id:          ID,
			controllers: strings.Split(parts[1], ","),
			path:        parts[2],
		}
		cgroups = append(cgroups, c)
	}
	return cgroups, nil

}

func getCurrentProcContainerID() (string, error) {
	cgroups, err := getProcControlGroupsFromFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}

	for _, cgroup := range cgroups {
		cid := containerutils.FindContainerID(cgroup.path)
		if cid != "" {
			return cid, nil
		}
	}
	return "", nil
}

func retrieveContainerIDFromProc(ctx *ebpfless.ContainerContext) error {
	cgroup, err := getCurrentProcContainerID()
	if err != nil {
		return err
	}
	ctx.ID = cgroup
	return nil
}

func getNSID() uint64 {
	var stat syscall.Stat_t
	if err := syscall.Lstat("/proc/self/ns/pid", &stat); err != nil {
		return rand.Uint64()
	}
	return stat.Ino
}

const (
	uids = 1 << iota
	gids
	all = uids | gids
)

var (
	keyUID = []byte("Uid")
	keyGid = []byte("Gid")
)

type procStatusInfo struct {
	uids         []uint32
	gids         []uint32
	parsedFields uint32
}

func parseProcStatusKV(key, value []byte, info *procStatusInfo) {
	switch {
	case bytes.Equal(key, keyUID), bytes.Equal(key, keyGid):
		values := bytes.Fields(value)
		ints := make([]uint32, 0, len(values))
		for _, v := range values {
			if i, err := strconv.ParseInt(string(v), 10, 32); err == nil {
				ints = append(ints, uint32(i))
			}
		}
		if key[0] == 'U' {
			info.uids = ints
			info.parsedFields |= uids
		} else {
			info.gids = ints
			info.parsedFields |= gids
		}
	}
}

func parseProcStatusLine(line []byte, info *procStatusInfo) {
	for i := range line {
		// the fields are all having format "field_name:\s+field_value", so we always
		// look for ":\t" and skip them
		if i+2 < len(line) && line[i] == ':' && unicode.IsSpace(rune(line[i+1])) {
			key := line[0:i]
			value := line[i+2:]
			parseProcStatusKV(key, value, info)
		}
	}
}

func readProcStatus(pid int) (*procStatusInfo, error) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var info procStatusInfo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		parseProcStatusLine(line, &info)
		if info.parsedFields == all {
			break
		}
	}

	return &info, nil
}

func getProcStatusCredentials(pid int) (*ebpfless.Credentials, error) {
	info, err := readProcStatus(pid)
	if err != nil {
		return nil, err
	}

	if info.parsedFields&(uids|gids) != (uids | gids) {
		return nil, errors.New("failed to read uids/gids")
	}

	creds := &ebpfless.Credentials{}

	if len(info.uids) > 0 {
		creds.UID = info.uids[0]
		if len(info.uids) > 1 {
			creds.EUID = info.uids[1]
		}
	}
	if len(info.gids) > 0 {
		creds.GID = info.gids[0]
		if len(info.gids) > 1 {
			creds.EGID = info.gids[1]
		}
	}

	return creds, nil
}
