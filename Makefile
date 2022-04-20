#
# Copyright 2020, Offchain Labs, Inc. All rights reserved.
#

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

precompile_names = AddressTable Aggregator BLS Debug FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubst %,./solgen/generated/%.go, $(precompile_names))

output_root=target

repo_dirs = arbos arbnode arbstate cmd precompiles solgen system_tests util validator wavmio
go_source = $(wildcard $(patsubst %,%/*.go, $(repo_dirs)) $(patsubst %,%/*/*.go, $(repo_dirs)))

das_rpc_files = das/dasrpc/wireFormat.pb.go das/dasrpc/wireFormat_grpc.pb.go

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)

replay_deps=arbos wavmio arbstate arbcompress solgen/go/node-interfacegen blsSignatures cmd/replay

replay_wasm=$(output_root)/machines/latest/replay.wasm

arbitrator_generated_header=$(output_root)/include/arbitrator.h
arbitrator_wasm_libs_nogo=$(output_root)/machines/latest/wasi_stub.wasm $(output_root)/machines/latest/host_io.wasm $(output_root)/machines/latest/soft-float.wasm
arbitrator_wasm_libs=$(arbitrator_wasm_libs_nogo) $(output_root)/machines/latest/go_stub.wasm $(output_root)/machines/latest/brotli.wasm
arbitrator_prover_lib=$(output_root)/lib/libprover.a
arbitrator_prover_bin=$(output_root)/bin/prover

arbitrator_tests_wat=$(wildcard arbitrator/prover/test-cases/*.wat)
arbitrator_tests_rust=$(wildcard arbitrator/prover/test-cases/rust/src/bin/*.rs)

arbitrator_test_wasms=$(patsubst %.wat,%.wasm, $(arbitrator_tests_wat)) $(patsubst arbitrator/prover/test-cases/rust/src/bin/%.rs,arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm, $(arbitrator_tests_rust)) arbitrator/prover/test-cases/go/main

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot

arbitrator_wasm_lib_flags_nogo=$(patsubst %, -l %, $(arbitrator_wasm_libs_nogo))
arbitrator_wasm_lib_flags=$(patsubst %, -l %, $(arbitrator_wasm_libs))

arbitrator_wasm_wasistub_files = $(wildcard arbitrator/wasm-libraries/wasi-stub/src/*/*)
arbitrator_wasm_gostub_files = $(wildcard arbitrator/wasm-libraries/go-stub/src/*/*)
arbitrator_wasm_hostio_files = $(wildcard arbitrator/wasm-libraries/host-io/src/*/*)

# user targets

push: lint test-go .make/fmt
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

all: build build-replay-env test-gen-proofs
	@touch .make/all

build: $(output_root)/bin/nitro $(output_root)/bin/deploy $(output_root)/bin/relay $(output_root)/bin/daserver $(output_root)/bin/seq-coordinator-invalidate
	@printf $(done)

build-node-deps: $(go_source) $(das_rpc_files) build-prover-header build-prover-lib .make/solgen .make/cbrotli-lib

test-go-deps: \
	build-replay-env \
	arbitrator/prover/test-cases/global-state.wasm \
	arbitrator/prover/test-cases/read-inboxmsg-10.wasm \
	arbitrator/prover/test-cases/global-state-wrapper.wasm \
	arbitrator/prover/test-cases/const.wasm

build-prover-header: $(arbitrator_generated_header)

build-prover-lib: $(arbitrator_prover_lib)

build-prover-bin: $(arbitrator_prover_bin)

build-replay-env: $(arbitrator_prover_bin) $(arbitrator_wasm_libs) $(replay_wasm) $(output_root)/machines/latest/machine.wavm.br

build-wasm-libs: $(arbitrator_wasm_libs)

build-wasm-bin: $(replay_wasm)

build-solidity: .make/solidity

$(das_rpc_files): das/wireFormat.proto
	cd das && protoc -I=. --go_out=.. --go-grpc_out=.. ./wireFormat.proto

contracts: .make/solgen
	@printf $(done)

format fmt: .make/fmt
	@printf $(done)

lint: .make/lint
	@printf $(done)

test-go: .make/test-go
	@printf $(done)

test-go-challenge: test-go-deps
	go test -v -timeout 120m ./system_tests/... -run TestFullChallenge -tags fullchallengetest
	@printf $(done)

test-go-redis: test-go-deps
	go test -p 1 -run TestSeqCoordinator -tags redistest ./system_tests/... ./arbnode/...
	@printf $(done)

test-gen-proofs: \
	$(patsubst arbitrator/prover/test-cases/%.wat,contracts/test/prover/proofs/%.json, $(arbitrator_tests_wat)) \
	$(patsubst arbitrator/prover/test-cases/rust/src/bin/%.rs,contracts/test/prover/proofs/rust-%.json, $(arbitrator_tests_rust)) \
	contracts/test/prover/proofs/go.json

wasm-ci-build: $(arbitrator_wasm_libs) $(arbitrator_test_wasms)
	@printf $(done)

clean:
	go clean -testcache
	rm -rf arbitrator/prover/test-cases/rust/target
	rm -f arbitrator/prover/test-cases/*.wasm
	rm -f arbitrator/prover/test-cases/go/main
	rm -rf $(output_root)
	rm -f contracts/test/prover/proofs/*.json
	rm -rf arbitrator/target
	rm -rf arbitrator/wasm-libraries/target
	rm -f arbitrator/wasm-libraries/soft-float/soft-float.wasm
	rm -f arbitrator/wasm-libraries/soft-float/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.a
	rm -f $(das_rpc_files)
	@rm -rf contracts/build contracts/cache solgen/go/
	@rm -f .make/*

docker:
	docker build -t nitro-node-slim --target nitro-node-slim .
	docker build -t nitro-node --target nitro-node .
	docker build -t nitro-node-dev --target nitro-node-dev .

# regular build rules

$(output_root)/bin/nitro: $(DEP_PREDICATE) build-node-deps
	go build -o $@ ./cmd/nitro

$(output_root)/bin/deploy: $(DEP_PREDICATE) build-node-deps
	go build -o $@ ./cmd/deploy

$(output_root)/bin/relay: $(DEP_PREDICATE) build-node-deps
	go build -o $@ ./cmd/relay

$(output_root)/bin/daserver: $(DEP_PREDICATE) build-node-deps
	go build -o $@ ./cmd/daserver

$(output_root)/bin/seq-coordinator-invalidate: $(DEP_PREDICATE) build-node-deps
	go build -o $@ ./cmd/seq-coordinator-invalidate

# recompile wasm, but don't change timestamp unless files differ
$(replay_wasm): $(DEP_PREDICATE) $(go_source) .make/solgen
	mkdir -p `dirname $(replay_wasm)`
	GOOS=js GOARCH=wasm go build -o $(output_root)/tmp/replay.wasm ./cmd/replay/...
	if ! diff -qN $(output_root)/tmp/replay.wasm $@ > /dev/null; then cp $(output_root)/tmp/replay.wasm $@; fi

$(arbitrator_prover_bin): $(DEP_PREDICATE) arbitrator/prover/src/*.rs arbitrator/prover/Cargo.toml
	mkdir -p `dirname $(arbitrator_prover_bin)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --bin prover
	install arbitrator/target/release/prover $@

$(arbitrator_prover_lib): $(DEP_PREDICATE) arbitrator/prover/src/*.rs arbitrator/prover/Cargo.toml
	mkdir -p `dirname $(arbitrator_prover_lib)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --lib
	install arbitrator/target/release/libprover.a $@

arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm: arbitrator/prover/test-cases/rust/src/bin/%.rs arbitrator/prover/test-cases/rust/src/lib.rs
	cargo build --manifest-path arbitrator/prover/test-cases/rust/Cargo.toml --release --target wasm32-wasi --bin $(patsubst arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm,%, $@)

arbitrator/prover/test-cases/go/main: arbitrator/prover/test-cases/go/main.go arbitrator/prover/test-cases/go/go.mod arbitrator/prover/test-cases/go/go.sum
	cd arbitrator/prover/test-cases/go && GOOS=js GOARCH=wasm go build main.go

$(arbitrator_generated_header): $(DEP_PREDICATE) arbitrator/prover/src/lib.rs arbitrator/prover/src/utils.rs
	@echo creating ${PWD}/$(arbitrator_generated_header)
	mkdir -p `dirname $(arbitrator_generated_header)`
	cd arbitrator && cbindgen --config cbindgen.toml --crate prover --output ../$(arbitrator_generated_header)

$(output_root)/machines/latest/wasi_stub.wasm: $(DEP_PREDICATE) $(arbitrator_wasm_wasistub_files)
	mkdir -p $(output_root)/machines/latest
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-unknown-unknown --package wasi-stub
	install arbitrator/wasm-libraries/target/wasm32-unknown-unknown/release/wasi_stub.wasm $@

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

$(output_root)/machines/latest/soft-float.wasm: $(DEP_PREDICATE) \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/softfloat.a
	mkdir -p $(output_root)/machines/latest
	wasm-ld \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat/build/Wasm-Clang/*.o \
		--no-entry -o $@ \
		--export wavm__f32_abs \
		--export wavm__f32_neg \
		--export wavm__f32_ceil \
		--export wavm__f32_floor \
		--export wavm__f32_trunc \
		--export wavm__f32_nearest \
		--export wavm__f32_sqrt \
		--export wavm__f32_add \
		--export wavm__f32_sub \
		--export wavm__f32_mul \
		--export wavm__f32_div \
		--export wavm__f32_min \
		--export wavm__f32_max \
		--export wavm__f32_copysign \
		--export wavm__f32_eq \
		--export wavm__f32_ne \
		--export wavm__f32_lt \
		--export wavm__f32_le \
		--export wavm__f32_gt \
		--export wavm__f32_ge \
		--export wavm__i32_trunc_f32_s \
		--export wavm__i32_trunc_f32_u \
		--export wavm__i64_trunc_f32_s \
		--export wavm__i64_trunc_f32_u \
		--export wavm__f32_convert_i32_s \
		--export wavm__f32_convert_i32_u \
		--export wavm__f32_convert_i64_s \
		--export wavm__f32_convert_i64_u \
		--export wavm__f64_abs \
		--export wavm__f64_neg \
		--export wavm__f64_ceil \
		--export wavm__f64_floor \
		--export wavm__f64_trunc \
		--export wavm__f64_nearest \
		--export wavm__f64_sqrt \
		--export wavm__f64_add \
		--export wavm__f64_sub \
		--export wavm__f64_mul \
		--export wavm__f64_div \
		--export wavm__f64_min \
		--export wavm__f64_max \
		--export wavm__f64_copysign \
		--export wavm__f64_eq \
		--export wavm__f64_ne \
		--export wavm__f64_lt \
		--export wavm__f64_le \
		--export wavm__f64_gt \
		--export wavm__f64_ge \
		--export wavm__i32_trunc_f64_s \
		--export wavm__i32_trunc_f64_u \
		--export wavm__i64_trunc_f64_s \
		--export wavm__i64_trunc_f64_u \
		--export wavm__f64_convert_i32_s \
		--export wavm__f64_convert_i32_u \
		--export wavm__f64_convert_i64_s \
		--export wavm__f64_convert_i64_u \
		--export wavm__f32_demote_f64 \
		--export wavm__f64_promote_f32

$(output_root)/machines/latest/go_stub.wasm: $(DEP_PREDICATE) $(wildcard arbitrator/wasm-libraries/go-stub/src/*/*)
	mkdir -p $(output_root)/machines/latest
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package go-stub
	install arbitrator/wasm-libraries/target/wasm32-wasi/release/go_stub.wasm $@

$(output_root)/machines/latest/host_io.wasm: $(DEP_PREDICATE) $(wildcard arbitrator/wasm-libraries/host-io/src/*/*)
	mkdir -p $(output_root)/machines/latest
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package host-io
	install arbitrator/wasm-libraries/target/wasm32-wasi/release/host_io.wasm $@

$(output_root)/machines/latest/brotli.wasm: $(DEP_PREDICATE) $(wildcard arbitrator/wasm-libraries/brotli/src/*/*) .make/cbrotli-wasm
	mkdir -p $(output_root)/machines/latest
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package brotli
	install arbitrator/wasm-libraries/target/wasm32-wasi/release/brotli.wasm $@

$(output_root)/machines/latest/machine.wavm.br: $(DEP_PREDICATE) $(arbitrator_prover_bin) $(arbitrator_wasm_libs) $(replay_wasm)
	$(arbitrator_prover_bin) $(replay_wasm) --generate-binaries $(output_root)/machines/latest -l $(output_root)/machines/latest/soft-float.wasm -l $(output_root)/machines/latest/wasi_stub.wasm -l $(output_root)/machines/latest/go_stub.wasm -l $(output_root)/machines/latest/host_io.wasm -l $(output_root)/machines/latest/brotli.wasm

arbitrator/prover/test-cases/%.wasm: arbitrator/prover/test-cases/%.wat
	wat2wasm $< -o $@

contracts/test/prover/proofs/float%.json: arbitrator/prover/test-cases/float%.wasm $(arbitrator_prover_bin) $(output_root)/machines/latest/soft-float.wasm
	$(arbitrator_prover_bin) $< -l $(output_root)/machines/latest/soft-float.wasm -o $@ -b --allow-hostapi --require-success --always-merkleize

contracts/test/prover/proofs/rust-%.json: arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm $(arbitrator_prover_bin) $(arbitrator_wasm_libs_nogo)
	$(arbitrator_prover_bin) $< $(arbitrator_wasm_lib_flags_nogo) -o $@ -b --allow-hostapi --require-success --inbox-add-stub-headers --inbox arbitrator/prover/test-cases/rust/data/msg0.bin --inbox arbitrator/prover/test-cases/rust/data/msg1.bin --delayed-inbox arbitrator/prover/test-cases/rust/data/msg0.bin --delayed-inbox arbitrator/prover/test-cases/rust/data/msg1.bin --preimages arbitrator/prover/test-cases/rust/data/preimages.bin

contracts/test/prover/proofs/go.json: arbitrator/prover/test-cases/go/main $(arbitrator_prover_bin) $(arbitrator_wasm_libs)
	$(arbitrator_prover_bin) $< $(arbitrator_wasm_lib_flags) -o $@ -i 5000000 --require-success

# avoid testing read-inboxmsg-10 in onestepproofs. It's used for go challenge testing.
contracts/test/prover/proofs/read-inboxmsg-10.json:
	echo "[]" > $@

contracts/test/prover/proofs/%.json: arbitrator/prover/test-cases/%.wasm $(arbitrator_prover_bin)
	$(arbitrator_prover_bin) $< -o $@ --allow-hostapi --always-merkleize

# strategic rules to minimize dependency building

.make/lint: $(DEP_PREDICATE) build-node-deps $(ORDER_ONLY_PREDICATE) .make
	golangci-lint run --fix
	yarn --cwd contracts solhint
	@touch $@

.make/fmt: $(DEP_PREDICATE) build-node-deps .make/yarndeps $(ORDER_ONLY_PREDICATE) .make
	golangci-lint run --disable-all -E gofmt --fix
	cargo fmt --all --manifest-path arbitrator/Cargo.toml -- --check
	yarn --cwd contracts prettier:solidity
	@touch $@

.make/test-go: $(DEP_PREDICATE) $(go_source) build-node-deps test-go-deps $(ORDER_ONLY_PREDICATE) .make
	gotestsum --format short-verbose
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
	@printf "%btesting cbrotli local build exists. If this step fails, run ./build-brotli.sh -l%b\n" $(color_pink) $(color_reset)
	test -f target/include/brotli/encode.h
	test -f target/include/brotli/decode.h
	test -f target/lib/libbrotlicommon-static.a
	test -f target/lib/libbrotlienc-static.a
	test -f target/lib/libbrotlidec-static.a
	@touch $@

.make/cbrotli-wasm: $(DEP_PREDICATE) $(ORDER_ONLY_PREDICATE) .make
	@printf "%btesting cbrotli wasm build exists. If this step fails, run ./build-brotli.sh -w%b\n" $(color_pink) $(color_reset)
	test -f target/lib-wasm/libbrotlicommon-static.a
	test -f target/lib-wasm/libbrotlienc-static.a
	test -f target/lib-wasm/libbrotlidec-static.a
	@touch $@

.make:
	mkdir .make


# Makefile settings

always:              # use this to force other rules to always build
.DELETE_ON_ERROR:    # causes a failure to delete its target
.PHONY: push all build build-node-deps test-go-deps build-prover-header build-prover-lib build-prover-bin build-replay-env build-solidity build-wasm-libs contracts format fmt lint test-go test-gen-proofs push clean docker
