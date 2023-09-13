// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package installertest

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/test/new-e2e/windows"
	"github.com/DataDog/datadog-agent/test/new-e2e/windows/agent"
	agenttest "github.com/DataDog/datadog-agent/test/new-e2e/windows/agent/test"
	"github.com/DataDog/datadog-agent/test/new-e2e/windows/hyperv"
	"github.com/DataDog/datadog-agent/test/new-e2e/windows/installer"
	"github.com/cenkalti/backoff"

	"github.com/pkg/sftp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"
)

var revertVM *bool = flag.Bool("revert", false, "Revert the VM before every test")

type testHost struct {
	host         string
	username     string
	password     string
	vmname       string
	snapshot     string
	testhostname string
}

type windowsInstallerSuite struct {
	suite.Suite

	target *testHost

	installer           string
	prevstableinstaller string

	sshclient *ssh.Client

	// test suite output dir
	suiteoutputdir string
	// individual test output dir
	testoutputdir string
}

func TestWindowsInstaller(t *testing.T) {

	// TODO: make all this configurable
	// TODO: use new-e2e/pulumi for provisioning
	prevstableinstaller := "ddagent-cli-7.44.1.msi"
	// testinstaller := "ddagent-cli-7.45.0-rc.5.msi"
	testinstaller := "datadog-agent-7.45.0-rc.5-1.x86_64.msi"

	hosts := []testHost{
		{
			host:         "172.23.180.180:22",
			username:     "user",
			password:     "user",
			vmname:       "Windows 10",
			snapshot:     "ssh",
			testhostname: "Windows 10 client",
		},
		{
			host:         "172.28.86.123:22",
			username:     "DDEV\\Administrator",
			password:     "123!@#QWEqwe",
			vmname:       "Windows 10",
			snapshot:     "ddev-ssh",
			testhostname: "Windows 10 domain client",
		},
		{
			host:         "172.23.238.202:22",
			username:     "DDEV\\Administrator",
			password:     "123!@#QWEqwe",
			vmname:       "Windows Server 2019",
			snapshot:     "ddev-ssh",
			testhostname: "Windows 2019 DC",
		},
		{
			host:         "172.28.89.187:22",
			username:     "DDEV\\Administrator",
			password:     "123!@#QWEqwe",
			vmname:       "Windows Server 2022",
			snapshot:     "ddev-ssh",
			testhostname: "Windows 2022 DC",
		},
	}

	for _, host := range hosts {
		t.Run(host.testhostname, func(t *testing.T) {
			suite.Run(t, &windowsInstallerSuite{
				suiteoutputdir:      filepath.Join("./output", time.Now().Format(time.RFC3339)),
				target:              &host,
				prevstableinstaller: prevstableinstaller,
				installer:           testinstaller,
			})
		})
	}
}

func (s *windowsInstallerSuite) SetupSuite() {
	// create output dir
	os.MkdirAll(s.suiteoutputdir, os.ModePerm)
}

func (s *windowsInstallerSuite) TearDownSuite() {
	fmt.Printf("Output directory: %s\n", s.suiteoutputdir)
}

func (s *windowsInstallerSuite) SetupTest() {
	// create output dir
	s.testoutputdir = filepath.Join(s.suiteoutputdir, s.T().Name())
	os.MkdirAll(s.testoutputdir, os.ModePerm)

	// revert VM
	if *revertVM {
		fmt.Println("Reverting VM")
		err := hyperv.RevertVM(s.target.vmname, s.target.snapshot)
		s.Require().NoError(err)
	}

	// connect to SSH
	sshconfig := &ssh.ClientConfig{
		User: s.target.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.target.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshclient, err := ssh.Dial("tcp", s.target.host, sshconfig)
	s.Require().NoError(err)

	s.sshclient = sshclient
	s.T().Cleanup(func() {
		fmt.Println("closing ssh")
		sshclient.Close()
	})

	output, err := windows.PsExec(s.sshclient, "ipconfig")
	s.Require().NoError(err)
	s.Require().NotEmpty(output)
	fmt.Println(output)
}

// Tests that the agent can be installed
func (s *windowsInstallerSuite) TestDefaultInstall() {
	t, err := NewTester(s.sshclient)
	s.Require().NoError(err)

	err = t.InstallAgent(s.sshclient, s.installer, "",
		filepath.Join(s.testoutputdir, "install.log"))
	s.Require().NoError(err)

	s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))
}

// TC-UPG-DC-001
// Tests that the agent can be upgraded
func (s *windowsInstallerSuite) TestDefaultUpgrade() {
	err := installer.InstallAgentWithDefaultUser(s.sshclient,
		s.prevstableinstaller, "",
		filepath.Join(s.testoutputdir, "install.log"))
	s.Require().NoError(err)

	s.Require().True(AssertDefaultInstalledUser(s.Assert(), s.sshclient))

	orig_version, err := agent.GetVersion(s.sshclient)
	s.Require().NoError(err)
	fmt.Printf("Agent version: %s", orig_version)

	t, err := NewTester(s.sshclient)
	s.Require().NoError(err)

	err = t.UpgradeAgent(s.sshclient, s.installer, "",
		filepath.Join(s.testoutputdir, "upgrade.log"))
	s.Require().NoError(err)

	s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))

	new_version, err := agent.GetVersion(s.sshclient)
	s.Require().NoError(err)
	fmt.Printf("agent version after upgrade: %s", new_version)
	s.Require().NotEqual(orig_version, new_version)
}

// Tests that the agent functions after rollback
func (s *windowsInstallerSuite) TestRollback() {
	err := installer.InstallAgentWithDefaultUser(s.sshclient,
		s.prevstableinstaller, "",
		filepath.Join(s.testoutputdir, "install.log"))
	s.Require().NoError(err)

	s.Require().True(AssertDefaultInstalledUser(s.Assert(), s.sshclient))

	orig_version, err := agent.GetVersion(s.sshclient)
	s.Require().NoError(err)
	fmt.Printf("Agent version: %s", orig_version)

	err = installer.InstallAgent(s.sshclient, s.installer, "WIXFAILWHENDEFERRED=1",
		filepath.Join(s.testoutputdir, "upgrade.log"))
	s.Require().Error(err)

	new_version, err := agent.GetVersion(s.sshclient)
	s.Require().NoError(err)
	fmt.Printf("agent version after rollback: %s", new_version)
	s.Require().Equal(orig_version, new_version)

	// TODO: ideally the installer will ensure the agent is running after rollback,
	//       OG installer doesn't so for now we copy the kitchen tests and start the agent
	_, err = windows.PsExec(s.sshclient, "start-service datadogagent")
	s.Require().NoError(err)

	// ensure agent is running
	err = backoff.Retry(func() error {
		_, err := agent.GetStatus(s.sshclient)
		return err
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(50*time.Millisecond), (1000/50)*300))
	s.Require().NoError(err)
	status, err := agent.GetStatus(s.sshclient)
	s.Require().NoError(err, "agent returns valid json status")
	agenttest.AssertRunningChecks(s.Assert(), status)

	// ensure directories exist
	s.Require().True(AssertInstalledDirectoriesExist(s.Assert(), s.sshclient))
}

// Tests that the agent can be uninstalled
func (s *windowsInstallerSuite) TestUninstall() {
	t, err := NewTester(s.sshclient)
	s.Require().NoError(err)

	err = t.InstallAgent(s.sshclient, s.installer, "",
		filepath.Join(s.testoutputdir, "install.log"))
	s.Require().NoError(err)

	s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))

	err = installer.UninstallAgent(s.sshclient,
		filepath.Join(s.testoutputdir, "uninstall.log"))
	s.Require().NoError(err)

	userexists, err := windows.LocalUserExists(s.sshclient, "ddagentuser")
	s.Require().NoError(err)
	s.Require().True(userexists, "user should still exist after uninstall")

	fmt.Printf("Checking expected paths were removed...")
	sftpclient, err := sftp.NewClient(s.sshclient)
	s.Require().NoError(err)
	defer sftpclient.Close()
	dirs := []string{
		"C:\\Program Files\\Datadog",
	}
	for _, dir := range dirs {
		_, err = sftpclient.Stat(dir)
		s.Require().Error(err, fmt.Sprintf("'%s' shouldn't exist but does", dir))
	}
	fmt.Println("done")
}

func (s *windowsInstallerSuite) TestNPM() {
	tcs := []struct {
		testname          string
		previnstaller     string
		previnstallerargs string
	}{
		// TC-NPM-001
		{"7.44Upgrade", "ddagent-cli-7.44.1.msi", ""},
		// TC-NPM-002
		{"7.44WithNPMUpgrade", "ddagent-cli-7.44.1.msi", "ADDLOCAL=ALL"},
		// TC-NPM-003
		{"NPMInstall", "", ""},
		// TC-NPM-004
		{"NPMReinstall", s.installer, ""},
		// TC-NPM-005
		{"NPMBetaUpgrade", "datadog-agent-7.23.2-beta1-1-x86_64.msi", "ADDLOCAL=ALL"},
	}

	firstTest := true
	for _, npmEnabled := range []bool{true, false} {
		for _, tc := range tcs {
			var tcname string
			if npmEnabled {
				tcname = fmt.Sprintf("%s/NPMEnabled", tc.testname)
			} else {
				tcname = fmt.Sprintf("%s/NPMDisabled", tc.testname)
			}
			s.Run(tcname, func() {
				if !firstTest {
					s.SetupTest()
				}
				firstTest = false

				if tc.previnstaller != "" {
					err := installer.InstallAgentWithDefaultUser(s.sshclient, tc.previnstaller, tc.previnstallerargs,
						filepath.Join(s.testoutputdir, "install.log"))
					s.Require().NoError(err)
				}

				t, err := NewTester(s.sshclient,
					WithExpectNPMRunning(npmEnabled))
				s.Require().NoError(err)

				err = setNetworkConfig(s.sshclient, npmEnabled)
				s.Require().NoError(err)

				logpath := filepath.Join(s.testoutputdir, "install.log")
				if tc.previnstaller != "" {
					logpath = filepath.Join(s.testoutputdir, "upgrade.log")
				}
				err = t.InstallAgent(s.sshclient, s.installer, "", logpath)
				s.Require().NoError(err)

				s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))
			})
		}
	}
}

func (s *windowsInstallerSuite) TestUpgradeChangeUser() {
	username := "testuser"
	password := "123!@#QWEqwe"
	tcs := []struct {
		testname string
		// args passed to installer
		installusername string
		installpassword string
		// args used by test
		createuser     bool
		expectusername string
		createpassword string
	}{
		// TC-UPG-DC-002
		{"NewUser", username, password, false, username, password},
		// TC-UPG-DC-003
		{"ExistingUser", username, password, true, username, password},
		// TC-UPG-DC-004
		{"WrongCredentials", username, "bad" + password + "bad", true, username, password},
	}
	for tc_i, tc := range tcs {
		s.Run(tc.testname, func() {
			if tc_i > 0 {
				s.SetupTest()
			}

			err := installer.InstallAgentWithDefaultUser(s.sshclient,
				s.prevstableinstaller, "",
				filepath.Join(s.testoutputdir, "install.log"))
			s.Require().NoError(err)
			s.Require().True(AssertDefaultInstalledUser(s.Assert(), s.sshclient))

			t, err := NewTester(s.sshclient,
				WithInstallUser(tc.installusername),
				WithInstallPassword(tc.installpassword),
				WithExpectedAgentUserFromUsername(s.sshclient, tc.expectusername, tc.createpassword, tc.createuser))
			s.Require().NoError(err)

			installerr := t.InstallAgent(s.sshclient,
				s.installer, "",
				filepath.Join(s.testoutputdir, "upgrade.log"))

			if tc.installpassword == tc.createpassword {
				// creds are correct, install should succeed
				// account should be created by installer if it does not exist
				s.Require().NoError(installerr)
				s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))
				return
			} else {
				// creds are wrong, install should succeed, but service won't be running
				s.Require().NoError(installerr)
				WithExpectAgentRunning(false)(t)
				s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))
				return
			}
		})
	}
}

func (s *windowsInstallerSuite) TestAgentUser() {
	hostinfo, err := windows.GetHostInfo(s.sshclient)
	s.Require().NoError(err)

	var domainpart string
	var servicedomainpart string
	if hostinfo.IsDomainController() {
		domainpart = windows.NetBIOSName(hostinfo.Domain)
		servicedomainpart = windows.NetBIOSName(hostinfo.Domain)
	} else {
		domainpart = windows.NetBIOSName(hostinfo.Hostname)
		servicedomainpart = "."
	}

	tcs := []struct {
		testname            string
		builtinaccount      bool
		username            string
		expecteddomain      string
		expecteduser        string
		expectedserviceuser string
	}{
		{"user_only", false, "testuser", domainpart, "testuser", fmt.Sprintf("%s\\testuser", servicedomainpart)},
		{"dotslash_user", false, ".\\testuser", domainpart, "testuser", fmt.Sprintf("%s\\testuser", servicedomainpart)},
		{"domain_user", false, fmt.Sprintf("%s\\testuser", domainpart), domainpart, "testuser", fmt.Sprintf("%s\\testuser", servicedomainpart)},
		{"LocalSystem", true, "LocalSystem", "NT AUTHORITY", "SYSTEM", "LocalSystem"},
		{"SYSTEM", true, "SYSTEM", "NT AUTHORITY", "SYSTEM", "LocalSystem"},
	}

	userpassword := "123!@#QWEqwe"
	for tc_i, tc := range tcs {
		s.Run(tc.testname, func() {
			if tc_i > 0 {
				s.SetupTest()
			}

			t, err := NewTester(s.sshclient,
				WithInstallUser(tc.username),
				WithInstallPassword(userpassword),
				WithExpectedAgentUser(tc.expecteddomain, tc.expecteduser, tc.expectedserviceuser))
			s.Require().NoError(err)

			err = t.InstallAgent(s.sshclient,
				s.installer, "",
				filepath.Join(s.testoutputdir, "install.log"))
			s.Require().NoError(err)

			s.Require().True(t.AssertExpectations(s.Assert(), s.sshclient))
		})
	}
}

func setNetworkConfig(client *ssh.Client, npmEnabled bool) error {
	sftpclient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("sftp connection failed: %v", err)
	}
	defer sftpclient.Close()

	configPath := installer.DefaultConfigPath
	err = sftpclient.MkdirAll(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config dir: %v", err)
	}
	err = windows.WriteFile(sftpclient,
		filepath.Join(configPath, "system-probe.yaml"),
		[]byte(fmt.Sprintf("network_config:\n  enabled: %v", npmEnabled)))
	if err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}
	return nil
}
