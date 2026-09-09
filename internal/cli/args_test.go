// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

const (
	testVersion     = "1.0.0"
	testProgramName = "neo4j-mcp"
	testHelpText    = "neo4j-mcp - Neo4j Model Context Protocol Server"
	testVersionText = "neo4j-mcp version: 1.0.0"
)

// captureOutput temporarily redirects stdout and stderr to capture output.
func captureOutput(fn func()) (stdout, stderr string) {
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	return string(outBytes), string(errBytes)
}

// exitMock captures os.Exit calls for testing.
type exitMock struct {
	called bool
	code   int
}

// mockExit records the exit call and panics to stop execution.
func (m *exitMock) Exit(code int) {
	m.called = true
	m.code = code
	panic(m)
}

func TestHandleArgs(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		version          string
		expectedExitCode int
		expectedOutput   string
		expectedStderr   string
	}{
		{
			name:             "no flags",
			args:             []string{testProgramName},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "no flags",
			args:             []string{testProgramName},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "version flag short form",
			args:             []string{testProgramName, "-v"},
			version:          testVersion,
			expectedExitCode: 0,
			expectedOutput:   testVersionText,
		},
		{
			name:             "version flag long form",
			args:             []string{testProgramName, "--version"},
			version:          testVersion,
			expectedExitCode: 0,
			expectedOutput:   testVersionText,
		},
		{
			name:             "help flag short form",
			args:             []string{testProgramName, "-h"},
			version:          testVersion,
			expectedExitCode: 0,
			expectedOutput:   testHelpText,
		},
		{
			name:             "help flag long form",
			args:             []string{testProgramName, "--help"},
			version:          testVersion,
			expectedExitCode: 0,
			expectedOutput:   testHelpText,
		},
		{
			name:             "unknown flag",
			args:             []string{testProgramName, "-x"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "unknown flag or argument: -x",
		},
		{
			name:             "version flag with extra arguments",
			args:             []string{testProgramName, "-v", "extra"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "unknown flag or argument: extra",
		},
		{
			name:             "version flag at end",
			args:             []string{testProgramName, "extra", "-v"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "unknown flag or argument: extra",
		},
		{
			name:             "help and version flags together - help takes precedence",
			args:             []string{testProgramName, "-v", "-h"},
			version:          testVersion,
			expectedExitCode: 0,
			expectedOutput:   testHelpText,
		},
		{
			name:             "help flag at end",
			args:             []string{testProgramName, "extra", "-h"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "unknown flag or argument: extra",
		},
		{
			name:             "uri",
			args:             []string{testProgramName, "--uri", "bolt://localhost:7687"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "username",
			args:             []string{testProgramName, "--username", "neo4j"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "password",
			args:             []string{testProgramName, "--password", "password"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "database",
			args:             []string{testProgramName, "--database", "neo4j"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "read-only",
			args:             []string{testProgramName, "--read-only", "true"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "telemetry",
			args:             []string{testProgramName, "--telemetry", "false"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "schema sample size",
			args:             []string{testProgramName, "--schema-sample-size", "500"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "transport mode",
			args:             []string{testProgramName, "--transport-mode", "http"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http port",
			args:             []string{testProgramName, "--http-port", "8443"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http host",
			args:             []string{testProgramName, "--http-host", "localhost"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http allowed origins",
			args:             []string{testProgramName, "--http-allowed-origins", "https://example.com"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http tls enabled",
			args:             []string{testProgramName, "--http-tls-enabled", "true"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http tls cert file",
			args:             []string{testProgramName, "--http-tls-cert-file", "cert.pem"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http tls key file",
			args:             []string{testProgramName, "--http-tls-key-file", "key.pem"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http auth header name",
			args:             []string{testProgramName, "--http-auth-header-name", "X-Custom-Auth"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http unauthenticated ping",
			args:             []string{testProgramName, "--http-allow-unauthenticated-ping", "true"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "http unauthenticated tools list",
			args:             []string{testProgramName, "--http-allow-unauthenticated-tools-list", "true"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "tools flag with one tool name",
			args:             []string{testProgramName, "--tools", "get-schema"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "tools flag with trailing comma after tool name",
			args:             []string{testProgramName, "--tools", "get-schema,"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "tools flag with two tool names",
			args:             []string{testProgramName, "--tools", "get-schema,read-cypher"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "request-timeout flag with valid value",
			args:             []string{testProgramName, "--request-timeout", "45s"},
			version:          testVersion,
			expectedExitCode: -1,
		},
		{
			name:             "request-timeout flag missing value",
			args:             []string{testProgramName, "--request-timeout"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--request-timeout requires a value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			originalOsExit := osExit
			t.Cleanup(func() {
				os.Args = originalArgs
				osExit = originalOsExit
			})

			os.Args = tt.args
			mock := &exitMock{}
			osExit = mock.Exit

			stdout, stderr := captureOutput(func() {
				defer func() {
					if r := recover(); r != mock {
						if r != nil {
							panic(r)
						}
					}
				}()
				HandleArgs(tt.version)
			})

			shouldExit := tt.expectedExitCode != -1
			if shouldExit != mock.called {
				t.Errorf("exit called: got %v, want %v", mock.called, shouldExit)
			}
			if mock.called && mock.code != tt.expectedExitCode {
				t.Errorf("exit code: got %d, want %d", mock.code, tt.expectedExitCode)
			}
			if tt.expectedStderr != "" && !strings.Contains(stderr, tt.expectedStderr) {
				t.Errorf("stderr: got %q, want to contain %q", stderr, tt.expectedStderr)
			}
			if tt.expectedOutput != "" && !strings.Contains(stdout, tt.expectedOutput) {
				t.Errorf("stdout: got %q, want to contain %q", stdout, tt.expectedOutput)
			}
		})
	}
}
