#!/bin/bash

# Read the binary file and convert to hex using xxd
xxd -p report.bin | tr -d '\n' > report.hex

# Extract the desired byte ranges (64:96 and 128:160)
mr_enclave=$(cut -c 129-192 report.hex)  # Extract bytes 64:96 (1-based index)
mr_signer=$(cut -c 257-320 report.hex)  # Extract bytes 128:160 (1-based index)

# Print the hex values
echo "MRENCLAVE: $mr_enclave"
echo "MRSIGNER: $mr_signer"