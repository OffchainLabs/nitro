#!/usr/bin/env bash
set -euo pipefail

TESTS=(
  TestRecordBlockSingleTransfer
  TestRecordBlockMultipleTransfers
  TestRecordBlockManyTransfers
  TestRecordBlockSolidityDeploy
  TestRecordBlockERC20Deploy
  TestRecordBlockERC20Transfers
  TestRecordBlockWasmDeploy
  TestRecordBlockWasmStorageWrite
  TestRecordBlockWasmMultipleStorageWrites
  TestRecordBlockWasmMulticallStorageOps
  TestRecordBlockWasmKeccak
  TestRecordBlockWasmMath
  TestRecordBlockWasmCreate
  TestRecordBlockWasmLogs
  TestRecordBlockSolidityRepeatedIncrements
  TestRecordBlockMixedEthAndSolidity
  TestRecordBlockMixedSolidityAndWasm
  TestRecordBlockMultipleSolidityDeploys
  TestRecordBlockWasmDeepMulticall
  TestRecordBlockWasmLargeMulticall
  TestRecordBlockLargeContractDeploy
  TestRecordBlockTransfersWithCalldata
  TestRecordBlockMultipleWasmDeploys
  TestRecordBlockPrecompileCalls
  TestRecordBlockERC20FullWorkflow
  TestRecordBlockWasmMulticallStoreAndLog
  TestRecordBlockWasmMultipleCreates
  TestRecordBlockMixedAll
)

PASSED=0
FAILED=0
FAILURES=()

for test in "${TESTS[@]}"; do
  echo "========================================"
  echo "Running: $test"
  echo "========================================"
  if go test -v -run "${test}\$" ./system_tests/... -count 1 -- \
    --recordBlockInputs.enable=true \
    --recordBlockInputs.WithBaseDir=target/ \
    --recordBlockInputs.WithTimestampDirEnabled=false \
    --recordBlockInputs.WithBlockIdInFileNameEnabled=false; then
    echo "PASS: $test"
    PASSED=$((PASSED + 1))
  else
    echo "FAIL: $test"
    FAILED=$((FAILED + 1))
    FAILURES+=("$test")
  fi
  echo ""
done

echo "========================================"
echo "Results: $PASSED passed, $FAILED failed out of ${#TESTS[@]} tests"
if [ ${#FAILURES[@]} -gt 0 ]; then
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do
    echo "  - $f"
  done
  exit 1
fi
