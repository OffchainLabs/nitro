TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
# Keep in sync with the sp1-* crate versions in Cargo.lock.
SP1_VERSION      := v6.2.1
BROTLI_SRC_DIR   := $(TOP)/target/build-sp1/brotli-src
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1

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
nitro-deps: # Build necessary nitro dependencies, including replay.wasm
nitro-deps: brotli
	@cd $(TOP) && $(MAKE) test-go-deps

.PHONY: build
build: # Mocked system build (temporary solution until we have all the necessary crates available)
build: nitro-deps
	@cargo build --locked -p stylus-compiler-runner --release

.PHONY: clippy
clippy: # Run clippy on the sp1 workspace crates.
	@cargo clippy --locked -p stylus-compiler-program -p stylus-compiler-runner -- -D warnings -A static_mut_refs -A clippy::too_many_arguments

.PHONY: compare
compare: # Compare compilation consistency between native and SP1 execution mode.
	@cd $(TOP) && cargo run --locked --manifest-path $(CURDIR)/Cargo.toml -p stylus-compiler-runner --release -- compare

.PHONY: prove
prove: # Generate and verify a proof of correct Stylus compilation.
	@cd $(TOP) && cargo run --locked --manifest-path $(CURDIR)/Cargo.toml -p stylus-compiler-runner --release -- prove

.PHONY: clean
clean: # Remove all generated artifacts.
	@rm -rf $(BROTLI_SRC_DIR) $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR) target
