#!/bin/bash
# Test script to run the end-to-end assume command test

set -e

echo "Running Assume Command E2E Integration Test"
echo "==========================================="

# Check if E2E testing is enabled
if [ "$GRANTED_E2E_TESTING" != "true" ]; then
    echo "E2E testing is not enabled"
    echo "Set GRANTED_E2E_TESTING=true to run these tests"
    exit 0
fi

# Check if binary path is provided
if [ ! -z "$GRANTED_BINARY_PATH" ]; then
    echo "Using pre-built binary: $GRANTED_BINARY_PATH"
else
    echo "No binary path provided, test will build its own"
fi

# Run the specific E2E test
echo "Testing assume command with mock server..."
GRANTED_E2E_TESTING=true go test -v -run TestAssumeCommandE2E ./pkg/integration_testing/...

echo ""
echo "Test completed successfully!"
echo ""
echo "This test:"
echo "1. Uses pre-built binary (if GRANTED_BINARY_PATH is set) or builds one"
echo "2. Sets up a mock AWS environment"
echo "3. Runs the assume command"
echo "4. Verifies credentials are output correctly"
echo ""
echo "To run with pre-built binary:"
echo "  GRANTED_E2E_TESTING=true GRANTED_BINARY_PATH=/path/to/dgranted ./test_e2e.sh"
echo ""
echo "In CI, this runs as part of the main test workflow in .github/workflows/test.yml"