TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1
OUTPUT_DIR       := $(TOP)/target/sp1
BLOCK_INPUTS_DIR := $(OUTPUT_DIR)/block-inputs

# Name of the block input (without .json) for the `run` target; override with BLOCK=<name>.
BLOCK ?= stylus

# Bump SP1's maximum heap memory size.
export SP1_ZKVM_MAX_MEMORY := 1099511627776

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
	@echo "Building brotli for RISC-V (SP1)..."
	@rm -rf $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR)
	@mkdir -p $(BROTLI_BUILD_DIR)
	@cp $(TOP)/brotli/CMakeLists.txt $(TOP)/brotli/CMakeLists.txt.bak; \
	trap 'mv $(TOP)/brotli/CMakeLists.txt.bak $(TOP)/brotli/CMakeLists.txt' EXIT INT TERM; \
	cp $(TOP)/crates/sp1/brotli_cmake_patch.txt $(TOP)/brotli/CMakeLists.txt; \
	if { cmake -S $(TOP)/brotli -B $(BROTLI_BUILD_DIR) \
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
	@cd $(TOP) && $(MAKE) test-go-deps

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
		mv $(BLOCK_INPUTS_DIR)/TestRecordBlockSignatures/*.json  $(BLOCK_INPUTS_DIR)/signatures.json; \
		rm -rf $(BLOCK_INPUTS_DIR)/TestRecordBlock*/; \
		echo "Blocks successfully recorded"; \
	else \
		echo "FAIL — record.log:" >&2; cat $(BLOCK_INPUTS_DIR)/record.log >&2; exit 1; \
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

.PHONY: run
run: # Run sp1-runner on $(BLOCK_INPUTS_DIR)/$(BLOCK).json (override with BLOCK=<name>).
	$(OUTPUT_DIR)/sp1-runner \
		--program $(OUTPUT_DIR)/dumped_replay_wasm.elf \
		--stylus-compiler-program $(OUTPUT_DIR)/stylus-compiler-program \
		--block-file $(BLOCK_INPUTS_DIR)/$(BLOCK).json

PROFILE_SCRIPT       ?= $(CURDIR)/profile.py
PROFILE_DELTA_SCRIPT ?= $(CURDIR)/profile_delta.py

.PHONY: profile
profile: # Profile bootloading, stylus compilation, and reexecution across all block types.
	@cargo build --release -p sp1-runner --features profiling
	@cp $(TOP)/target/release/sp1-runner $(OUTPUT_DIR)/sp1-runner-profiling
	@python3 $(PROFILE_SCRIPT) \
		--output-dir $(OUTPUT_DIR) \
		--block-inputs-dir $(BLOCK_INPUTS_DIR) \
		$(if $(JSON_OUTPUT),--json-output $(JSON_OUTPUT),)

.PHONY: profile-snapshot
profile-snapshot: # Run `profile` and save the JSON snapshot to $(OUTPUT_DIR)/profile-$(LABEL).json.
	@if [ -z "$(LABEL)" ]; then echo "ERROR: LABEL is required (e.g. make profile-snapshot LABEL=baseline)" >&2; exit 1; fi
	@$(MAKE) -f $(firstword $(MAKEFILE_LIST)) profile JSON_OUTPUT=$(OUTPUT_DIR)/profile-$(LABEL).json
	@echo "Snapshot written to $(OUTPUT_DIR)/profile-$(LABEL).json"

.PHONY: profile-compare
profile-compare: # Compare two snapshots: BASE=<old-label> NEW=<new-label>; writes markdown to $(OUTPUT_DIR).
	@if [ -z "$(BASE)" ] || [ -z "$(NEW)" ]; then \
		echo "ERROR: BASE and NEW are required (e.g. make profile-compare BASE=baseline NEW=k256)" >&2; exit 1; \
	fi
	@python3 $(PROFILE_DELTA_SCRIPT) \
		--old $(OUTPUT_DIR)/profile-$(BASE).json \
		--new $(OUTPUT_DIR)/profile-$(NEW).json \
		--output $(OUTPUT_DIR)/profile-delta-$(BASE)-vs-$(NEW).md
	@cat $(OUTPUT_DIR)/profile-delta-$(BASE)-vs-$(NEW).md

.PHONY: clean
clean: # Remove all generated artifacts.
	@rm -rf $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR) $(OUTPUT_DIR)
