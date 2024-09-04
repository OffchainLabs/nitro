# Copyright 2021-2024, Offchain Labs, Inc.
# For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

# Docker builds mess up file timestamps. Then again, in docker builds we never
# have to update an existing file. So - for docker, convert all dependencies
# to order-only dependencies (timestamps ignored).
# WARNING: when using this trick, you cannot use the $< automatic variable

ifeq ($(origin NITRO_BUILD_IGNORE_TIMESTAMPS),undefined)
 DEP_PREDICATE:=
 ORDER_ONLY_PREDICATE:=|
else
 DEP_PREDICATE:=|
 ORDER_ONLY_PREDICATE:=
endif


ifneq ($(origin NITRO_VERSION),undefined)
 GOLANG_LDFLAGS += -X github.com/offchainlabs/nitro/cmd/util/confighelpers.version=$(NITRO_VERSION)
endif

ifneq ($(origin NITRO_DATETIME),undefined)
 GOLANG_LDFLAGS += -X github.com/offchainlabs/nitro/cmd/util/confighelpers.datetime=$(NITRO_DATETIME)
endif

ifneq ($(origin NITRO_MODIFIED),undefined)
 GOLANG_LDFLAGS += -X github.com/offchainlabs/nitro/cmd/util/confighelpers.modified=$(NITRO_MODIFIED)
endif

ifneq ($(origin GOLANG_LDFLAGS),undefined)
 GOLANG_PARAMS = -ldflags="-extldflags '-ldl' $(GOLANG_LDFLAGS)"
endif

UNAME_S := $(shell uname -s)

# In Mac OSX, there are a lot of warnings emitted if these environment variables aren't set.
ifeq ($(UNAME_S), Darwin)
  export MACOSX_DEPLOYMENT_TARGET := $(shell sw_vers -productVersion)
  export CGO_LDFLAGS := -Wl,-no_warn_duplicate_libraries
endif

precompile_names = AddressTable Aggregator BLS Debug FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubst %,./solgen/generated/%.go, $(precompile_names))

output_root=target
output_latest=$(output_root)/machines/latest

repo_dirs = arbos arbcompress arbnode arbutil arbstate cmd das precompiles solgen system_tests util validator wavmio
go_source.go = $(wildcard $(patsubst %,%/*.go, $(repo_dirs)) $(patsubst %,%/*/*.go, $(repo_dirs)))
go_source.s  = $(wildcard $(patsubst %,%/*.s, $(repo_dirs)) $(patsubst %,%/*/*.s, $(repo_dirs)))
go_source = $(go_source.go) $(go_source.s)

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)

replay_wasm=$(output_latest)/replay.wasm

arb_brotli_files = $(wildcard arbitrator/brotli/src/*.* arbitrator/brotli/src/*/*.* arbitrator/brotli/*.toml arbitrator/brotli/*.rs) .make/cbrotli-lib .make/cbrotli-wasm

arbitrator_generated_header=$(output_root)/include/arbitrator.h
arbitrator_wasm_libs=$(patsubst %, $(output_root)/machines/latest/%.wasm, forward wasi_stub host_io soft-float arbcompress user_host program_exec)
arbitrator_stylus_lib=$(output_root)/lib/libstylus.a
prover_bin=$(output_root)/bin/prover
arbitrator_jit=$(output_root)/bin/jit

arbitrator_cases=arbitrator/prover/test-cases

arbitrator_tests_wat=$(wildcard $(arbitrator_cases)/*.wat)
arbitrator_tests_rust=$(wildcard $(arbitrator_cases)/rust/src/bin/*.rs)

arbitrator_test_wasms=$(patsubst %.wat,%.wasm, $(arbitrator_tests_wat)) $(patsubst $(arbitrator_cases)/rust/src/bin/%.rs,$(arbitrator_cases)/rust/target/wasm32-wasi/release/%.wasm, $(arbitrator_tests_rust)) $(arbitrator_cases)/go/testcase.wasm

arbitrator_tests_link_info = $(shell cat $(arbitrator_cases)/link.txt | xargs)
arbitrator_tests_link_deps = $(patsubst %,$(arbitrator_cases)/%.wasm, $(arbitrator_tests_link_info))

arbitrator_tests_forward_wats = $(wildcard $(arbitrator_cases)/forward/*.wat)
arbitrator_tests_forward_deps = $(arbitrator_tests_forward_wats:wat=wasm)

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot

arbitrator_wasm_lib_flags=$(patsubst %, -l %, $(arbitrator_wasm_libs))

rust_arbutil_files = $(wildcard arbitrator/arbutil/src/*.* arbitrator/arbutil/src/*/*.* arbitrator/arbutil/*.toml arbitrator/caller-env/src/*.* arbitrator/caller-env/src/*/*.* arbitrator/caller-env/*.toml) .make/cbrotli-lib

prover_direct_includes = $(patsubst %,$(output_latest)/%.wasm, forward forward_stub)
prover_dir = arbitrator/prover/
rust_prover_files = $(wildcard $(prover_dir)/src/*.* $(prover_dir)/src/*/*.* $(prover_dir)/*.toml $(prover_dir)/*.rs) $(rust_arbutil_files) $(prover_direct_includes) $(arb_brotli_files)

wasm_lib = arbitrator/wasm-libraries
wasm_lib_cargo = $(wasm_lib)/.cargo/config.toml
wasm_lib_deps = $(wildcard $(wasm_lib)/$(1)/*.toml $(wasm_lib)/$(1)/src/*.rs $(wasm_lib)/$(1)/*.rs) $(wasm_lib_cargo) $(rust_arbutil_files) $(arb_brotli_files) .make/machines
wasm_lib_go_abi = $(call wasm_lib_deps,go-abi)
wasm_lib_forward = $(call wasm_lib_deps,forward)
wasm_lib_user_host_trait = $(call wasm_lib_deps,user-host-trait)
wasm_lib_user_host = $(call wasm_lib_deps,user-host) $(wasm_lib_user_host_trait)

forward_dir = $(wasm_lib)/forward

stylus_files = $(wildcard $(stylus_dir)/*.toml $(stylus_dir)/src/*.rs) $(wasm_lib_user_host_trait) $(rust_prover_files)

jit_dir = arbitrator/jit
jit_files = $(wildcard $(jit_dir)/*.toml $(jit_dir)/*.rs $(jit_dir)/src/*.rs $(jit_dir)/src/*/*.rs) $(stylus_files)

wasm32_wasi = target/wasm32-wasi/release
wasm32_unknown = target/wasm32-unknown-unknown/release

stylus_dir = arbitrator/stylus
stylus_test_dir = arbitrator/stylus/tests
stylus_cargo = arbitrator/stylus/tests/.cargo/config.toml

rust_sdk = arbitrator/langs/rust
c_sdk = arbitrator/langs/c
stylus_lang_rust = $(wildcard $(rust_sdk)/*/src/*.rs $(rust_sdk)/*/src/*/*.rs $(rust_sdk)/*/*.toml)
stylus_lang_c    = $(wildcard $(c_sdk)/*/*.c $(c_sdk)/*/*.h)
stylus_lang_bf   = $(wildcard arbitrator/langs/bf/src/*.* arbitrator/langs/bf/src/*.toml)

STYLUS_NIGHTLY_VER ?= "+nightly"

cargo_nightly = cargo $(STYLUS_NIGHTLY_VER) build -Z build-std=std,panic_abort -Z build-std-features=panic_immediate_abort

get_stylus_test_wasm = $(stylus_test_dir)/$(1)/$(wasm32_unknown)/$(1).wasm
get_stylus_test_rust = $(wildcard $(stylus_test_dir)/$(1)/*.toml $(stylus_test_dir)/$(1)/src/*.rs) $(stylus_cargo) $(stylus_lang_rust)
get_stylus_test_c    = $(wildcard $(c_sdk)/examples/$(1)/*.c $(c_sdk)/examples/$(1)/*.h) $(stylus_lang_c)
stylus_test_bfs      = $(wildcard $(stylus_test_dir)/bf/*.b)

stylus_test_keccak_wasm           = $(call get_stylus_test_wasm,keccak)
stylus_test_keccak_src            = $(call get_stylus_test_rust,keccak)
stylus_test_keccak-100_wasm       = $(call get_stylus_test_wasm,keccak-100)
stylus_test_keccak-100_src        = $(call get_stylus_test_rust,keccak-100)
stylus_test_fallible_wasm         = $(call get_stylus_test_wasm,fallible)
stylus_test_fallible_src          = $(call get_stylus_test_rust,fallible)
stylus_test_storage_wasm          = $(call get_stylus_test_wasm,storage)
stylus_test_storage_src           = $(call get_stylus_test_rust,storage)
stylus_test_multicall_wasm        = $(call get_stylus_test_wasm,multicall)
stylus_test_multicall_src         = $(call get_stylus_test_rust,multicall)
stylus_test_log_wasm              = $(call get_stylus_test_wasm,log)
stylus_test_log_src               = $(call get_stylus_test_rust,log)
stylus_test_create_wasm           = $(call get_stylus_test_wasm,create)
stylus_test_create_src            = $(call get_stylus_test_rust,create)
stylus_test_math_wasm             = $(call get_stylus_test_wasm,math)
stylus_test_math_src              = $(call get_stylus_test_rust,math)
stylus_test_evm-data_wasm         = $(call get_stylus_test_wasm,evm-data)
stylus_test_evm-data_src          = $(call get_stylus_test_rust,evm-data)
stylus_test_sdk-storage_wasm      = $(call get_stylus_test_wasm,sdk-storage)
stylus_test_sdk-storage_src       = $(call get_stylus_test_rust,sdk-storage)
stylus_test_erc20_wasm            = $(call get_stylus_test_wasm,erc20)
stylus_test_erc20_src             = $(call get_stylus_test_rust,erc20)
stylus_test_read-return-data_wasm = $(call get_stylus_test_wasm,read-return-data)
stylus_test_read-return-data_src  = $(call get_stylus_test_rust,read-return-data)

stylus_test_wasms = $(stylus_test_keccak_wasm) $(stylus_test_keccak-100_wasm) $(stylus_test_fallible_wasm) $(stylus_test_storage_wasm) $(stylus_test_multicall_wasm) $(stylus_test_log_wasm) $(stylus_test_create_wasm) $(stylus_test_math_wasm) $(stylus_test_sdk-storage_wasm) $(stylus_test_erc20_wasm) $(stylus_test_read-return-data_wasm) $(stylus_test_evm-data_wasm) $(stylus_test_bfs:.b=.wasm)
stylus_benchmarks = $(wildcard $(stylus_dir)/*.toml $(stylus_dir)/src/*.rs) $(stylus_test_wasms)

# user targets

.PHONY: push
push: lint test-go .make/fmt
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

.PHONY: all
all: build build-replay-env test-gen-proofs
	@touch .make/all

.PHONY: build
build: $(patsubst %,$(output_root)/bin/%, nitro deploy relay daserver datool seq-coordinator-invalidate nitro-val seq-coordinator-manager dbconv)
	@printf $(done)

.PHONY: build-node-deps
build-node-deps: $(go_source) build-prover-header build-prover-lib build-jit .make/solgen .make/cbrotli-lib

.PHONY: test-go-deps
test-go-deps: \
	build-replay-env \
	$(stylus_test_wasms) \
	$(arbitrator_stylus_lib) \
	$(arbitrator_generated_header) \
	$(patsubst %,$(arbitrator_cases)/%.wasm, global-state read-inboxmsg-10 global-state-wrapper const)

.PHONY: build-prover-header
build-prover-header: $(arbitrator_generated_header)

.PHONY: build-prover-lib
build-prover-lib: $(arbitrator_stylus_lib)

.PHONY: build-prover-bin
build-prover-bin: $(prover_bin)

.PHONY: build-jit
build-jit: $(arbitrator_jit)

.PHONY: build-replay-env
build-replay-env: $(prover_bin) $(arbitrator_jit) $(arbitrator_wasm_libs) $(replay_wasm) $(output_latest)/machine.wavm.br

.PHONY: build-wasm-libs
build-wasm-libs: $(arbitrator_wasm_libs)

.PHONY: build-wasm-bin
build-wasm-bin: $(replay_wasm)

.PHONY: build-solidity
build-solidity: .make/solidity

.PHONY: contracts
contracts: .make/solgen
	@printf $(done)

.PHONY: format fmt
format fmt: .make/fmt
	@printf $(done)

.PHONY: lint
lint: .make/lint
	@printf $(done)

.PHONY: stylus-benchmarks
stylus-benchmarks: $(stylus_benchmarks)
	cargo test --manifest-path $< --release --features benchmark benchmark_ -- --nocapture
	@printf $(done)

.PHONY: test-go
test-go: .make/test-go
	@printf $(done)

.PHONY: test-go-challenge
test-go-challenge: test-go-deps
	gotestsum --format short-verbose --no-color=false -- -timeout 120m ./system_tests/... -run TestChallenge -tags challengetest
	@printf $(done)

.PHONY: test-go-stylus
test-go-stylus: test-go-deps
	gotestsum --format short-verbose --no-color=false -- -timeout 120m ./system_tests/... -run TestProgramArbitrator -tags stylustest
	@printf $(done)

.PHONY: test-go-redis
test-go-redis: test-go-deps
	TEST_REDIS=redis://localhost:6379/0 gotestsum --format short-verbose --no-color=false -- -p 1 -run TestRedis ./system_tests/... ./arbnode/...
	@printf $(done)

.PHONY: test-gen-proofs
test-gen-proofs: \
        $(arbitrator_test_wasms) \
	$(patsubst $(arbitrator_cases)/%.wat,contracts/test/prover/proofs/%.json, $(arbitrator_tests_wat)) \
	$(patsubst $(arbitrator_cases)/rust/src/bin/%.rs,contracts/test/prover/proofs/rust-%.json, $(arbitrator_tests_rust)) \
	contracts/test/prover/proofs/go.json
	@printf $(done)

.PHONY: test-rust
test-rust: .make/test-rust
	@printf $(done)

# Runs the fastest and most reliable and high-value tests.
.PHONY: tests
tests: test-go test-rust
	@printf $(done)

# Runs all tests, including slow and unreliable tests.
#  Currently, NOT including:
#  - test-go-redis (These testts require additional setup and are not as reliable)
.PHONY: tests-all
tests-all: tests test-go-challenge test-go-stylus test-gen-proofs
	@printf $(done)

.PHONY: wasm-ci-build
wasm-ci-build: $(arbitrator_wasm_libs) $(arbitrator_test_wasms) $(stylus_test_wasms) $(output_latest)/user_test.wasm
	@printf $(done)

.PHONY: clean
clean:
	go clean -testcache
	rm -rf $(arbitrator_cases)/rust/target
	rm -f $(arbitrator_cases)/*.wasm $(arbitrator_cases)/go/testcase.wasm
	rm -rf arbitrator/wasm-testsuite/tests
	rm -rf $(output_root)
	rm -f contracts/test/prover/proofs/*.json contracts/test/prover/spec-proofs/*.json
	rm -rf arbitrator/target
	rm -rf arbitrator/wasm-libraries/target
	rm -f arbitrator/wasm-libraries/soft-float/soft-float.wasm
	rm -f arbitrator/wasm-libraries/soft-float/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.a
	rm -f arbitrator/wasm-libraries/forward/*.wat
	rm -rf arbitrator/stylus/tests/*/target/ arbitrator/stylus/tests/*/*.wasm
	@rm -rf contracts/build contracts/cache solgen/go/
	@rm -f .make/*

.PHONY: docker
docker:
	docker build -t nitro-node-slim --target nitro-node-slim .
	docker build -t nitro-node --target nitro-node .
	docker build -t nitro-node-dev --target nitro-node-dev .

# regular build rules

$(output_root)/bin/nitro: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/nitro"

$(output_root)/bin/deploy: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/deploy"

$(output_root)/bin/relay: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/relay"

$(output_root)/bin/daserver: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/daserver"

$(output_root)/bin/datool: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/datool"

$(output_root)/bin/seq-coordinator-invalidate: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/seq-coordinator-invalidate"

$(output_root)/bin/nitro-val: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/nitro-val"

$(output_root)/bin/seq-coordinator-manager: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/seq-coordinator-manager"

$(output_root)/bin/dbconv: $(DEP_PREDICATE) build-node-deps
	go build $(GOLANG_PARAMS) -o $@ "$(CURDIR)/cmd/dbconv"

# recompile wasm, but don't change timestamp unless files differ
$(replay_wasm): $(DEP_PREDICATE) $(go_source) .make/solgen
	mkdir -p `dirname $(replay_wasm)`
	GOOS=wasip1 GOARCH=wasm go build -o $@ ./cmd/replay/...

$(prover_bin): $(DEP_PREDICATE) $(rust_prover_files)
	mkdir -p `dirname $(prover_bin)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --bin prover ${CARGOFLAGS}
	install arbitrator/target/release/prover $@

$(arbitrator_stylus_lib): $(DEP_PREDICATE) $(stylus_files)
	mkdir -p `dirname $(arbitrator_stylus_lib)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --lib -p stylus ${CARGOFLAGS}
	install arbitrator/target/release/libstylus.a $@

$(arbitrator_jit): $(DEP_PREDICATE) $(jit_files)
	mkdir -p `dirname $(arbitrator_jit)`
	cargo build --manifest-path arbitrator/Cargo.toml --release -p jit ${CARGOFLAGS}
	install arbitrator/target/release/jit $@

$(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm: $(arbitrator_cases)/rust/src/bin/%.rs $(arbitrator_cases)/rust/src/lib.rs $(arbitrator_cases)/rust/.cargo/config.toml
	cargo build --manifest-path $(arbitrator_cases)/rust/Cargo.toml --release --target wasm32-wasi --config $(arbitrator_cases)/rust/.cargo/config.toml --bin $(patsubst $(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm,%, $@)

$(arbitrator_cases)/go/testcase.wasm: $(arbitrator_cases)/go/*.go .make/solgen
	cd $(arbitrator_cases)/go && GOOS=wasip1 GOARCH=wasm go build -o testcase.wasm

$(arbitrator_generated_header): $(DEP_PREDICATE) $(stylus_files)
	@echo creating ${PWD}/$(arbitrator_generated_header)
	mkdir -p `dirname $(arbitrator_generated_header)`
	cd arbitrator/stylus && cbindgen --config cbindgen.toml --crate stylus --output ../../$(arbitrator_generated_header)
	@touch -c $@ # cargo might decide to not rebuild the header

$(output_latest)/wasi_stub.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,wasi-stub)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-unknown-unknown --config $(wasm_lib_cargo) --package wasi-stub
	install arbitrator/wasm-libraries/$(wasm32_unknown)/wasi_stub.wasm $@

arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/softfloat.a: $(DEP_PREDICATE) \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/Makefile \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/platform.h \
		arbitrator/wasm-libraries/soft-float/SoftFloat/source/*.c \
		arbitrator/wasm-libraries/soft-float/SoftFloat/source/include/*.h \
		arbitrator/wasm-libraries/soft-float/SoftFloat/source/8086/*.c \
		arbitrator/wasm-libraries/soft-float/SoftFloat/source/8086/*.h
	cd arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang && make $(MAKEFLAGS)

arbitrator/wasm-libraries/soft-float/bindings32.o: $(DEP_PREDICATE) arbitrator/wasm-libraries/soft-float/bindings32.c
	clang arbitrator/wasm-libraries/soft-float/bindings32.c --sysroot $(WASI_SYSROOT) -I arbitrator/wasm-libraries/soft-float/SoftFloat/source/include -target wasm32-wasi -Wconversion -c -o $@

arbitrator/wasm-libraries/soft-float/bindings64.o: $(DEP_PREDICATE) arbitrator/wasm-libraries/soft-float/bindings64.c
	clang arbitrator/wasm-libraries/soft-float/bindings64.c --sysroot $(WASI_SYSROOT) -I arbitrator/wasm-libraries/soft-float/SoftFloat/source/include -target wasm32-wasi -Wconversion -c -o $@

$(output_latest)/soft-float.wasm: $(DEP_PREDICATE) \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/softfloat.a \
		.make/wasm-lib .make/machines
	wasm-ld \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.o \
		--no-entry -o $@ \
		$(patsubst %,--export wavm__f32_%, abs neg ceil floor trunc nearest sqrt add sub mul div min max) \
		$(patsubst %,--export wavm__f32_%, copysign eq ne lt le gt ge) \
		$(patsubst %,--export wavm__f64_%, abs neg ceil floor trunc nearest sqrt add sub mul div min max) \
		$(patsubst %,--export wavm__f64_%, copysign eq ne lt le gt ge) \
		$(patsubst %,--export wavm__i32_trunc_%,     f32_s f32_u f64_s f64_u) \
		$(patsubst %,--export wavm__i32_trunc_sat_%, f32_s f32_u f64_s f64_u) \
		$(patsubst %,--export wavm__i64_trunc_%,     f32_s f32_u f64_s f64_u) \
		$(patsubst %,--export wavm__i64_trunc_sat_%, f32_s f32_u f64_s f64_u) \
		$(patsubst %,--export wavm__f32_convert_%, i32_s i32_u i64_s i64_u) \
		$(patsubst %,--export wavm__f64_convert_%, i32_s i32_u i64_s i64_u) \
		--export wavm__f32_demote_f64 \
		--export wavm__f64_promote_f32

$(output_latest)/host_io.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,host-io) $(wasm_lib_go_abi)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --config $(wasm_lib_cargo) --package host-io
	install arbitrator/wasm-libraries/$(wasm32_wasi)/host_io.wasm $@

$(output_latest)/user_host.wasm: $(DEP_PREDICATE) $(wasm_lib_user_host) $(rust_prover_files) $(output_latest)/forward_stub.wasm .make/machines
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --config $(wasm_lib_cargo) --package user-host
	install arbitrator/wasm-libraries/$(wasm32_wasi)/user_host.wasm $@

$(output_latest)/program_exec.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,program-exec) $(rust_prover_files) .make/machines
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --config $(wasm_lib_cargo) --package program-exec
	install arbitrator/wasm-libraries/$(wasm32_wasi)/program_exec.wasm $@

$(output_latest)/user_test.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,user-test) $(rust_prover_files) .make/machines
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --config $(wasm_lib_cargo) --package user-test
	install arbitrator/wasm-libraries/$(wasm32_wasi)/user_test.wasm $@

$(output_latest)/arbcompress.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,brotli) $(wasm_lib_go_abi)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --config $(wasm_lib_cargo) --package arbcompress
	install arbitrator/wasm-libraries/$(wasm32_wasi)/arbcompress.wasm $@

$(output_latest)/forward.wasm: $(DEP_PREDICATE) $(wasm_lib_forward) .make/machines
	cargo run --manifest-path $(forward_dir)/Cargo.toml -- --path $(forward_dir)/forward.wat
	wat2wasm $(wasm_lib)/forward/forward.wat -o $@

$(output_latest)/forward_stub.wasm: $(DEP_PREDICATE) $(wasm_lib_forward) .make/machines
	cargo run --manifest-path $(forward_dir)/Cargo.toml -- --path $(forward_dir)/forward_stub.wat --stub
	wat2wasm $(wasm_lib)/forward/forward_stub.wat -o $@

$(output_latest)/machine.wavm.br: $(DEP_PREDICATE) $(prover_bin) $(arbitrator_wasm_libs) $(replay_wasm)
	$(prover_bin) $(replay_wasm) --generate-binaries $(output_latest) \
	$(patsubst %,-l $(output_latest)/%.wasm, forward soft-float wasi_stub host_io user_host arbcompress program_exec)

$(arbitrator_cases)/%.wasm: $(arbitrator_cases)/%.wat
	wat2wasm $< -o $@

$(stylus_test_dir)/%.wasm: $(stylus_test_dir)/%.b $(stylus_lang_bf)
	cargo run --manifest-path arbitrator/langs/bf/Cargo.toml $< -o $@

$(stylus_test_keccak_wasm): $(stylus_test_keccak_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_keccak-100_wasm): $(stylus_test_keccak-100_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_fallible_wasm): $(stylus_test_fallible_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_storage_wasm): $(stylus_test_storage_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_multicall_wasm): $(stylus_test_multicall_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_log_wasm): $(stylus_test_log_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_create_wasm): $(stylus_test_create_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_math_wasm): $(stylus_test_math_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_evm-data_wasm): $(stylus_test_evm-data_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_read-return-data_wasm): $(stylus_test_read-return-data_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_sdk-storage_wasm): $(stylus_test_sdk-storage_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

$(stylus_test_erc20_wasm): $(stylus_test_erc20_src)
	$(cargo_nightly) --manifest-path $< --release --config $(stylus_cargo)
	@touch -c $@ # cargo might decide to not rebuild the binary

contracts/test/prover/proofs/float%.json: $(arbitrator_cases)/float%.wasm $(prover_bin) $(output_latest)/soft-float.wasm
	$(prover_bin) $< -l $(output_latest)/soft-float.wasm -o $@ -b --allow-hostapi --require-success

contracts/test/prover/proofs/no-stack-pollution.json: $(arbitrator_cases)/no-stack-pollution.wasm $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --require-success

target/testdata/preimages.bin:
	mkdir -p `dirname $@`
	python3 scripts/create-test-preimages.py $@

contracts/test/prover/proofs/rust-%.json: $(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm $(prover_bin) $(arbitrator_wasm_libs) target/testdata/preimages.bin
	$(prover_bin) $< $(arbitrator_wasm_lib_flags) -o $@ -b --allow-hostapi --require-success --inbox-add-stub-headers --inbox $(arbitrator_cases)/rust/data/msg0.bin --inbox $(arbitrator_cases)/rust/data/msg1.bin --delayed-inbox $(arbitrator_cases)/rust/data/msg0.bin --delayed-inbox $(arbitrator_cases)/rust/data/msg1.bin --preimages target/testdata/preimages.bin

contracts/test/prover/proofs/go.json: $(arbitrator_cases)/go/testcase.wasm $(prover_bin) $(arbitrator_wasm_libs) target/testdata/preimages.bin $(arbitrator_tests_link_deps) $(arbitrator_cases)/user.wasm
	$(prover_bin) $< $(arbitrator_wasm_lib_flags) -o $@ -b --require-success --preimages target/testdata/preimages.bin  --stylus-modules $(arbitrator_cases)/user.wasm

# avoid testing user.wasm in onestepproofs. It can only run as stylus program.
contracts/test/prover/proofs/user.json:
	echo "[]" > $@

# avoid testing read-inboxmsg-10 in onestepproofs. It's used for go challenge testing.
contracts/test/prover/proofs/read-inboxmsg-10.json:
	echo "[]" > $@

contracts/test/prover/proofs/global-state.json:
	echo "[]" > $@

contracts/test/prover/proofs/forward-test.json: $(arbitrator_cases)/forward-test.wasm $(arbitrator_tests_forward_deps) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi $(patsubst %,-l %, $(arbitrator_tests_forward_deps))

contracts/test/prover/proofs/link.json: $(arbitrator_cases)/link.wasm $(arbitrator_tests_link_deps) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --stylus-modules $(arbitrator_tests_link_deps) --require-success

contracts/test/prover/proofs/dynamic.json: $(patsubst %,$(arbitrator_cases)/%.wasm, dynamic user) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --stylus-modules $(arbitrator_cases)/user.wasm --require-success

contracts/test/prover/proofs/bulk-memory.json: $(patsubst %,$(arbitrator_cases)/%.wasm, bulk-memory) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --stylus-modules $(arbitrator_cases)/user.wasm -b

contracts/test/prover/proofs/%.json: $(arbitrator_cases)/%.wasm $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi

# strategic rules to minimize dependency building

.make/lint: $(DEP_PREDICATE) build-node-deps $(ORDER_ONLY_PREDICATE) .make
	go run ./linters ./...
	golangci-lint run --fix
	yarn --cwd contracts solhint
	@touch $@

.make/fmt: $(DEP_PREDICATE) build-node-deps .make/yarndeps $(ORDER_ONLY_PREDICATE) .make
	golangci-lint run --disable-all -E gofmt --fix
	cargo fmt -p arbutil -p prover -p jit -p stylus --manifest-path arbitrator/Cargo.toml -- --check
	cargo fmt --all --manifest-path arbitrator/wasm-testsuite/Cargo.toml -- --check
	cargo fmt --all --manifest-path arbitrator/langs/rust/Cargo.toml -- --check
	yarn --cwd contracts prettier:solidity
	@touch $@

.make/test-go: $(DEP_PREDICATE) $(go_source) build-node-deps test-go-deps $(ORDER_ONLY_PREDICATE) .make
	gotestsum --format short-verbose --no-color=false
	@touch $@

.make/test-rust: $(DEP_PREDICATE) wasm-ci-build $(ORDER_ONLY_PREDICATE) .make
	cargo test --manifest-path arbitrator/Cargo.toml --release
	@touch $@

.make/solgen: $(DEP_PREDICATE) solgen/gen.go .make/solidity $(ORDER_ONLY_PREDICATE) .make
	mkdir -p solgen/go/
	go run solgen/gen.go
	@touch $@

.make/solidity: $(DEP_PREDICATE) safe-smart-account/contracts/*/*.sol safe-smart-account/contracts/*.sol contracts/src/*/*.sol .make/yarndeps $(ORDER_ONLY_PREDICATE) .make
	yarn --cwd safe-smart-account build
	yarn --cwd contracts build
	yarn --cwd contracts build:forge:yul
	@touch $@

.make/yarndeps: $(DEP_PREDICATE) contracts/package.json contracts/yarn.lock $(ORDER_ONLY_PREDICATE) .make
	yarn --cwd safe-smart-account install
	yarn --cwd contracts install
	@touch $@

.make/cbrotli-lib: $(DEP_PREDICATE) $(ORDER_ONLY_PREDICATE) .make
	test -f target/include/brotli/encode.h || ./scripts/build-brotli.sh -l
	test -f target/include/brotli/decode.h || ./scripts/build-brotli.sh -l
	test -f target/lib/libbrotlicommon-static.a || ./scripts/build-brotli.sh -l
	test -f target/lib/libbrotlienc-static.a || ./scripts/build-brotli.sh -l
	test -f target/lib/libbrotlidec-static.a || ./scripts/build-brotli.sh -l
	@touch $@

.make/cbrotli-wasm: $(DEP_PREDICATE) $(ORDER_ONLY_PREDICATE) .make
	test -f target/lib-wasm/libbrotlicommon-static.a || ./scripts/build-brotli.sh -w -d
	test -f target/lib-wasm/libbrotlienc-static.a || ./scripts/build-brotli.sh -w -d
	test -f target/lib-wasm/libbrotlidec-static.a || ./scripts/build-brotli.sh -w -d
	@touch $@

.make/wasm-lib: $(DEP_PREDICATE) arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/softfloat.a  $(ORDER_ONLY_PREDICATE) .make
	test -f arbitrator/wasm-libraries/soft-float/bindings32.o || ./scripts/build-brotli.sh -f -d -t ..
	test -f arbitrator/wasm-libraries/soft-float/bindings64.o || ./scripts/build-brotli.sh -f -d -t ..
	@touch $@

.make/machines: $(DEP_PREDICATE) $(ORDER_ONLY_PREDICATE) .make
	mkdir -p $(output_latest)
	touch $@

.make:
	mkdir .make


# Makefile settings

always:              # use this to force other rules to always build
.DELETE_ON_ERROR:    # causes a failure to delete its target
