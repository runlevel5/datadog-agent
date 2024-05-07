// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package servicedetector

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

const (
	springBootApp = "app/app.jar"
)

func mockfs(t *testing.T) fs.SubFS {
	writer := bytes.NewBuffer(nil)
	createMockSpringBootApp(t, writer)
	// use test data e mock data together
	return &fstest.MapFS{
		springBootApp: &fstest.MapFile{Data: writer.Bytes()},
	}
}

func TestExtractServiceMetadata(t *testing.T) {
	tests := []struct {
		name                       string
		cmdline                    []string
		envs                       []string
		fs                         fs.SubFS
		expectedServiceTag         string
		expectedAdditionalServices []string
	}{
		{
			name:               "empty",
			cmdline:            []string{},
			expectedServiceTag: "",
		},
		{
			name:               "blank",
			cmdline:            []string{""},
			expectedServiceTag: "",
		},
		{
			name: "single arg executable",
			cmdline: []string{
				"./my-server.sh",
			},
			expectedServiceTag: "my-server",
		},
		{
			name: "single arg executable with special chars",
			cmdline: []string{
				"./-my-server.sh-",
			},
			expectedServiceTag: "my-server",
		},
		{
			name: "sudo",
			cmdline: []string{
				"sudo", "-E", "-u", "dog", "/usr/local/bin/myApp", "-items=0,1,2,3", "-foo=bar",
			},
			expectedServiceTag: "myApp",
		},
		{
			name: "python flask argument",
			cmdline: []string{
				"/opt/python/2.7.11/bin/python2.7", "flask", "run", "--host=0.0.0.0",
			},
			expectedServiceTag: "flask",
		},
		{
			name: "python - flask argument in path",
			cmdline: []string{
				"/opt/python/2.7.11/bin/python2.7", "/opt/dogweb/bin/flask", "run", "--host=0.0.0.0", "--without-threads",
			},
			expectedServiceTag: "flask",
		},
		{
			name: "python flask in single argument",
			cmdline: []string{
				"/opt/python/2.7.11/bin/python2.7 flask run --host=0.0.0.0",
			},
			expectedServiceTag: "flask",
		},
		{
			name: "python - module hello",
			cmdline: []string{
				"python3", "-m", "hello",
			},
			expectedServiceTag: "hello",
		},
		{
			name: "ruby - td-agent",
			cmdline: []string{
				"ruby", "/usr/sbin/td-agent", "--log", "/var/log/td-agent/td-agent.log", "--daemon", "/var/run/td-agent/td-agent.pid",
			},
			expectedServiceTag: "td-agent",
		},
		{
			name: "java using the -jar flag to define the service",
			cmdline: []string{
				"java", "-Xmx4000m", "-Xms4000m", "-XX:ReservedCodeCacheSize=256m", "-jar", "/opt/sheepdog/bin/myservice.jar",
			},
			expectedServiceTag: "myservice",
		},
		{
			name: "java class name as service",
			cmdline: []string{
				"java", "-Xmx4000m", "-Xms4000m", "-XX:ReservedCodeCacheSize=256m", "com.datadog.example.HelloWorld",
			},
			expectedServiceTag: "HelloWorld",
		},
		{
			name: "java kafka",
			cmdline: []string{
				"java", "-Xmx4000m", "-Xms4000m", "-XX:ReservedCodeCacheSize=256m", "kafka.Kafka",
			},
			expectedServiceTag: "Kafka",
		},
		{
			name: "java parsing for org.apache projects with cassandra as the service",
			cmdline: []string{
				"/usr/bin/java", "-Xloggc:/usr/share/cassandra/logs/gc.log", "-ea", "-XX:+HeapDumpOnOutOfMemoryError", "-Xss256k", "-Dlogback.configurationFile=logback.xml",
				"-Dcassandra.logdir=/var/log/cassandra", "-Dcassandra.storagedir=/data/cassandra",
				"-cp", "/etc/cassandra:/usr/share/cassandra/lib/HdrHistogram-2.1.9.jar:/usr/share/cassandra/lib/cassandra-driver-core-3.0.1-shaded.jar",
				"org.apache.cassandra.service.CassandraDaemon",
			},
			expectedServiceTag: "cassandra",
		},
		{
			name: "java space in java executable path",
			cmdline: []string{
				"/home/dd/my java dir/java", "com.dog.cat",
			},
			expectedServiceTag: "cat",
		}, {
			name: "node js with package.json not present",
			cmdline: []string{
				"/usr/bin/node",
				"--require",
				"/private/node-patches_legacy/register.js",
				"--preserve-symlinks-main",
				"--",
				"/somewhere/index.js",
			},
			expectedServiceTag: "",
		},
		{
			name: "node js with a broken package.json",
			cmdline: []string{
				"/usr/bin/node",
				"./testdata/inner/index.js",
			},
			expectedServiceTag: "",
		},
		{
			name: "node js with a valid package.json",
			cmdline: []string{
				"/usr/bin/node",
				"--require",
				"/private/node-patches_legacy/register.js",
				"--preserve-symlinks-main",
				"--",
				"./testdata/index.js",
			},
			expectedServiceTag: "my-awesome-package",
		},
		{
			name: "node js with a valid nested package.json and cwd",
			cmdline: []string{
				"/usr/bin/node",
				"--require",
				"/private/node-patches_legacy/register.js",
				"--preserve-symlinks-main",
				"--",
				"index.js",
			},
			envs:               []string{"PWD=testdata/deep"}, // it's relative but it's ok for testing purposes
			expectedServiceTag: "my-awesome-package",
		},
		{
			name: "spring boot default options",
			cmdline: []string{
				"java",
				"-jar",
				"app/app.jar",
			},
			fs:                 mockfs(t),
			expectedServiceTag: "default-app",
		},
		{
			name: "wildfly 18 standalone",
			cmdline: []string{"home/app/.sdkman/candidates/java/17.0.4.1-tem/bin/java",
				"-D[Standalone]",
				"-server",
				"-Xms64m",
				"-Xmx512m",
				"-XX:MetaspaceSize=96M",
				"-XX:MaxMetaspaceSize=256m",
				"-Djava.net.preferIPv4Stack=true",
				"-Djboss.modules.system.pkgs=org.jboss.byteman",
				"-Djava.awt.headless=true",
				"--add-exports=java.base/sun.nio.ch=ALL-UNNAMED",
				"--add-exports=jdk.unsupported/sun.misc=ALL-UNNAMED",
				"--add-exports=jdk.unsupported/sun.reflect=ALL-UNNAMED",
				"-Dorg.jboss.boot.log.file=testdata/jboss/standalone/log/server.log",
				"-Dlogging.configuration=file:testdata/jboss/standalone/configuration/logging.properties",
				"-jar",
				"testdata/jboss/jboss-modules.jar",
				"-mp",
				"testdata/jboss/modules",
				"org.jboss.as.standalone",
				"-Djboss.home.dir=testdata/jboss",
				"-Djboss.server.base.dir=testdata/jboss/standalone"},
			fs:                         realFs{},
			expectedServiceTag:         "jboss-modules",
			expectedAdditionalServices: []string{"my-jboss-webapp", "some_context_root", "web3"},
		},
		{
			name: "wildfly 18 domain",
			cmdline: []string{"/home/app/.sdkman/candidates/java/17.0.4.1-tem/bin/java",
				"--add-exports=java.base/sun.nio.ch=ALL-UNNAMED",
				"--add-exports=jdk.unsupported/sun.reflect=ALL-UNNAMED",
				"--add-exports=jdk.unsupported/sun.misc=ALL-UNNAMED",
				"-D[Server:server-one]",
				"-D[pcid:780891833]",
				"-Xms64m",
				"-Xmx512m",
				"-server",
				"-XX:MetaspaceSize=96m",
				"-XX:MaxMetaspaceSize=256m",
				"-Djava.awt.headless=true",
				"-Djava.net.preferIPv4Stack=true",
				"-Djboss.home.dir=testdata/jboss",
				"-Djboss.modules.system.pkgs=org.jboss.byteman",
				"-Djboss.server.log.dir=testdata/jboss/domain/servers/server-one/log",
				"-Djboss.server.temp.dir=testdata/jboss/domain/servers/server-one/tmp",
				"-Djboss.server.data.dir=testdata/jboss/domain/servers/server-one/data",
				"-Dorg.jboss.boot.log.file=testdata/jboss/domain/servers/server-one/log/server.log",
				"-Dlogging.configuration=file:testdata/jboss/domain/configuration/default-server-logging.properties",
				"-jar",
				"testdata/jboss/jboss-modules.jar",
				"-mp",
				"testdata/jboss/modules",
				"org.jboss.as.server"},
			expectedServiceTag:         "jboss-modules",
			expectedAdditionalServices: []string{"web3", "web4"},
		},
		{
			name: "weblogic 12",
			cmdline: []string{"/u01/jdk/bin/java",
				"-Djava.security.egd=file:/dev/./urandom",
				"-cp",
				"/u01/oracle/wlserver/server/lib/weblogic-launcher.jar",
				"-Dlaunch.use.env.classpath=true",
				"-Dweblogic.Name=AdminServer",
				"-Djava.security.policy=/u01/oracle/wlserver/server/lib/weblogic.policy",
				"-Djava.system.class.loader=com.oracle.classloader.weblogic.LaunchClassLoader",
				"-javaagent:/u01/oracle/wlserver/server/lib/debugpatch-agent.jar",
				"-da",
				"-Dwls.home=/u01/oracle/wlserver/server",
				"-Dweblogic.home=/u01/oracle/wlserver/server",
				"weblogic.Server"},
			envs:                       []string{"PWD=testdata/weblogic"},
			expectedServiceTag:         "Server",
			expectedAdditionalServices: []string{"my_context", "sample4", "some_context_root"},
		},
		{
			name: "java with dd_service as system property",
			cmdline: []string{
				"/usr/bin/java", "-Ddd.service=custom", "-jar", "app.jar",
			},
			expectedServiceTag: "custom",
		},
		{
			name: "Tomcat 10.X",
			cmdline: []string{
				"/usr/bin/java",
				"-Djava.util.logging.config.file=testdata/tomcat/conf/logging.properties",
				"-Djava.util.logging.manager=org.apache.juli.ClassLoaderLogManager",
				"-Djdk.tls.ephemeralDHKeySize=2048",
				"-Djava.protocol.handler.pkgs=org.apache.catalina.webresources",
				"-Dorg.apache.catalina.security.SecurityListener.UMASK=0027",
				"--add-opens=java.base/java.lang=ALL-UNNAMED",
				"--add-opens=java.base/java.io=ALL-UNNAMED",
				"--add-opens=java.base/java.util=ALL-UNNAMED",
				"--add-opens=java.base/java.util.concurrent=ALL-UNNAMED",
				"--add-opens=java.rmi/sun.rmi.transport=ALL-UNNAMED",
				"-classpath",
				"testdata/tomcat/bin/bootstrap.jar:testdata/tomcat/bin/tomcat-juli.jar",
				"-Dcatalina.base=testdata/tomcat",
				"-Dcatalina.home=testdata/tomcat",
				"-Djava.io.tmpdir=testdata/tomcat/temp",
				"org.apache.catalina.startup.Bootstrap",
				"start",
			},
			expectedServiceTag:         "catalina",
			expectedAdditionalServices: []string{"app2", "custom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFs := tt.fs
			if testFs == nil {
				testFs = &realFs{}
			}

			meta, ok := NewWithFS(tt.cmdline, tt.envs, testFs).Detect()
			if len(tt.expectedServiceTag) == 0 {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, tt.expectedServiceTag, meta.Name)
				require.Equal(t, tt.expectedAdditionalServices, meta.AdditionalNames)
			}
		})
	}
}

func writeFile(writer *zip.Writer, name string, content string) error {
	w, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(content))
	return err
}

type chainedFS struct {
	chain []fs.FS
}

func (c chainedFS) Open(name string) (fs.File, error) {
	var err error
	for _, current := range c.chain {
		var f fs.File
		f, err = current.Open(name)
		if err == nil {
			return f, nil
		}
	}
	return nil, err
}

func (c chainedFS) Sub(dir string) (fs.FS, error) {
	for _, current := range c.chain {
		if sub, ok := current.(fs.SubFS); ok {
			return sub.Sub(dir)
		}
	}
	return nil, errors.New("no suitable SubFS in the chain")
}

type shadowFS struct {
	filesystem fs.FS
	parent     fs.FS
	globs      []string
}

func (s shadowFS) Open(name string) (fs.File, error) {
	var fsys fs.FS
	if s.parent != nil {
		fsys = s.parent
	} else {
		fsys = s.filesystem
	}
	for _, current := range s.globs {
		ok, err := path.Match(current, name)
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, fs.ErrNotExist
		}
	}
	return fsys.Open(name)
}

func (s shadowFS) Sub(dir string) (fs.FS, error) {
	fsys, err := fs.Sub(s.filesystem, dir)
	if err != nil {
		return nil, err
	}
	return shadowFS{filesystem: fsys, parent: s}, nil
}
