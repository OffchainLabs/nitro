#!/bin/bash

# Define dummy data
DUMMY_DATA="some-dummy-data"

# Create the user_report_data structure with dummy data
REPORT_DATA=$(printf "%-64s" "$DUMMY_DATA")

# Write to /dev/attestation/user_report_data
echo -n "$REPORT_DATA" > /dev/attestation/user_report_data
if [ $? -ne 0 ]; then
  echo "Failed to write to /dev/attestation/user_report_data"
  exit 1
fi
echo "Successfully wrote user report data."

# Attempt to read and print the report as a hex dump
if ! dd if=/dev/attestation/report bs=1 | xxd -p; then
  echo "Failed to read from /dev/attestation/report"
  exit 1
fi

echo "Successfully read attestation report."

# Adjust ownership for the .arbitrum folder to the 'user' inside the container
if [ -d "/home/user/.arbitrum" ]; then
   sudo chown -R user:user /home/user/.arbitrum
fi

# Start Nitro process
exec /usr/local/bin/nitro \
    --validation.wasm.enable-wasmroots-check=false \
    --conf.file /config/poster_config.json