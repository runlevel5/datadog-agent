// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build windows

// Package path holds path related files
package path

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"

	"github.com/DataDog/datadog-agent/pkg/util/winutil"
)

// Resolver describes a resolvers for path and file names
type Resolver struct {
	volumeMap map[string]string
}

// NewResolver returns a new path resolver
func NewResolver() (*Resolver, error) {
	volumeMap, err := buildVolumeMap()
	if err != nil {
		return nil, err
	}

	return &Resolver{volumeMap: volumeMap}, nil
}

// ResolveUserPath resolves a device path to a user path
func (r *Resolver) ResolveUserPath(devicePath string) string {
	// filepath doesn't seem to like the \Device\HarddiskVolume1 format
	pathchunks := strings.Split(devicePath, "\\")
	if len(pathchunks) > 2 {
		if strings.EqualFold(pathchunks[1], "device") {
			pathchunks[2] = r.volumeMap[strings.ToLower(pathchunks[2])]
			return filepath.Join(pathchunks[2:]...)
		}
	}
	return devicePath
}

func buildVolumeMap() (map[string]string, error) {
	buf := make([]uint16, 1024)
	bufferLength := uint32(len(buf))

	_, err := windows.GetLogicalDriveStrings(bufferLength, &buf[0])
	if err != nil {
		return nil, err
	}

	drives := winutil.ConvertWindowsStringList(buf)
	volumeMap := make(map[string]string)

	for _, drive := range drives {
		t := windows.GetDriveType(windows.StringToUTF16Ptr(drive[:3]))
		/*
			DRIVE_UNKNOWN
			0
			The drive type cannot be determined.
			DRIVE_NO_ROOT_DIR
			1
			The root path is invalid; for example, there is no volume mounted at the specified path.
			DRIVE_REMOVABLE
			2
			The drive has removable media; for example, a floppy drive, thumb drive, or flash card reader.
			DRIVE_FIXED
			3
			The drive has fixed media; for example, a hard disk drive or flash drive.
			DRIVE_REMOTE
			4
			The drive is a remote (network) drive.
			DRIVE_CDROM
			5
			The drive is a CD-ROM drive.
			DRIVE_RAMDISK
			6
			The drive is a RAM disk.
		*/
		if t == windows.DRIVE_FIXED {
			volpath := make([]uint16, 1024)
			vollen := uint32(len(volpath))
			_, err = windows.QueryDosDevice(windows.StringToUTF16Ptr(drive[:2]), &volpath[0], vollen)
			if err == nil {
				devname := windows.UTF16PtrToString(&volpath[0])
				paths := strings.Split(devname, "\\") // apparently, filepath.split doesn't like volume names

				if len(paths) > 2 {
					// the \Device leads to the first entry being empty
					if strings.EqualFold(paths[1], "device") {
						volumeMap[strings.ToLower(paths[2])] = drive
					}
				}
			}
		}
	}
	return volumeMap, nil
}
