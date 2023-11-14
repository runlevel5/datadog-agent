// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package windows

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

func LocalUserExists(client *ssh.Client, user string) (bool, error) {
	cmd := fmt.Sprintf("(Get-LocalUser).Name -Contains '%s'", user)
	out, err := PsExec(client, cmd)
	if err != nil {
		return false, err
	}
	return out == "True", nil
}

func CreateLocalUser(client *ssh.Client, user string, password string) error {
	cmd := fmt.Sprintf("net.exe user '%s' '%s' /ADD", user, password)
	out, err := PsExec(client, cmd)
	if err != nil {
		fmt.Println(out)
		return fmt.Errorf("Failed to create user '%s': %v", user, err)
	}
	return nil
}

func GetSIDForUser(client *ssh.Client, user string) (string, error) {
	cmd := fmt.Sprintf("(Get-WmiObject Win32_UserAccount -Filter \"Name=`'%s`'\").SID", user)
	return PsExec(client, cmd)
}
