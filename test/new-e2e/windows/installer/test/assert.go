// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package installertest

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/test/new-e2e/windows"
	"github.com/DataDog/datadog-agent/test/new-e2e/windows/installer"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

func AssertDefaultInstalledUser(a *assert.Assertions, client *ssh.Client) bool {
	// get hostname
	hostinfo, err := windows.GetHostInfo(client)
	if !a.NoError(err) {
		return false
	}
	username, userdomain, serviceuser := installer.DefaultAgentUser(hostinfo)

	return AssertInstalledUser(a, client, username, userdomain, serviceuser)
}

func AssertInstalledUser(a *assert.Assertions, client *ssh.Client, expectedusername string, expecteddomain string, expectedserviceuser string) bool {
	// check registry keys
	username, err := windows.GetRegistryValue(client, "HKLM:\\SOFTWARE\\Datadog\\Datadog Agent", "installedUser")
	if !a.NoError(err) {
		return false
	}
	domain, err := windows.GetRegistryValue(client, "HKLM:\\SOFTWARE\\Datadog\\Datadog Agent", "installedDomain")
	if !a.NoError(err) {
		return false
	}
	username = strings.ToLower(username)
	expectedusername = strings.ToLower(expectedusername)
	// It's not a perfect test to be comparing the NetBIOS version of each domain, but the installer isn't
	// consistent with what it writes to the registry. On domain controllers, if the user exists then the domain part comes from the output
	// of LookupAccountName, which seems to consistently be a NetBIOS name. However, if the installer creates the account and a domain part wasn't
	// provided, then the FQDN is used and written to the registry.
	domain = windows.NetBIOSName(domain)
	expecteddomain = windows.NetBIOSName(expecteddomain)
	if strings.Contains(expectedserviceuser, "\\") {
		parts := strings.Split(expectedserviceuser, "\\")
		netbios := windows.NetBIOSName(parts[0])
		expectedserviceuser = fmt.Sprintf("%s\\%s", netbios, parts[1])
	}

	if !a.Equal(expectedusername, username, "installedUser registry value should be %s", expectedusername) {
		return false
	}
	if !a.Equal(expecteddomain, domain, "installedDomain registry value should be %s", expecteddomain) {
		return false
	}

	// check service users
	svcs := []struct {
		name    string
		account string
	}{
		{"datadogagent", expectedserviceuser},
		{"datadog-trace-agent", expectedserviceuser},
		{"datadog-system-probe", "LocalSystem"},
		{"datadog-process-agent", "LocalSystem"},
	}
	for _, svc := range svcs {
		user, err := windows.GetServiceAccountName(client, svc.name)
		if !a.NoError(err) {
			return false
		}
		// Ditto above comment about comparing NetBIOS version of domain part
		if strings.Contains(user, "\\") {
			parts := strings.Split(user, "\\")
			netbios := windows.NetBIOSName(parts[0])
			user = fmt.Sprintf("%s\\%s", netbios, parts[1])
		}
		if !a.Equal(strings.ToLower(svc.account), strings.ToLower(user), "%s logon account should be %s", svc.name, svc.account) {
			return false
		}
	}

	return true
}

func AssertInstalledDirectoriesExist(a *assert.Assertions, client *ssh.Client) bool {
	sftpclient, err := sftp.NewClient(client)
	if err != nil {
		return false
	}
	defer sftpclient.Close()
	dirs := []string{
		"C:\\ProgramData\\Datadog\\checks.d",
		"C:\\ProgramData\\Datadog\\logs",
		"C:\\ProgramData\\Datadog\\run",
		"C:\\Program Files\\Datadog\\Datadog Agent\\embedded3",
	}
	for _, dir := range dirs {
		_, err = sftpclient.Stat(dir)
		a.NoError(err, fmt.Sprintf("'%s' should exist but doesn't", dir))
	}

	return true
}
