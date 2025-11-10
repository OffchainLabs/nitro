#!/usr/bin/env bash
# Automation script for running Nethermind comparison system tests
# Usage: ./scripts/run-comparison-test.sh [--verbose] [--timeout SECONDS]

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
NETHERMIND_REPO="/Users/daniilankusin/RiderProjects/nethermind-arbitrum"
NITRO_REPO="/Users/daniilankusin/GolandProjects/arbitrum-nitro"
NETHERMIND_LOG="${NETHERMIND_REPO}/.data/logs/arbitrum-system-test.log"
INIT_TIMEOUT=60  # seconds to wait for Nethermind initialization
TEST_TIMEOUT=120 # seconds for test execution
VERBOSE=false

# Process command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --timeout|-t)
            TEST_TIMEOUT="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose        Show all logs (including filtered ones)"
            echo "  -t, --timeout SECS   Set test timeout in seconds (default: 600)"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Cleanup function
cleanup() {
    local exit_code=$?
    echo ""
    echo -e "${YELLOW}[CLEANUP]${NC} Stopping processes..."

    # Kill Nethermind (dotnet process)
    if pgrep -f "dotnet.*nethermind.dll" > /dev/null; then
        echo -e "${YELLOW}[CLEANUP]${NC} Stopping Nethermind..."
        pkill -TERM -f "dotnet.*nethermind.dll" 2>/dev/null || true
        sleep 2
        # Force kill if still running
        if pgrep -f "dotnet.*nethermind.dll" > /dev/null; then
            echo -e "${YELLOW}[CLEANUP]${NC} Force killing Nethermind..."
            pkill -KILL -f "dotnet.*nethermind.dll" 2>/dev/null || true
        fi
    fi

    # Kill any go test processes
    if pgrep -f "go.*test.*system_tests" > /dev/null; then
        echo -e "${YELLOW}[CLEANUP]${NC} Stopping Nitro test..."
        pkill -TERM -f "go.*test.*system_tests" 2>/dev/null || true
        sleep 2
        # Force kill if still running
        if pgrep -f "go.*test.*system_tests" > /dev/null; then
            echo -e "${YELLOW}[CLEANUP]${NC} Force killing Nitro test..."
            pkill -KILL -f "go.*test.*system_tests" 2>/dev/null || true
        fi
    fi

    echo -e "${GREEN}[CLEANUP]${NC} Cleanup complete"
    exit $exit_code
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

# Create awk filter script for log processing
create_log_filter() {
    if [ "$VERBOSE" = true ]; then
        # Verbose mode: just strip ANSI escape codes that aren't complete
        awk '{ print }'
    else
        # Filtered mode: use awk to filter and colorize
        awk -v RED="$RED" -v GREEN="$GREEN" -v YELLOW="$YELLOW" -v BLUE="$BLUE" -v MAGENTA="$MAGENTA" -v NC="$NC" '
        BEGIN {
            in_diff = 0
        }

        # Skip noisy patterns
        /ld: warning:/ { next }
        /Address 0xA4B05.*Normalized odd number/ { next }
        /Failed to load snapshot.*missing or corrupted/ { next }
        /Getting file info.*stat.*no such file/ { next }
        /Head block is not reachable/ { next }
        /^INFO.*Address 0xA4B05/ { next }
        /^WARN.*Failed to load snapshot/ { next }
        /^WARN.*Getting file info/ { next }

        # Detect start of diff block
        /Diff details/ || /ERROR.*mismatch/ {
            in_diff = 1
            gsub(/ERROR/, RED "ERROR" NC)
            gsub(/mismatch/, MAGENTA "mismatch" NC)
            print
            next
        }

        # Inside diff block - show all structured content
        in_diff == 1 {
            # Check if line looks like diff content (indented, has diff markers, or struct syntax)
            if (match($0, /^[[:space:]]+/) || match($0, /^[-+]/) ||
                match($0, /BlockHash|SendRoot|&execution\.|common\.|string\(|Inverse|^\}/)) {

                # Colorize diff markers
                if (match($0, /^[-]/)) {
                    print RED $0 NC
                } else if (match($0, /^[+]/)) {
                    print GREEN $0 NC
                } else {
                    # Colorize field names
                    gsub(/BlockHash/, MAGENTA "BlockHash" NC)
                    gsub(/SendRoot/, MAGENTA "SendRoot" NC)
                    print
                }
                next
            } else {
                # Exit diff block
                in_diff = 0
            }
        }

        # Show important patterns
        /RPC|DigestInitMessage|DigestMessage|SetFinalityData|CompareExecutionClient/ ||
        /FAIL|PASS/ ||
        /fatal|panic/ ||
        /assertion.*failed/ ||
        /state root/ ||
        /execution mismatch/ ||
        /common_test.go.*error occurred/ {
            # Colorize important keywords
            gsub(/ERROR|Error/, RED "&" NC)
            gsub(/PASS|Success|success/, GREEN "&" NC)
            gsub(/FAIL|Failed|failed/, RED "&" NC)
            gsub(/mismatch|divergence/, MAGENTA "&" NC)
            gsub(/fatal|panic/, RED "&" NC)
            print
            next
        }
        '
    fi
}

echo -e "${BLUE}╔════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Nethermind Comparison Test Automation Runner     ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}[CONFIG]${NC} Nethermind repo: ${NETHERMIND_REPO}"
echo -e "${BLUE}[CONFIG]${NC} Nitro repo: ${NITRO_REPO}"
echo -e "${BLUE}[CONFIG]${NC} Init timeout: ${INIT_TIMEOUT}s"
echo -e "${BLUE}[CONFIG]${NC} Test timeout: ${TEST_TIMEOUT}s"
echo -e "${BLUE}[CONFIG]${NC} Verbose mode: ${VERBOSE}"
echo ""

# Step 1: Start Nethermind
echo -e "${YELLOW}[STEP 1/4]${NC} Starting Nethermind..."
cd "${NETHERMIND_REPO}"

# Clean old log file if exists
if [ -f "${NETHERMIND_LOG}" ]; then
    rm -f "${NETHERMIND_LOG}"
fi

# Start Nethermind in background
make clean-run-system-test > /dev/null 2>&1 &
NETHERMIND_PID=$!

echo -e "${GREEN}[STEP 1/4]${NC} Nethermind started (PID: ${NETHERMIND_PID})"

# Step 2: Wait for Nethermind initialization
echo -e "${YELLOW}[STEP 2/4]${NC} Waiting for Nethermind RPC initialization..."

INIT_START=$(date +%s)
INITIALIZED=false

while [ $(($(date +%s) - INIT_START)) -lt $INIT_TIMEOUT ]; do
    if [ -f "${NETHERMIND_LOG}" ]; then
        if grep -q "JSON RPC.*http://127.0.0.1:20545" "${NETHERMIND_LOG}"; then
            INITIALIZED=true
            break
        fi

        # Check for initialization errors
        if grep -qiE "(fatal|panic|exception.*startup)" "${NETHERMIND_LOG}"; then
            echo -e "${RED}[STEP 2/4]${NC} Nethermind initialization failed!"
            echo -e "${RED}[ERROR]${NC} Check log file: ${NETHERMIND_LOG}"
            tail -20 "${NETHERMIND_LOG}"
            exit 2
        fi
    fi

    # Show progress indicator
    echo -ne "${YELLOW}[STEP 2/4]${NC} Waiting... $(($(date +%s) - INIT_START))s / ${INIT_TIMEOUT}s\r"
    sleep 1
done

if [ "$INITIALIZED" = false ]; then
    echo -e "${RED}[STEP 2/4]${NC} Timeout waiting for Nethermind initialization!"
    echo -e "${RED}[ERROR]${NC} Log file: ${NETHERMIND_LOG}"
    if [ -f "${NETHERMIND_LOG}" ]; then
        echo -e "${RED}[ERROR]${NC} Last 20 lines of log:"
        tail -20 "${NETHERMIND_LOG}"
    fi
    exit 2
fi

echo -e "${GREEN}[STEP 2/4]${NC} Nethermind RPC is ready!"

# Give Nethermind a moment to stabilize
sleep 2

# Step 3: Run Nitro test
echo -e "${YELLOW}[STEP 3/4]${NC} Running Nitro comparison test..."
cd "${NITRO_REPO}"

# Set required environment variables
export PR_NETH_RPC_CLIENT_URL="http://localhost:20545"
export PR_NETH_WS_CLIENT_URL="ws://localhost:28551"

# Run test with timeout
TEST_OUTPUT=$(mktemp)
TEST_EXIT_CODE=0

echo -e "${BLUE}[TEST]${NC} Starting TestExecutionClientOnlyComparison..."
echo -e "${BLUE}[TEST]${NC} Output filtering: $([ "$VERBOSE" = true ] && echo "disabled (verbose mode)" || echo "enabled")"
echo ""

# Run test with filtered output
(
    timeout ${TEST_TIMEOUT}s go test -run TestExecutionClientOnlyComparison ./system_tests/ -v 2>&1 | create_log_filter
    exit ${PIPESTATUS[0]}
) &
TEST_PID=$!

# Wait for test to complete
wait $TEST_PID 2>/dev/null || TEST_EXIT_CODE=$?

echo ""

# Step 4: Report results
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║              TEST PASSED! ✓                        ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════╝${NC}"
    exit 0
elif [ $TEST_EXIT_CODE -eq 124 ]; then
    echo -e "${RED}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║              TEST TIMEOUT! ✗                       ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════╝${NC}"
    echo -e "${RED}[ERROR]${NC} Test exceeded ${TEST_TIMEOUT}s timeout"
    exit 1
else
    echo -e "${RED}╔════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║              TEST FAILED! ✗                        ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════╝${NC}"
    echo -e "${RED}[ERROR]${NC} Test exited with code: $TEST_EXIT_CODE"

    # Show last lines of Nethermind log in case there were errors
    if [ -f "${NETHERMIND_LOG}" ]; then
        echo ""
        echo -e "${YELLOW}[DEBUG]${NC} Last 20 lines of Nethermind log:"
        tail -20 "${NETHERMIND_LOG}" | create_log_filter
    fi

    exit 1
fi
