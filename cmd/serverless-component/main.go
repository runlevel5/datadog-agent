// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/cmd/serverless-component/command"
	"github.com/DataDog/datadog-agent/pkg/util/flavor"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const defaultLogFile = "/var/log/datadog/logs-agent.log"

func main() {
	currentTime := time.Now().UnixNano()
	bootTime := os.Args[1]
	bootTimeInt64, _ := strconv.ParseInt(bootTime, 10, 64)
	fmt.Printf("boot in : %d\n", currentTime-bootTimeInt64)
	os.Exit(0)
	flavor.SetFlavor(flavor.DefaultAgent)

	if err := command.MakeRootCommand(defaultLogFile).Execute(); err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}
