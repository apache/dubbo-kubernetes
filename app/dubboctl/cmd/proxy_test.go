//go:build !windows
// +build !windows

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

import (
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

import (
	dubbo_cmd "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestCmd(t *testing.T) {
	test.RunSpecs(t, "cmd Suite")
}

var _ = Describe("proxy", func() {
	var cancel func()
	var ctx context.Context
	_ = dubbo_cmd.RunCmdOpts{
		SetupSignalHandler: func() (context.Context, context.Context) {
			return ctx, ctx
		},
	}
	var tmpDir string
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		tmpDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		if tmpDir != "" {
			if tmpDir != "" {
				// when
				err := os.RemoveAll(tmpDir)
				// then
				Expect(err).ToNot(HaveOccurred())
			}
		}
	})
	type testCase struct {
		envVars      map[string]string
		args         []string
		expectedFile string
	}
	DescribeTable("should be possible to start dataplane (Envoy) using `dubbo-proxy run`",
		func(giveFunc func() testCase) {
			given := giveFunc()

			// setup
			envoyPidFile := filepath.Join(tmpDir, "envoy-mock.pid")
			envoyCmdlineFile := filepath.Join(tmpDir, "envoy-mock.cmdline")
			corednsPidFile := filepath.Join(tmpDir, "coredns-mock.pid")
			corednsCmdlineFile := filepath.Join(tmpDir, "coredns-mock.cmdline")

			// and
			env := given.envVars
			env["ENVOY_MOCK_PID_FILE"] = envoyPidFile
			env["ENVOY_MOCK_CMDLINE_FILE"] = envoyCmdlineFile
			env["COREDNS_MOCK_PID_FILE"] = corednsPidFile
			env["COREDNS_MOCK_CMDLINE_FILE"] = corednsCmdlineFile
			for key, value := range env {
				Expect(os.Setenv(key, value)).To(Succeed())
			}

			// given

			reader, writer := io.Pipe()
			go func() {
				defer GinkgoRecover()
				io.ReadAll(reader)
			}()

			cmd := getRootCmd([]string{"proxy", "--proxy-type=ingress", "--dataplane-file=/mnt/d/code/go/test/1.yaml"})
			cmd.SetOut(writer)
			cmd.SetErr(writer)
			cancel()

			// when
			By("starting the Dubbo proxy")
			errCh := make(chan error)
			go func() {
				defer close(errCh)
				errCh <- cmd.Execute()
			}()

			// then
			var actualConfigFile string
			envoyPid := verifyComponentProcess("Envoy", envoyPidFile, envoyCmdlineFile, func(actualArgs []string) {
				Expect(actualArgs[0]).To(Equal("--version"))
				Expect(actualArgs[1]).To(Equal("--config-path"))
				actualConfigFile = actualArgs[2]
				Expect(actualConfigFile).To(BeARegularFile())
				if given.expectedFile != "" {
					Expect(actualArgs[2]).To(Equal(given.expectedFile))
				}
			})

			err := <-errCh
			Expect(err).ToNot(HaveOccurred())

			By("waiting for dataplane (Envoy) to get stopped")
			Eventually(func() bool {
				//send sig 0 to check whether Envoy process still exists
				err := syscall.Kill(int(envoyPid), syscall.Signal(0))
				// we expect Envoy process to get killed by now
				return err != nil
			}, "5s", "100ms").Should(BeTrue())

		},
		Entry("can be launched with env vars", func() testCase {
			return testCase{
				envVars: map[string]string{
					"DUBBO_CONTROL_PLANE_API_SERVER_URL":  "http://localhost:1234",
					"DUBBO_DATAPLANE_NAME":                "example",
					"DUBBO_DATAPLANE_MESH":                "default",
					"DUBBO_DATAPLANE_RUNTIME_BINARY_PATH": filepath.Join("testdata", "envoy-mock.sleep.sh"),
					// Notice: DUBBO_DATAPLANE_RUNTIME_CONFIG_DIR is not set in order to let `dubbo-dp` to create a temporary directory
					"DUBBO_DNS_CORE_DNS_BINARY_PATH": filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				args:         []string{},
				expectedFile: "",
			}
		}),
		Entry("can be launched with env vars and given config dir", func() testCase {
			return testCase{
				envVars: map[string]string{
					"DUBBO_CONTROL_PLANE_API_SERVER_URL":  "http://localhost:1234",
					"DUBBO_DATAPLANE_NAME":                "example",
					"DUBBO_DATAPLANE_MESH":                "default",
					"DUBBO_DATAPLANE_RUNTIME_BINARY_PATH": filepath.Join("testdata", "envoy-mock.sleep.sh"),
					"DUBBO_DATAPLANE_RUNTIME_CONFIG_DIR":  tmpDir,
					"DUBBO_DNS_CORE_DNS_BINARY_PATH":      filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				args:         []string{},
				expectedFile: filepath.Join(tmpDir, "bootstrap.yaml"),
			}
		}),
		Entry("can be launched with args", func() testCase {
			return testCase{
				envVars: map[string]string{},
				args: []string{
					"--cp-address", "http://localhost:1234",
					"--name", "example",
					"--mesh", "default",
					"--binary-path", filepath.Join("testdata", "envoy-mock.sleep.sh"),
					// Notice: --config-dir is not set in order to let `dubbo-dp` to create a temporary directory
					"--dns-coredns-path", filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				expectedFile: "",
			}
		}),
		Entry("can be launched with args and given config dir", func() testCase {
			return testCase{
				envVars: map[string]string{},
				args: []string{
					"--cp-address", "http://localhost:1234",
					"--name", "example",
					"--mesh", "default",
					"--binary-path", filepath.Join("testdata", "envoy-mock.sleep.sh"),
					"--config-dir", tmpDir,
					"--dns-coredns-path", filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				expectedFile: filepath.Join(tmpDir, "bootstrap.yaml"),
			}
		}),
		Entry("can be launched with args and dataplane token", func() testCase {
			return testCase{
				envVars: map[string]string{},
				args: []string{
					"--cp-address", "http://localhost:1234",
					"--name", "example",
					"--mesh", "default",
					"--binary-path", filepath.Join("testdata", "envoy-mock.sleep.sh"),
					"--dataplane-token-file", filepath.Join("testdata", "token"),
					// Notice: --config-dir is not set in order to let `dubbo-dp` to create a temporary directory
					"--dns-coredns-path", filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				expectedFile: "",
			}
		}),
		Entry("can be launched without Envoy Admin API (env vars)", func() testCase {
			return testCase{
				envVars: map[string]string{
					"DUBBO_CONTROL_PLANE_API_SERVER_URL":  "http://localhost:1234",
					"DUBBO_DATAPLANE_NAME":                "example",
					"DUBBO_DATAPLANE_MESH":                "default",
					"DUBBO_DATAPLANE_RUNTIME_BINARY_PATH": filepath.Join("testdata", "envoy-mock.sleep.sh"),
					// Notice: DUBBO_DATAPLANE_RUNTIME_CONFIG_DIR is not set in order to let `dubbo-dp` to create a temporary directory
					"DUBBO_DNS_CORE_DNS_BINARY_PATH": filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				args:         []string{},
				expectedFile: "",
			}
		}),
		Entry("can be launched without Envoy Admin API (command-line args)", func() testCase {
			return testCase{
				envVars: map[string]string{},
				args: []string{
					"--cp-address", "http://localhost:1234",
					"--name", "example",
					"--mesh", "default",
					"--binary-path", filepath.Join("testdata", "envoy-mock.sleep.sh"),
					// Notice: --config-dir is not set in order to let `dubbo-dp` to create a temporary directory
					"--dns-coredns-path", filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				expectedFile: "",
			}
		}),
		Entry("can be launched with dataplane template", func() testCase {
			return testCase{
				envVars: map[string]string{},
				args: []string{
					"--cp-address", "http://localhost:1234",
					"--binary-path", filepath.Join("testdata", "envoy-mock.sleep.sh"),
					"--dataplane-token-file", filepath.Join("testdata", "token"),
					"--dataplane-file", filepath.Join("testdata", "dataplane_template.yaml"),
					"--dataplane-var", "name=example",
					"--dataplane-var", "address=127.0.0.1",
					"--dns-coredns-path", filepath.Join("testdata", "coredns-mock.sleep.sh"),
				},
				expectedFile: "",
			}
		}),
	)

})

func verifyComponentProcess(processDescription, pidfile string, cmdlinefile string, argsVerifier func(expectedArgs []string)) int64 {
	var pid int64
	By(fmt.Sprintf("waiting for dataplane (%s) to get started", processDescription))
	Eventually(func() bool {
		data, err := os.ReadFile(pidfile)
		if err != nil {
			return false
		}
		pid, err = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 32)
		return err == nil
	}, "5s", "100ms").Should(BeTrue())
	Expect(pid).ToNot(BeZero())

	By(fmt.Sprintf("verifying the arguments %s was launched with", processDescription))
	// when
	cmdline, err := os.ReadFile(cmdlinefile)

	// then
	Expect(err).ToNot(HaveOccurred())
	// and
	if argsVerifier != nil {
		actualArgs := strings.FieldsFunc(string(cmdline), func(c rune) bool {
			return c == '\n'
		})
		argsVerifier(actualArgs)
	}
	return pid
}
