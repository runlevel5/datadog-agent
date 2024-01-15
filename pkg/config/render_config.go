// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build ignore

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DataDog/datadog-agent/pkg/config"
)

func main() {
	if len(os.Args[1:]) != 3 {
		panic("please use 'go run render_config.go <component_name> <template_file> <destination_file>'")
	}

	component := os.Args[1]
	tplFile, _ := filepath.Abs(os.Args[2])
	tplFilename := filepath.Base(tplFile)
	destFile, _ := filepath.Abs(os.Args[3])

	f, err := os.Create(destFile)
	if err != nil {
		panic(err)
	}

	err = config.WriteConfigFromFile(f, component, tplFilename, tplFile)
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully wrote", destFile)
}
