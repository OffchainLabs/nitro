# Copyright 2021-2023, Offchain Labs, Inc.
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
 GOLANG_PARAMS = -ldflags="$(GOLANG_LDFLAGS)"
endif

precompile_names = AddressTable Aggregator BLS Debug FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubst %,./solgen/generated/%.go, $(precompile_names))

output_root=target
output_latest=$(output_root)/machines/latest

repo_dirs = arbos arbnode arbutil arbstate cmd das precompiles solgen system_tests util validator wavmio
go_source.go = $(wildcard $(patsubst %,%/*.go, $(repo_dirs)) $(patsubst %,%/*/*.go, $(repo_dirs)))
go_source.s  = $(wildcard $(patsubst %,%/*.s, $(repo_dirs)) $(patsubst %,%/*/*.s, $(repo_dirs)))
go_source = $(go_source.go) $(go_source.s)

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)

replay_deps=arbos wavmio arbstate arbcompress solgen/go/node-interfacegen blsSignatures cmd/replay

replay_wasm=$(output_latest)/replay.wasm

arbitrator_generated_header=$(output_root)/include/arbitrator.h
arbitrator_wasm_libs_nogo=$(patsubst %, $(output_root)/machines/latest/%.wasm, wasi_stub host_io soft-float)
arbitrator_wasm_libs=$(arbitrator_wasm_libs_nogo) $(patsubst %,$(output_root)/machines/latest/%.wasm, go_stub brotli forward user_host)
arbitrator_stylus_lib=$(output_root)/lib/libstylus.a
prover_bin=$(output_root)/bin/prover
arbitrator_jit=$(output_root)/bin/jit

arbitrator_cases=arbitrator/prover/test-cases

arbitrator_tests_wat=$(wildcard $(arbitrator_cases)/*.wat)
arbitrator_tests_rust=$(wildcard $(arbitrator_cases)/rust/src/bin/*.rs)

arbitrator_test_wasms=$(patsubst %.wat,%.wasm, $(arbitrator_tests_wat)) $(patsubst $(arbitrator_cases)/rust/src/bin/%.rs,$(arbitrator_cases)/rust/target/wasm32-wasi/release/%.wasm, $(arbitrator_tests_rust)) $(arbitrator_cases)/go/main

arbitrator_tests_link_info = $(shell cat $(arbitrator_cases)/link.txt | xargs)
arbitrator_tests_link_deps = $(patsubst %,$(arbitrator_cases)/%.wasm, $(arbitrator_tests_link_info))

arbitrator_tests_forward_wats = $(wildcard $(arbitrator_cases)/forward/*.wat)
arbitrator_tests_forward_deps = $(arbitrator_tests_forward_wats:wat=wasm)

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot

arbitrator_wasm_lib_flags_nogo=$(patsubst %, -l %, $(arbitrator_wasm_libs_nogo))
arbitrator_wasm_lib_flags=$(patsubst %, -l %, $(arbitrator_wasm_libs))

rust_arbutil_files = $(wildcard arbitrator/arbutil/src/*.* arbitrator/arbutil/src/*/*.* arbitrator/arbutil/*.toml)

prover_direct_includes = $(patsubst %,$(output_latest)/%.wasm, forward forward_stub)
prover_src = arbitrator/prover/src
rust_prover_files = $(wildcard $(prover_src)/*.* $(prover_src)/*/*.* arbitrator/prover/*.toml) $(rust_arbutil_files) $(prover_direct_includes)

wasm_lib = arbitrator/wasm-libraries
wasm_lib_deps = $(wildcard $(wasm_lib)/$(1)/*.toml $(wasm_lib)/$(1)/src/*.rs $(wasm_lib)/$(1)/*.rs) $(rust_arbutil_files) .make/machines
wasm_lib_go_abi = $(call wasm_lib_deps,go-abi) $(go_js_files)
wasm_lib_forward = $(call wasm_lib_deps,forward)
wasm_lib_user_host_trait = $(call wasm_lib_deps,user-host-trait)
wasm_lib_user_host = $(call wasm_lib_deps,user-host) $(wasm_lib_user_host_trait)

forward_dir = $(wasm_lib)/forward

stylus_files = $(wildcard $(stylus_dir)/*.toml $(stylus_dir)/src/*.rs) $(wasm_lib_user_host_trait) $(rust_prover_files)

jit_dir = arbitrator/jit
go_js_files = $(wildcard arbitrator/wasm-libraries/go-js/*.toml arbitrator/wasm-libraries/go-js/src/*.rs)
jit_files = $(wildcard $(jit_dir)/*.toml $(jit_dir)/*.rs $(jit_dir)/src/*.rs $(jit_dir)/src/*/*.rs) $(stylus_files) $(go_js_files)

go_js_test_dir = arbitrator/wasm-libraries/go-js-test
go_js_test_files = $(wildcard $(go_js_test_dir)/*.go $(go_js_test_dir)/*.mod)
go_js_test = $(go_js_test_dir)/js-test.wasm
go_js_test_libs = $(patsubst %, $(output_latest)/%.wasm, soft-float wasi_stub go_stub)

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

cargo_nightly = cargo +nightly build -Z build-std=std,panic_abort -Z build-std-features=panic_immediate_abort

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
stylus_test_evm-data_wasm         = $(call get_stylus_test_wasm,evm-data)
stylus_test_evm-data_src          = $(call get_stylus_test_rust,evm-data)
stylus_test_sdk-storage_wasm      = $(call get_stylus_test_wasm,sdk-storage)
stylus_test_sdk-storage_src       = $(call get_stylus_test_rust,sdk-storage)
stylus_test_erc20_wasm            = $(call get_stylus_test_wasm,erc20)
stylus_test_erc20_src             = $(call get_stylus_test_rust,erc20)
stylus_test_read-return-data_wasm = $(call get_stylus_test_wasm,read-return-data)
stylus_test_read-return-data_src  = $(call get_stylus_test_rust,read-return-data)

stylus_test_wasms = $(stylus_test_keccak_wasm) $(stylus_test_keccak-100_wasm) $(stylus_test_fallible_wasm) $(stylus_test_storage_wasm) $(stylus_test_multicall_wasm) $(stylus_test_log_wasm) $(stylus_test_create_wasm) $(stylus_test_sdk-storage_wasm) $(stylus_test_erc20_wasm) $(stylus_test_read-return-data_wasm) $(stylus_test_evm-data_wasm) $(stylus_test_bfs:.b=.wasm)
stylus_benchmarks = $(wildcard $(stylus_dir)/*.toml $(stylus_dir)/src/*.rs) $(stylus_test_wasms)

# user targets

push: lint test-go .make/fmt
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

all: build build-replay-env test-gen-proofs
	@touch .make/all

build: $(patsubst %,$(output_root)/bin/%, nitro deploy relay daserver datool seq-coordinator-invalidate nitro-val)
	@printf $(done)

build-node-deps: $(go_source) build-prover-header build-prover-lib build-jit .make/solgen .make/cbrotli-lib

test-go-deps: \
	build-replay-env \
	$(stylus_test_wasms) \
	$(arbitrator_stylus_lib) \
	$(patsubst %,$(arbitrator_cases)/%.wasm, global-state read-inboxmsg-10 global-state-wrapper const)

build-prover-header: $(arbitrator_generated_header)

build-prover-lib: $(arbitrator_stylus_lib)

build-prover-bin: $(prover_bin)

build-jit: $(arbitrator_jit)

build-replay-env: $(prover_bin) $(arbitrator_jit) $(arbitrator_wasm_libs) $(replay_wasm) $(output_latest)/machine.wavm.br

build-wasm-libs: $(arbitrator_wasm_libs)

build-wasm-bin: $(replay_wasm)

build-solidity: .make/solidity

contracts: .make/solgen
	@printf $(done)

format fmt: .make/fmt
	@printf $(done)

lint: .make/lint
	@printf $(done)

stylus-benchmarks: $(stylus_benchmarks)
	cargo test --manifest-path $< --release --features benchmark benchmark_ -- --nocapture
	@printf $(done)

test-go: .make/test-go
	@printf $(done)

test-go-challenge: test-go-deps
	go test -v -timeout 120m ./system_tests/... -run TestChallenge -tags challengetest
	@printf $(done)

test-go-stylus: test-go-deps
	go test -v -timeout 120m ./system_tests/... -run TestProgramArbitrator -tags stylustest
	@printf $(done)

test-go-redis: test-go-deps
	TEST_REDIS=redis://localhost:6379/0 go test -p 1 -run TestRedis ./system_tests/... ./arbnode/...
	@printf $(done)

test-js-runtime: $(go_js_test) $(arbitrator_jit) $(go_js_test_libs) $(prover_bin)
	./target/bin/jit --binary $< --go-arg --cranelift --require-success
	$(prover_bin) $< -s 90000000 -l $(go_js_test_libs) --require-success

test-gen-proofs: \
        $(arbitrator_test_wasms) \
	$(patsubst $(arbitrator_cases)/%.wat,contracts/test/prover/proofs/%.json, $(arbitrator_tests_wat)) \
	$(patsubst $(arbitrator_cases)/rust/src/bin/%.rs,contracts/test/prover/proofs/rust-%.json, $(arbitrator_tests_rust)) \
	contracts/test/prover/proofs/go.json

wasm-ci-build: $(arbitrator_wasm_libs) $(arbitrator_test_wasms) $(stylus_test_wasms) $(output_latest)/user_test.wasm
	@printf $(done)

clean:
	go clean -testcache
	rm -rf $(arbitrator_cases)/rust/target
	rm -f $(arbitrator_cases)/*.wasm $(arbitrator_cases)/go/main
	rm -rf arbitrator/wasm-testsuite/tests
	rm -rf $(output_root)
	rm -f contracts/test/prover/proofs/*.json contracts/test/prover/spec-proofs/*.json
	rm -rf arbitrator/target
	rm -rf arbitrator/wasm-libraries/target
	rm -f arbitrator/wasm-libraries/soft-float/soft-float.wasm
	rm -f arbitrator/wasm-libraries/soft-float/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.a
	rm -rf arbitrator/stylus/tests/*/target/ arbitrator/stylus/tests/*/*.wasm
	@rm -rf contracts/build contracts/cache solgen/go/
	@rm -f .make/*

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

# recompile wasm, but don't change timestamp unless files differ
$(replay_wasm): $(DEP_PREDICATE) $(go_source) .make/solgen
	mkdir -p `dirname $(replay_wasm)`
	GOOS=js GOARCH=wasm go build -o $@ ./cmd/replay/...

$(prover_bin): $(DEP_PREDICATE) $(rust_prover_files)
	mkdir -p `dirname $(prover_bin)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --bin prover ${CARGOFLAGS}
	install arbitrator/target/release/prover $@

$(arbitrator_stylus_lib): $(DEP_PREDICATE) $(stylus_files)
	mkdir -p `dirname $(arbitrator_stylus_lib)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --lib -p stylus ${CARGOFLAGS}
	install arbitrator/target/release/libstylus.a $@

$(arbitrator_jit): $(DEP_PREDICATE) .make/cbrotli-lib $(jit_files)
	mkdir -p `dirname $(arbitrator_jit)`
	cargo build --manifest-path arbitrator/Cargo.toml --release -p jit ${CARGOFLAGS}
	install arbitrator/target/release/jit $@

$(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm: $(arbitrator_cases)/rust/src/bin/%.rs $(arbitrator_cases)/rust/src/lib.rs
	cargo build --manifest-path $(arbitrator_cases)/rust/Cargo.toml --release --target wasm32-wasi --bin $(patsubst $(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm,%, $@)

$(arbitrator_cases)/go/main: $(arbitrator_cases)/go/main.go
	cd $(arbitrator_cases)/go && GOOS=js GOARCH=wasm go build main.go

$(arbitrator_generated_header): $(DEP_PREDICATE) $(stylus_files)
	@echo creating ${PWD}/$(arbitrator_generated_header)
	mkdir -p `dirname $(arbitrator_generated_header)`
	cd arbitrator/stylus && cbindgen --config cbindgen.toml --crate stylus --output ../../$(arbitrator_generated_header)
	@touch -c $@ # cargo might decide to not rebuild the header

$(output_latest)/wasi_stub.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,wasi-stub)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-unknown-unknown --package wasi-stub
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

$(output_latest)/go_stub.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,go-stub) $(wasm_lib_go_abi) $(go_js_files)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package go-stub
	install arbitrator/wasm-libraries/$(wasm32_wasi)/go_stub.wasm $@

$(output_latest)/host_io.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,host-io) $(wasm_lib_go_abi)
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package host-io
	install arbitrator/wasm-libraries/$(wasm32_wasi)/host_io.wasm $@

$(output_latest)/user_host.wasm: $(DEP_PREDICATE) $(wasm_lib_user_host) $(rust_prover_files) $(output_latest)/forward_stub.wasm .make/machines
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package user-host
	install arbitrator/wasm-libraries/$(wasm32_wasi)/user_host.wasm $@

$(output_latest)/user_test.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,user-test) $(rust_prover_files) .make/machines
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package user-test
	install arbitrator/wasm-libraries/$(wasm32_wasi)/user_test.wasm $@

$(output_latest)/brotli.wasm: $(DEP_PREDICATE) $(call wasm_lib_deps,brotli) $(wasm_lib_go_abi) .make/cbrotli-wasm
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package brotli
	install arbitrator/wasm-libraries/$(wasm32_wasi)/brotli.wasm $@

$(output_latest)/forward.wasm: $(DEP_PREDICATE) $(wasm_lib_forward) .make/machines
	cargo run --manifest-path $(forward_dir)/Cargo.toml -- --path $(forward_dir)/forward.wat
	wat2wasm $(wasm_lib)/forward/forward.wat -o $@

$(output_latest)/forward_stub.wasm: $(DEP_PREDICATE) $(wasm_lib_forward) .make/machines
	cargo run --manifest-path $(forward_dir)/Cargo.toml -- --path $(forward_dir)/forward_stub.wat --stub
	wat2wasm $(wasm_lib)/forward/forward_stub.wat -o $@

$(output_latest)/machine.wavm.br: $(DEP_PREDICATE) $(prover_bin) $(arbitrator_wasm_libs) $(replay_wasm)
	$(prover_bin) $(replay_wasm) --generate-binaries $(output_latest) \
	$(patsubst %,-l $(output_latest)/%.wasm, forward soft-float wasi_stub go_stub host_io user_host brotli)

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

$(go_js_test): $(go_js_test_files)
	cd $(go_js_test_dir) && GOOS=js GOARCH=wasm go build -o js-test.wasm

contracts/test/prover/proofs/float%.json: $(arbitrator_cases)/float%.wasm $(prover_bin) $(output_latest)/soft-float.wasm
	$(prover_bin) $< -l $(output_latest)/soft-float.wasm -o $@ -b --allow-hostapi --require-success --always-merkleize

contracts/test/prover/proofs/no-stack-pollution.json: $(arbitrator_cases)/no-stack-pollution.wasm $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --require-success --always-merkleize

contracts/test/prover/proofs/rust-%.json: $(arbitrator_cases)/rust/$(wasm32_wasi)/%.wasm $(prover_bin) $(arbitrator_wasm_libs_nogo)
	$(prover_bin) $< $(arbitrator_wasm_lib_flags_nogo) -o $@ -b --allow-hostapi --require-success --inbox-add-stub-headers --inbox $(arbitrator_cases)/rust/data/msg0.bin --inbox $(arbitrator_cases)/rust/data/msg1.bin --delayed-inbox $(arbitrator_cases)/rust/data/msg0.bin --delayed-inbox $(arbitrator_cases)/rust/data/msg1.bin --preimages $(arbitrator_cases)/rust/data/preimages.bin

contracts/test/prover/proofs/go.json: $(arbitrator_cases)/go/main $(prover_bin) $(arbitrator_wasm_libs)
	$(prover_bin) $< $(arbitrator_wasm_lib_flags) -o $@ -i 5000000 --require-success

# avoid testing read-inboxmsg-10 in onestepproofs. It's used for go challenge testing.
contracts/test/prover/proofs/read-inboxmsg-10.json:
	echo "[]" > $@

contracts/test/prover/proofs/global-state.json:
	echo "[]" > $@

contracts/test/prover/proofs/forward-test.json: $(arbitrator_cases)/forward-test.wasm $(arbitrator_tests_forward_deps) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --always-merkleize $(patsubst %,-l %, $(arbitrator_tests_forward_deps))

contracts/test/prover/proofs/link.json: $(arbitrator_cases)/link.wasm $(arbitrator_tests_link_deps) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --always-merkleize --stylus-modules $(arbitrator_tests_link_deps)

contracts/test/prover/proofs/dynamic.json: $(patsubst %,$(arbitrator_cases)/%.wasm, dynamic user) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --always-merkleize --stylus-modules $(arbitrator_cases)/user.wasm

contracts/test/prover/proofs/bulk-memory.json: $(patsubst %,$(arbitrator_cases)/%.wasm, bulk-memory) $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --always-merkleize --stylus-modules $(arbitrator_cases)/user.wasm -b

contracts/test/prover/proofs/%.json: $(arbitrator_cases)/%.wasm $(prover_bin)
	$(prover_bin) $< -o $@ --allow-hostapi --always-merkleize

# strategic rules to minimize dependency building

.make/lint: $(DEP_PREDICATE) build-node-deps $(ORDER_ONLY_PREDICATE) .make
	go run linter/pointercheck/pointer.go ./...
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

.make/solgen: $(DEP_PREDICATE) solgen/gen.go .make/solidity $(ORDER_ONLY_PREDICATE) .make
	mkdir -p solgen/go/
	go run solgen/gen.go
	@touch $@

.make/solidity: $(DEP_PREDICATE) contracts/src/*/*.sol .make/yarndeps $(ORDER_ONLY_PREDICATE) .make
	yarn --cwd contracts build
	@touch $@

.make/yarndeps: $(DEP_PREDICATE) contracts/package.json contracts/yarn.lock $(ORDER_ONLY_PREDICATE) .make
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
.PHONY: push all build build-node-deps test-go-deps build-prover-header build-prover-lib build-prover-bin build-jit build-replay-env build-solidity build-wasm-libs contracts format fmt lint stylus-benchmarks test-go test-gen-proofs push clean docker
