TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1
OUTPUT_DIR       := $(TOP)/target/sp1
BLOCK_INPUTS_DIR := $(OUTPUT_DIR)/block-inputs

# Bump SP1's maximum heap memory size.
export SP1_ZKVM_MAX_MEMORY := 1099511627776

# On macOS, point secp256k1-sys (and any other cc-rs crate) at the SP1-provided
# RISC-V toolchain. Without these, the host ar/ranlib produce an empty
# libsecp256k1.a (ELF objects can't be indexed by macOS ranlib) and linking
# the SP1 program fails with undefined rustsecp256k1_v0_10_0_* symbols.
ifeq ($(shell uname -s),Darwin)
export RISCV_GNU_TOOLCHAIN               := $(HOME)/.sp1/riscv
export AR_riscv64im_unknown_none_elf     := $(HOME)/.sp1/riscv/bin/riscv64-unknown-elf-ar
export RANLIB_riscv64im_unknown_none_elf := $(HOME)/.sp1/riscv/bin/riscv64-unknown-elf-ranlib
endif

.PHONY: help
help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: install-sp1
install-sp1: # Install the SP1 RISC-V toolchain.
	@curl -L https://sp1up.succinct.xyz | bash
	@$(HOME)/.sp1/bin/sp1up -c
	@$(RISCV)/bin/riscv64-unknown-elf-gcc --version

.PHONY: brotli
brotli: # Build brotli for RISC-V (SP1); artifacts go to $(BROTLI_LIB_DIR).
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
		echo "Brotli successfully built"; \
	else \
		echo "FAIL — see $(BROTLI_BUILD_DIR)/build.log" >&2; exit 1; \
	fi

.PHONY: nitro-deps
nitro-deps: # Build necessary nitro dependencies, including replay.wasm
nitro-deps: brotli
	@cd $(TOP) && $(MAKE) build-replay-env test-go-deps

.PHONY: record-blocks
record-blocks: # Run all block-recording system tests; artifacts go to $(BLOCK_INPUTS_DIR).
record-blocks: nitro-deps
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
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockTransfer/*.json    $(BLOCK_INPUTS_DIR)/transfer.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockSolidity/*.json    $(BLOCK_INPUTS_DIR)/solidity.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockStylus/*.json      $(BLOCK_INPUTS_DIR)/stylus.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockStylusHeavy/*.json $(BLOCK_INPUTS_DIR)/stylus_heavy.json; \
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockMixed/*.json       $(BLOCK_INPUTS_DIR)/mixed.json; \
		rm -rf $(BLOCK_INPUTS_DIR)/TestRecordBlock*/; \
		echo "Blocks successfully recorded"; \
	else \
		echo "FAIL — see $(BLOCK_INPUTS_DIR)/record.log" >&2; exit 1; \
	fi

.PHONY: build
build: # Build the SP1 runner; artifacts go to $(OUTPUT_DIR).
build: nitro-deps
	@mkdir -p $(OUTPUT_DIR)
	@# Rebuild replay.wasm without running `./scripts/remove_reference_types.sh`.
	@cd $(TOP) && GOOS=wasip1 GOARCH=wasm go build -o $(OUTPUT_DIR)/replay.wasm $(TOP)/cmd/replay/...
	@cargo run --release -p sp1-builder -- \
		--replay-wasm $(OUTPUT_DIR)/replay.wasm \
		--output-folder $(OUTPUT_DIR)
	@cargo build --release -p sp1-runner
	@cp $(TOP)/target/elf-compilation/riscv64im-succinct-zkvm-elf/release/stylus-compiler-program $(OUTPUT_DIR)
	@cp $(TOP)/target/release/sp1-runner $(OUTPUT_DIR)

.PHONY: clean
clean: # Remove all generated artifacts.
	@rm -rf $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR) $(OUTPUT_DIR)
