//go:build e2e

package helpers

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildServer compile the project and store the output binary in a TempDir,
// Returns a callback to delete the temporary directory when is is no longer needed.
func BuildServer() (string, func(), error) {
	// Create temporary directory for the build
	tmpDir := os.TempDir()

	// Create a unique subdirectory for this test run
	buildDir, err := os.MkdirTemp(tmpDir, "mcp-server-test-*")
	if err != nil {
		return "", nil, err
	}

	// Define cleanup function
	cleanup := func() {
		if err := os.RemoveAll(buildDir); err != nil {
			log.Printf("failed to cleanup build directory: %v", err)
		}
	}

	// Define binary path
	binaryName := "neo4j-mcp"
	binaryPath := filepath.Join(buildDir, binaryName)

	// Get the project root directory (go up from test/e2e/)
	projectRoot := filepath.Join("..", "..")

	// Build the server binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(projectRoot, "cmd", "neo4j-mcp")
	cmd.Env = os.Environ() // Use current environment

	// Capture build output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, err
	}

	log.Printf("Built server binary at: %s", binaryPath)
	if len(output) > 0 {
		log.Printf("Build output: %s", string(output))
	}

	// Verify the binary was created, if not cleanup and return
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}
