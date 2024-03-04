#!/usr/bin/env bash
# The script starts up the test node (with timeout 1 minute), to make sure the
# nitro-testnode script isn't out of sync with flags with nitro.
# This is used in CI, basically as smoke test.

timeout 60 ./nitro-testnode/test-node.bash --init --dev || exit_status=$?

if  [ -n "$exit_status" ] && [ $exit_status -ne 0 ] && [ $exit_status -ne 124 ]; then
    echo "Startup failed."
    exit $exit_status
fi

echo "Startup succeeded."
