#
# Copyright 2020, Offchain Labs, Inc. All rights reserved.
#

precompile_names = AddressTable Aggregator BLS Debug FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubst %,./solgen/generated/%.go, $(precompile_names))

arbitrator_output_root=arbitrator/target/env

repo_dirs = arbos arbnode arbstate cmd precompiles solgen system_tests util validator wavmio
go_source = $(wildcard $(patsubst %,%/*.go, $(repo_dirs)) $(patsubst %,%/*/*.go, $(repo_dirs)))

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)

replay_wasm=$(arbitrator_output_root)/lib/replay.wasm

arbitrator_generated_header=$(arbitrator_output_root)/include/arbitrator.h
arbitrator_wasm_libs_nogo=$(arbitrator_output_root)/lib/wasi_stub.wasm $(arbitrator_output_root)/lib/host_io.wasm $(arbitrator_output_root)/lib/soft-float.wasm
arbitrator_wasm_libs=$(arbitrator_wasm_libs_nogo) $(arbitrator_output_root)/lib/go_stub.wasm
arbitrator_prover_lib=$(arbitrator_output_root)/lib/libprover.a
arbitrator_prover_bin=$(arbitrator_output_root)/bin/prover

arbitrator_tests_wat=$(wildcard arbitrator/prover/test-cases/*.wat)
arbitrator_tests_rust=$(wildcard arbitrator/prover/test-cases/rust/src/bin/*.rs)

arbitrator_test_wasms=$(patsubst %.wat,%.wasm, $(arbitrator_tests_wat)) $(patsubst arbitrator/prover/test-cases/rust/src/bin/%.rs,arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm, $(arbitrator_tests_rust)) arbitrator/prover/test-cases/go/main

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot

arbitrator_wasm_lib_flags_nogo=$(patsubst %, -l %, $(arbitrator_wasm_libs_nogo))
arbitrator_wasm_lib_flags=$(patsubst %, -l %, $(arbitrator_wasm_libs))
# user targets

.DELETE_ON_ERROR: # causes a failure to delete its target
.PHONY: push all build build-node-deps build-replay-env contracts format fmt lint test-go test-gen-proofs push clean docker

push: lint test-go
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

all: node build-replay-env test-gen-proofs
	@touch .make/all

build: node
	@printf $(done)

build-node-deps: $(go_source) $(arbitrator_generated_header) $(arbitrator_prover_lib) .make/solgen

build-replay-env: $(arbitrator_prover_bin) $(arbitrator_wasm_libs) $(replay_wasm)

contracts: .make/solgen
	@printf $(done)

format fmt: .make/fmt
	@printf $(done)

lint: .make/lint
	@printf $(done)

test-go: .make/test-go
	@printf $(done)

test-gen-proofs: \
	$(patsubst arbitrator/prover/test-cases/%.wat,solgen/test/proofs/%.json, $(arbitrator_tests_wat)) \
	$(patsubst arbitrator/prover/test-cases/rust/src/bin/%.rs,solgen/test/proofs/rust-%.json, $(arbitrator_tests_rust)) \
	solgen/test/proofs/go.json

clean:
	go clean -testcache
	rm -rf arbitrator/prover/test-cases/rust/target
	rm -f arbitrator/prover/test-cases/*.wasm
	rm -f arbitrator/prover/test-cases/go/main
	rm -rf $(arbitrator_output_root)
	rm -f solgen/test/proofs/*.json
	rm -rf arbitrator/target
	rm -rf arbitrator/wasm-libraries/target
	rm -f arbitrator/wasm-libraries/soft-float/soft-float.wasm
	rm -f arbitrator/wasm-libraries/soft-float/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.o
	rm -f arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.a
	@rm -rf solgen/artifacts solgen/cache solgen/go/
	@rm -f .make/*

docker:
	docker build -t nitro-node .

# regular build rules

node: build-node-deps
	go build ./cmd/node

$(replay_wasm): build-node-deps
	GOOS=js GOARCH=wasm go build -o $@ ./cmd/replay/...

$(arbitrator_prover_bin): arbitrator/prover/src/*.rs arbitrator/prover/Cargo.toml
	mkdir -p `dirname $(arbitrator_prover_bin)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --bin prover
	install -D arbitrator/target/release/prover $@

$(arbitrator_prover_lib): arbitrator/prover/src/*.rs arbitrator/prover/Cargo.toml
	mkdir -p `dirname $(arbitrator_prover_lib)`
	cargo build --manifest-path arbitrator/Cargo.toml --release --lib
	install -D arbitrator/target/release/libprover.a $@

arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm: arbitrator/prover/test-cases/rust/src/bin/%.rs arbitrator/prover/test-cases/rust/src/lib.rs
	cargo build --manifest-path arbitrator/prover/test-cases/rust/Cargo.toml --release --target wasm32-wasi --bin $(patsubst arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm,%, $@)

arbitrator/prover/test-cases/go/main: arbitrator/prover/test-cases/go/main.go arbitrator/prover/test-cases/go/go.mod arbitrator/prover/test-cases/go/go.sum
	cd arbitrator/prover/test-cases/go && GOOS=js GOARCH=wasm go build main.go

$(arbitrator_generated_header): arbitrator/prover/src/lib.rs arbitrator/prover/src/utils.rs
	@echo creating ${PWD}/$(arbitrator_generated_header)
	mkdir -p `dirname $(arbitrator_generated_header)`
	cd arbitrator && cbindgen --config cbindgen.toml --crate prover --output ../$(arbitrator_generated_header)

$(arbitrator_output_root)/lib/wasi_stub.wasm: arbitrator/wasm-libraries/wasi-stub/src/**
	mkdir -p $(arbitrator_output_root)/lib
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-unknown-unknown --package wasi-stub
	install -D arbitrator/wasm-libraries/target/wasm32-unknown-unknown/release/wasi_stub.wasm $@

arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/softfloat.a: \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/Makefile \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/platform.h \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/source/*.c \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/source/include/*.h \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.c \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.h
	cd arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang && make $(MAKEFLAGS)

arbitrator/wasm-libraries/soft-float/bindings%.o: arbitrator/wasm-libraries/soft-float/bindings%.c
	clang $< --sysroot $(WASI_SYSROOT) -I arbitrator/wasm-libraries/soft-float/SoftFloat-3e/source/include -target wasm32-wasi -Wconversion -c -o $@

$(arbitrator_output_root)/lib/soft-float.wasm: \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/softfloat.a
	mkdir -p $(arbitrator_output_root)/lib
	wasm-ld \
		arbitrator/wasm-libraries/soft-float/bindings32.o \
		arbitrator/wasm-libraries/soft-float/bindings64.o \
		arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.o \
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


$(arbitrator_output_root)/lib/go_stub.wasm: arbitrator/wasm-libraries/go-stub/src/**
	mkdir -p $(arbitrator_output_root)/lib
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package go-stub
	install -D arbitrator/wasm-libraries/target/wasm32-wasi/release/go_stub.wasm $@

$(arbitrator_output_root)/lib/host_io.wasm: arbitrator/wasm-libraries/host-io/src/**
	mkdir -p $(arbitrator_output_root)/lib
	cargo build --manifest-path arbitrator/wasm-libraries/Cargo.toml --release --target wasm32-wasi --package host-io
	install -D arbitrator/wasm-libraries/target/wasm32-wasi/release/host_io.wasm $@

arbitrator/prover/test-cases/%.wasm: arbitrator/prover/test-cases/%.wat
	wat2wasm $< -o $@

solgen/test/proofs/%.json: arbitrator/prover/test-cases/%.wasm $(arbitrator_prover_bin)
	$(arbitrator_prover_bin) $< -o $@ --always-merkleize

solgen/test/proofs/float%.json: arbitrator/prover/test-cases/float%.wasm $(arbitrator_prover_bin) $(arbitrator_output_root)/lib/soft-float.wasm
	$(arbitrator_prover_bin) $< -l $(arbitrator_output_root)/lib/soft-float.wasm -o $@ -b --always-merkleize

solgen/test/proofs/rust-%.json: arbitrator/prover/test-cases/rust/target/wasm32-wasi/release/%.wasm $(arbitrator_prover_bin) $(arbitrator_wasm_libs_nogo)
	$(arbitrator_prover_bin) $< $(arbitrator_wasm_lib_flags_nogo) -o $@ -b --allow-hostapi --inbox-add-stub-headers --inbox arbitrator/prover/test-cases/rust/messages/msg0.bin --inbox arbitrator/prover/test-cases/rust/messages/msg1.bin --delayed-inbox arbitrator/prover/test-cases/rust/messages/msg0.bin --delayed-inbox arbitrator/prover/test-cases/rust/messages/msg1.bin

solgen/test/proofs/go.json: arbitrator/prover/test-cases/go/main $(arbitrator_prover_bin) $(arbitrator_wasm_libs)
	$(arbitrator_prover_bin) $< $(arbitrator_wasm_lib_flags) -o $@ -i 5000000

# strategic rules to minimize dependency building

.make/lint: build-node-deps | .make
	golangci-lint run --fix
	@touch $@

.make/fmt: build-node-deps | .make
	golangci-lint run --disable-all -E gofmt --fix
	cargo fmt --all --manifest-path arbitrator/Cargo.toml -- --check
	@touch $@

.make/test-go: $(go_source) build-node-deps | .make
	gotestsum --format short-verbose
	@touch $@

.make/solgen: solgen/gen.go .make/solidity | .make
	mkdir -p solgen/go/
	go run solgen/gen.go
	@touch $@

.make/solidity: solgen/src/*/*.sol .make/yarndeps | .make
	yarn --cwd solgen build
	@touch $@

.make/yarndeps: solgen/package.json solgen/yarn.lock | .make
	yarn --cwd solgen install
	@touch $@

.make:
	mkdir .make


# Makefile settings

always:              # use this to force other rules to always build
.DELETE_ON_ERROR:    # causes a failure to delete its target
