// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package npm

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/components"
	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/e2e"
	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/environments"
	awsdocker "github.com/DataDog/datadog-agent/test/new-e2e/pkg/environments/aws/docker"

	"github.com/DataDog/test-infra-definitions/components/docker"
	"github.com/DataDog/test-infra-definitions/resources/aws"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/ec2"
)

type dockerHostNginxEnv struct {
	environments.DockerHost
	// Extra Components
	HTTPBinHost *components.RemoteHost
}

type ec2VMContainerizedSuite struct {
	e2e.BaseSuite[dockerHostNginxEnv]
}

func dockerHostHttpbinEnvProvisioner() e2e.PulumiEnvRunFunc[dockerHostNginxEnv] {
	return func(ctx *pulumi.Context, env *dockerHostNginxEnv) error {
		awsEnv, err := aws.NewEnvironment(ctx)
		if err != nil {
			return err
		}
		env.DockerHost.AwsEnvironment = &awsEnv

		opts := []awsdocker.ProvisionerOption{
			awsdocker.WithAgentOptions(systemProbeConfigNPMEnv()...),
		}
		params := awsdocker.GetProvisionerParams(opts...)
		awsdocker.Run(ctx, &env.DockerHost, params)

		vmName := "httpbinvm"

		nginxHost, err := ec2.NewVM(awsEnv, vmName)
		if err != nil {
			return err
		}
		err = nginxHost.Export(ctx, &env.HTTPBinHost.HostOutput)
		if err != nil {
			return err
		}

		// install docker.io
		manager, _, err := docker.NewManager(*awsEnv.CommonEnvironment, nginxHost, true)
		if err != nil {
			return err
		}

		composeContents := []docker.ComposeInlineManifest{dockerHTTPBinCompose()}
		_, err = manager.ComposeStrUp("httpbin", composeContents, pulumi.StringMap{})
		if err != nil {
			return err
		}

		return nil
	}
}

// TestEC2VMSuite will validate running the agent on a single EC2 VM
func TestEC2VMContainerizedSuite(t *testing.T) {
	s := &ec2VMContainerizedSuite{}

	e2eParams := []e2e.SuiteOption{e2e.WithProvisioner(e2e.NewTypedPulumiProvisioner("dockerHostHttpbin", dockerHostHttpbinEnvProvisioner(), nil))}

	// Source of our kitchen CI images test/kitchen/platforms.json
	// Other VM image can be used, our kitchen CI images test/kitchen/platforms.json
	// ec2params.WithImageName("ami-a4dc46db", os.AMD64Arch, ec2os.AmazonLinuxOS) // ubuntu-16-04-4.4
	e2e.Run(t, s, e2eParams...)
}

// SetupSuite
func (v *ec2VMContainerizedSuite) SetupSuite() {
	v.BaseSuite.SetupSuite()

	v.Env().RemoteHost.MustExecute("sudo apt install -y apache2-utils")

	// prefetch docker image locally
	v.Env().RemoteHost.MustExecute("docker pull public.ecr.aws/k8m1l3p1/alpine/curler:latest")
	v.Env().RemoteHost.MustExecute("docker pull public.ecr.aws/docker/library/httpd:latest")
	v.Env().RemoteHost.MustExecute("docker pull public.ecr.aws/patrickc/troubleshoot-util:latest")
}

// BeforeTest will be called before each test
func (v *ec2VMContainerizedSuite) BeforeTest(suiteName, testName string) {
	v.BaseSuite.BeforeTest(suiteName, testName)

	// default is to reset the current state of the fakeintake aggregators
	if !v.BaseSuite.IsDevMode() {
		v.Env().FakeIntake.Client().FlushServerAndResetAggregators()
	}
}

// TestFakeIntakeNPMHostRequests Validate the agent can communicate with the (fake) backend and send connections every 30 seconds
// 2 tests generate the request on the host and on docker
//   - looking for 1 host to send CollectorConnections payload to the fakeintake
//   - looking for 3 payloads and check if the last 2 have a span of 30s +/- 500ms
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM_HostRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate a connection
	v.Env().RemoteHost.MustExecute("curl " + testURL)

	test1HostFakeIntakeNPM(&v.BaseSuite, v.Env().FakeIntake)
}

// TestFakeIntakeNPMDockerRequests Validate the agent can communicate with the (fake) backend and send connections every 30 seconds
// 2 tests generate the request on the host and on docker
//   - looking for 1 host to send CollectorConnections payload to the fakeintake
//   - looking for 3 payloads and check if the last 2 have a span of 30s +/- 500ms
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM_DockerRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate a connection
	v.Env().RemoteHost.MustExecute("docker run public.ecr.aws/k8m1l3p1/alpine/curler curl " + testURL)

	test1HostFakeIntakeNPM(&v.BaseSuite, v.Env().FakeIntake)
}

// TestFakeIntakeNPM600cnxBucket_HostRequests Validate the agent can communicate with the (fake) backend and send connections
// every 30 seconds with a maximum of 600 connections per payloads, if more another payload will follow.
//   - looking for 1 host to send CollectorConnections payload to the fakeintake
//   - looking for n payloads and check if the last 2 have a maximum span of 100ms
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM600cnxBucket_HostRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate connections
	v.Env().RemoteHost.MustExecute("ab -n 600 -c 600 " + testURL)

	test1HostFakeIntakeNPM600cnxBucket(&v.BaseSuite, v.Env().FakeIntake)
}

// TestFakeIntakeNPM600cnxBucket_DockerRequests Validate the agent can communicate with the (fake) backend and send connections
// every 30 seconds with a maximum of 600 connections per payloads, if more another payload will follow.
//   - looking for 1 host to send CollectorConnections payload to the fakeintake
//   - looking for n payloads and check if the last 2 have a maximum span of 100ms
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM600cnxBucket_DockerRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate connections
	v.Env().RemoteHost.MustExecute("docker run public.ecr.aws/docker/library/httpd ab -n 600 -c 600 " + testURL)

	test1HostFakeIntakeNPM600cnxBucket(&v.BaseSuite, v.Env().FakeIntake)
}

// TestFakeIntakeNPM_TCP_UDP_DNS_HostRequests validate we received tcp, udp, and DNS connections
// with some basic checks, like IPs/Ports present, DNS query has been captured, ...
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM_TCP_UDP_DNS_HostRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate connections
	v.Env().RemoteHost.MustExecute("curl " + testURL)
	v.Env().RemoteHost.MustExecute("dig @8.8.8.8 www.google.ch")

	test1HostFakeIntakeNPMTCPUDPDNS(&v.BaseSuite, v.Env().FakeIntake)
}

// TestFakeIntakeNPM_TCP_UDP_DNS_DockerRequests validate we received tcp, udp, and DNS connections
// with some basic checks, like IPs/Ports present, DNS query has been captured, ...
func (v *ec2VMContainerizedSuite) TestFakeIntakeNPM_TCP_UDP_DNS_DockerRequests() {
	testURL := "http://" + v.Env().HTTPBinHost.Address + "/"

	// generate connections
	v.Env().RemoteHost.MustExecute("docker run public.ecr.aws/k8m1l3p1/alpine/curler curl " + testURL)
	v.Env().RemoteHost.MustExecute("docker run public.ecr.aws/patrickc/troubleshoot-util dig @8.8.8.8 www.google.ch")

	test1HostFakeIntakeNPMTCPUDPDNS(&v.BaseSuite, v.Env().FakeIntake)
}
