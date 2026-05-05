TOP              := $(abspath $(CURDIR)/../..)
RISCV            := $(HOME)/.sp1/riscv
BROTLI_BUILD_DIR := $(TOP)/target/build-sp1/brotli
BROTLI_LIB_DIR   := $(TOP)/target/lib-sp1
OUTPUT_DIR       := $(TOP)/target/sp1

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
	@cargo build -p stylus-compiler-runner --release

.PHONY: clean
clean: # Remove all generated artifacts.
	@rm -rf $(BROTLI_BUILD_DIR) $(BROTLI_LIB_DIR) $(OUTPUT_DIR)
