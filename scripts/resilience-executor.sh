#!/bin/bash
# resilience-executor - Run OkResilience tests from JSON configuration
# 
# This script executes resilience test cases defined in a JSON file.
# It validates gateway resilience by generating traffic and checking metrics.
#
# Usage: ./scripts/resilience-executor [options]
#
# Options:
#   -f, --file <path>      Path to resilience test file (default: resilience-tests.json)
#   -t, --test <name>      Run a specific test by name
#   -r, --retries <num>    Max retries for metrics sync (default: 5)
#   -d, --delay <sec>      Delay between retries in seconds (default: 5)
#   -h, --help             Show this help message

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
EXECUTOR_BINARY="$PROJECT_ROOT/resilience-executor"
TEST_FILE="$PROJECT_ROOT/resilience-tests.json"
MAX_RETRIES=5
DELAY_SECONDS=5
TEST_NAME=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--file)
            TEST_FILE="$2"
            shift 2
            ;;
        -t|--test)
            TEST_NAME="$2"
            shift 2
            ;;
        -r|--retries)
            MAX_RETRIES="$2"
            shift 2
            ;;
        -d|--delay)
            DELAY_SECONDS="$2"
            shift 2
            ;;
        -h|--help)
            grep "^#" "$0" | tail -n +2 | sed 's/^# *//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Build the resilience-executor
echo "🔨 Building resilience-executor..."
cd "$PROJECT_ROOT"
if go build -o resilience-executor ./cmd/test-runner; then
    echo "✓ Build successful"
else
    echo "✗ Build failed"
    exit 1
fi

# Verify binary exists and is executable
if [ ! -x "$EXECUTOR_BINARY" ]; then
    echo "✗ Error: resilience-executor binary not found or not executable"
    exit 1
fi

# Construct arguments
ARGS="--test-file=$TEST_FILE --max-retries=$MAX_RETRIES --delay-seconds=$DELAY_SECONDS"
if [ -n "$TEST_NAME" ]; then
    ARGS="$ARGS --test-name=$TEST_NAME"
fi

# Execute the test runner
"$EXECUTOR_BINARY" $ARGS
