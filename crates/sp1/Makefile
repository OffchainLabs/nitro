TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1
BLOCK_INPUTS_DIR := $(TOP)/target/sp1/block-inputs

# Build brotli for RISC-V (SP1); artifacts go to $(BROTLI_LIB_DIR).
.PHONY: brotli
brotli:
	@cp $(TOP)/crates/sp1/brotli_cmake_patch.txt $(TOP)/brotli/CMakeLists.txt
	@rm -rf $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR)
	@mkdir -p $(BROTLI_BUILD_DIR)
	@if { cmake -S $(TOP)/brotli -B $(BROTLI_BUILD_DIR) \
		-DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
		-DCMAKE_TRY_COMPILE_TARGET_TYPE=STATIC_LIBRARY \
		-DCMAKE_SYSTEM_NAME=Generic \
		-DCMAKE_C_COMPILER=$(RISCV)/bin/riscv64-unknown-elf-gcc \
		"-DCMAKE_C_FLAGS=-march=rv64im -mabi=lp64 -DBROTLI_BUILD_PORTABLE -mcmodel=medany -ffunction-sections -fdata-sections -fPIC" \
		-DCMAKE_AR=$(RISCV)/bin/riscv64-unknown-elf-ar \
		-DCMAKE_RANLIB=$(RISCV)/bin/riscv64-unknown-elf-ranlib \
		-DCMAKE_BUILD_TYPE=Release \
		-DCMAKE_INSTALL_PREFIX=$(BROTLI_LIB_DIR) \
		-DBROTLI_DISABLE_TESTS=ON \
		&& $(MAKE) -C $(BROTLI_BUILD_DIR) \
		&& $(MAKE) -C $(BROTLI_BUILD_DIR) install; } \
		>$(BROTLI_BUILD_DIR)/build.log 2>&1; then \
		echo "ok  $(BROTLI_LIB_DIR)"; \
	else \
		echo "FAIL — see $(BROTLI_BUILD_DIR)/build.log" >&2; exit 1; \
	fi

# Run all block-recording system tests; artifacts go to $(BLOCK_INPUTS_DIR).
.PHONY: record-blocks
record-blocks:
	@rm -rf $(BLOCK_INPUTS_DIR)
	@mkdir -p $(BLOCK_INPUTS_DIR)
	@if cd $(TOP) && go test \
		-tags block_recording \
		-run 'TestRecordBlock' \
		-timeout 5m \
		-parallel 1 \
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
