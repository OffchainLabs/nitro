# Copyright 2026, Offchain Labs, Inc.
# For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
# Keep in sync with the sp1-* crate versions in Cargo.lock.
SP1_VERSION      := v6.2.1
BROTLI_SRC_DIR   := $(TOP)/target/build-sp1/brotli-src
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1
OUTPUT_DIR       := $(TOP)/target/sp1
BLOCK_INPUTS_DIR := $(OUTPUT_DIR)/block-inputs

# Bump SP1's maximum heap memory size.
export SP1_ZKVM_MAX_MEMORY := 1099511627776

.PHONY: help
help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: install-sp1
install-sp1: # Install the SP1 RISC-V toolchain.
	@curl -L https://sp1up.succinct.xyz | bash
	@$(HOME)/.sp1/bin/sp1up --version $(SP1_VERSION) -c
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
nitro-deps: # Build necessary nitro dependencies, including contract bindings and replay.wasm
nitro-deps: brotli
	@cd $(TOP) && $(MAKE) contracts
	@cd $(TOP) && $(MAKE) test-go-deps

.PHONY: record-blocks
record-blocks: # Run all block-recording system tests; artifacts go to $(BLOCK_INPUTS_DIR).
record-blocks: nitro-deps
	@rm -rf $(BLOCK_INPUTS_DIR)
	@mkdir -p $(BLOCK_INPUTS_DIR)
	@set -e; \
	cd $(TOP); \
	if ! go test \
		-tags block_recording \
		-run 'TestRecordBlock' \
		-timeout 5m \
		-parallel 1 \
		./system_tests/ -- \
		-recordBlockInputs.enable=true \
		-recordBlockInputs.WithBaseDir=$(BLOCK_INPUTS_DIR) \
		-recordBlockInputs.WithTimestampDirEnabled=false \
		>$(BLOCK_INPUTS_DIR)/record.log 2>&1; then \
		echo "FAIL — record.log:" >&2; cat $(BLOCK_INPUTS_DIR)/record.log >&2; exit 1; \
	fi; \
	found=0; \
	for dir in $(BLOCK_INPUTS_DIR)/TestRecordBlock*/; do \
		[ -d "$$dir" ] || continue; \
		found=$$((found+1)); \
		set -- "$$dir"*.json; \
		if [ $$# -ne 1 ] || [ ! -f "$$1" ]; then \
			echo "expected exactly one recording in $$dir, got: $$*" >&2; exit 1; \
		fi; \
		name=$$(basename "$$dir" | sed -e 's/^TestRecordBlock//' -e 's/\([a-z0-9]\)\([A-Z]\)/\1_\2/g' | tr '[:upper:]' '[:lower:]'); \
		if [ -e "$(BLOCK_INPUTS_DIR)/$$name.json" ]; then \
			echo "duplicate block name $$name.json (test names collide after snake_casing?)" >&2; exit 1; \
		fi; \
		mv "$$1" "$(BLOCK_INPUTS_DIR)/$$name.json"; \
		rmdir "$$dir"; \
	done; \
	if [ "$$found" -eq 0 ]; then \
		echo "no TestRecordBlock recordings produced — did all tests skip?" >&2; exit 1; \
	fi; \
	echo "Recorded $$found blocks"

.PHONY: build
build: # Build the SP1 workspace; replay artifacts go to $(OUTPUT_DIR).
build: nitro-deps
	@mkdir -p $(OUTPUT_DIR)
	@# Rebuild replay.wasm without running `./scripts/remove_reference_types.sh`.
	@cd $(TOP) && GOOS=wasip1 GOARCH=wasm go build -o $(OUTPUT_DIR)/replay.wasm $(TOP)/cmd/replay/...
	@cargo build --locked --release
	@cargo run --locked --release -p replay-builder -- \
		--replay-wasm $(OUTPUT_DIR)/replay.wasm \
		--output-folder $(OUTPUT_DIR)
	@cp $(CURDIR)/target/release/replay-runner $(OUTPUT_DIR)

.PHONY: run
run: # Validate a block with the SP1 replay runner.
	$(OUTPUT_DIR)/replay-runner --program $(OUTPUT_DIR)/replay-program.elf

.PHONY: test
test: # Run the sp1 workspace unit tests.
test: build
	@cargo test --locked --release

.PHONY: clippy
clippy: # Run clippy on the sp1 workspace crates.
clippy: build
	@cargo clippy --locked --workspace -- -D warnings

.PHONY: stylus-compiler-compare
stylus-compiler-compare: # Compare Stylus compilation consistency between native and SP1 execution mode.
	@cd $(TOP) && cargo run --locked --manifest-path $(CURDIR)/Cargo.toml -p stylus-compiler-runner --release -- compare

.PHONY: stylus-compiler-prove
stylus-compiler-prove: # Generate and verify a proof of correct Stylus compilation.
	@cd $(TOP) && cargo run --locked --manifest-path $(CURDIR)/Cargo.toml -p stylus-compiler-runner --release -- prove

.PHONY: clean
clean: # Remove all generated artifacts.
	@rm -rf $(BROTLI_SRC_DIR) $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR) target
