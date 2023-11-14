// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package windows

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func GetServiceAccountName(client *ssh.Client, service string) (string, error) {
	cmd := fmt.Sprintf("(Get-WmiObject Win32_Service -Filter \"Name=`'%s`'\").StartName", service)
	return PsExec(client, cmd)
}

func GetServiceInfo(client *ssh.Client, service string) (map[string]any, error) {
	cmd := fmt.Sprintf("Get-Service -Name '%s' | ConvertTo-Json", service)
	output, err := PsExec(client, cmd)
	if err != nil {
		fmt.Println(output)
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetServiceSDDL(client *ssh.Client, service string) (string, error) {
	cmd := fmt.Sprintf("sc.exe sdshow '%s'", service)
	out, err := PsExec(client, cmd)
	return strings.TrimSpace(out), err
}
