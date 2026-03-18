#!/bin/bash
set -e

# Smoke test linux-amd64 binary from wheel
# Usage: smoke-test-wheel.sh <expected-version>

EXPECTED_VERSION="${1:?Expected version argument}"

WHL=$(ls dist/*linux*x86_64*.whl | head -n1)
unzip -o "$WHL" -d /tmp/wheel_test
BIN=$(find /tmp/wheel_test -type f -name "neo4j-mcp-server" | head -n1)
chmod +x "$BIN"
OUTPUT=$("$BIN" --version)
echo "Binary output: $OUTPUT"
echo "$OUTPUT" | grep -F "$EXPECTED_VERSION"
