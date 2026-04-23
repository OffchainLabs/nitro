TOP              := $(abspath $(CURDIR)/../..)
BLOCK_INPUTS_DIR := $(TOP)/target/sp1/block-inputs

# Run all block-recording system tests and write block input JSONs to $(BLOCK_INPUTS_DIR).
# Each test produces $(BLOCK_INPUTS_DIR)/<TestName>/block_inputs_<id>.json.
.PHONY: record-blocks
record-blocks:
	@rm -rf $(BLOCK_INPUTS_DIR)
	@mkdir -p $(BLOCK_INPUTS_DIR)
	@if cd $(TOP) && go test \
		-tags block_recording \
		-run 'TestRecordBlock' \
		-timeout 5m \
		./system_tests/ -- \
		-recordBlockInputs.enable=true \
		-recordBlockInputs.WithBaseDir=$(BLOCK_INPUTS_DIR) \
		-recordBlockInputs.WithTimestampDirEnabled=false \
		>$(BLOCK_INPUTS_DIR)/record.log 2>&1; then \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockTransfer/*.json   $(BLOCK_INPUTS_DIR)/transfer.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockSolidity/*.json   $(BLOCK_INPUTS_DIR)/solidity.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockStylus/*.json     $(BLOCK_INPUTS_DIR)/stylus.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockStylusHeavy/*.json $(BLOCK_INPUTS_DIR)/stylus_heavy.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockMixed/*.json      $(BLOCK_INPUTS_DIR)/mixed.json; \
		rm -rf $(BLOCK_INPUTS_DIR)/TestRecordBlock*/; \
		echo "ok  $(BLOCK_INPUTS_DIR)"; \
	else \
		echo "FAIL — see $(BLOCK_INPUTS_DIR)/record.log" >&2; exit 1; \
	fi
