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
		expectedExitCode int    // -1 means no exit, 0 or 1 for exit codes
		expectedOutput   string // substring to find in stdout or stderr
		expectedStderr   string // substring to find in stderr (if non-empty, output is checked in stderr instead of stdout)
	}{
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
			name:             "neo4j-uri configuration flag",
			args:             []string{testProgramName, "--neo4j-uri", "bolt://localhost:7687"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "multiple configuration flags",
			args:             []string{testProgramName, "--neo4j-uri", "bolt://localhost:7687", "--neo4j-username", "user"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flags are allowed
		},
		{
			name:             "configuration flag missing value - at end",
			args:             []string{testProgramName, "--neo4j-uri"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-uri requires a value",
		},
		{
			name:             "configuration flag missing value - followed by another flag",
			args:             []string{testProgramName, "--neo4j-uri", "--neo4j-username", "user"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-uri requires a value (got flag --neo4j-username instead)",
		},
		{
			name:             "neo4j-password missing value",
			args:             []string{testProgramName, "--neo4j-password"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-password requires a value",
		},
		{
			name:             "neo4j-database missing value - followed by another flag",
			args:             []string{testProgramName, "--neo4j-database", "--neo4j-uri", "bolt://localhost"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-database requires a value (got flag --neo4j-uri instead)",
		},
		{
			name:             "configuration flags with valid values",
			args:             []string{testProgramName, "--neo4j-uri", "bolt://localhost:7687", "--neo4j-username", "neo4j", "--neo4j-password", "password", "--neo4j-database", "neo4j"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit
		},
		{
			name:             "schema sample size flag with valid value",
			args:             []string{testProgramName, "--neo4j-schema-sample-size", "500"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit
		},
		{
			name:             "schema sample size flag missing value",
			args:             []string{testProgramName, "--neo4j-schema-sample-size"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-schema-sample-size requires a value",
		},
		{
			name:             "transport mode flag valid value",
			args:             []string{testProgramName, "--neo4j-transport-mode", "http"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "transport mode flag missing value",
			args:             []string{testProgramName, "--neo4j-transport-mode"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-transport-mode requires a value",
		},
		{
			name:             "transport mode flag missing value followed by another flag",
			args:             []string{testProgramName, "--neo4j-transport-mode", "--neo4j-uri", "bolt://localhost:7687"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-transport-mode requires a value (got flag --neo4j-uri instead)",
		},
		{
			name:             "http tls enabled flag with valid value",
			args:             []string{testProgramName, "--neo4j-http-tls-enabled", "true"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http tls enabled flag missing value",
			args:             []string{testProgramName, "--neo4j-http-tls-enabled"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-tls-enabled requires a value",
		},
		{
			name:             "http tls cert file flag with valid value",
			args:             []string{testProgramName, "--neo4j-http-tls-cert-file", "/path/to/cert.pem"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http tls cert file flag missing value",
			args:             []string{testProgramName, "--neo4j-http-tls-cert-file"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-tls-cert-file requires a value",
		},
		{
			name:             "http tls key file flag with valid value",
			args:             []string{testProgramName, "--neo4j-http-tls-key-file", "/path/to/key.pem"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http tls key file flag missing value",
			args:             []string{testProgramName, "--neo4j-http-tls-key-file"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-tls-key-file requires a value",
		},
		{
			name:             "http allowed origins flag with valid value",
			args:             []string{testProgramName, "--neo4j-http-allowed-origins", "https://example.com"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http allowed origins flag with multiple origins",
			args:             []string{testProgramName, "--neo4j-http-allowed-origins", "https://example.com,https://example2.com"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http allowed origins flag missing value",
			args:             []string{testProgramName, "--neo4j-http-allowed-origins"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-allowed-origins requires a value",
		},
		{
			name:             "http auth header name flag with valid value",
			args:             []string{testProgramName, "--neo4j-http-auth-header-name", "X-Custom-Auth"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http auth header name flag missing value",
			args:             []string{testProgramName, "--neo4j-http-auth-header-name"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-auth-header-name requires a value",
		},
		{
			name:             "double dash separator stops flag processing",
			args:             []string{testProgramName, "--", "--unknown-flag"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, -- stops our flag processing
		},
		{
			name:             "double dash separator with config flags before it",
			args:             []string{testProgramName, "--neo4j-uri", "bolt://localhost:7687", "--", "--unknown-flag"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, config flag before -- is valid
		},
		{
			name:             "http allow unauthenticated ping with valid value",
			args:             []string{testProgramName, "--neo4j-http-allow-unauthenticated-ping", "true"},
			version:          testVersion,
			expectedExitCode: -1, // Should not exit, flag is allowed
		},
		{
			name:             "http allow unauthenticated ping with missing value",
			args:             []string{testProgramName, "--neo4j-http-allow-unauthenticated-ping"},
			version:          testVersion,
			expectedExitCode: 1,
			expectedStderr:   "--neo4j-http-allow-unauthenticated-ping requires a value",
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

			// Verify exit behaviour
			shouldExit := tt.expectedExitCode != -1
			if shouldExit != mock.called {
				t.Errorf("exit called: got %v, want %v", mock.called, shouldExit)
			}

			if mock.called && mock.code != tt.expectedExitCode {
				t.Errorf("exit code: got %d, want %d", mock.code, tt.expectedExitCode)
			}

			// Verify stderr output
			if tt.expectedStderr != "" {
				if !strings.Contains(stderr, tt.expectedStderr) {
					t.Errorf("stderr: got %q, want to contain %q", stderr, tt.expectedStderr)
				}
			}

			// Verify output
			if tt.expectedOutput != "" {
				if !strings.Contains(stdout, tt.expectedOutput) {
					t.Errorf("stdout: got %q, want to contain %q", stdout, tt.expectedOutput)
				}
			}
		})
	}
}
