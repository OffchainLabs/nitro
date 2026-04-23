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
		echo "ok  $(BLOCK_INPUTS_DIR)"; \
	else \
		echo "FAIL — see $(BLOCK_INPUTS_DIR)/record.log" >&2; exit 1; \
	fi
