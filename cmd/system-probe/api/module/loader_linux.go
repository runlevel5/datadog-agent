// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package module

import (
	"kernel.org/pub/linux/libs/security/libcap/cap"

	"github.com/DataDog/datadog-agent/cmd/system-probe/config"
	"github.com/DataDog/datadog-agent/pkg/ebpf"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func preRegister(cfg *config.Config) error {
	log.Infof("process capabilities: %s", cap.GetProc().String())
	return ebpf.Setup(ebpf.NewConfig())
}

func postRegister(_ *config.Config) error {
	ebpf.FlushBTF()
	return nil
}
