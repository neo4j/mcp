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
