#
# Copyright 2020, Offchain Labs, Inc. All rights reserved.
#

precompile_names = AddressTable Aggregator BLS Debug FunctionTable GasInfo Info osTest Owner RetryableTx Statistics Sys
precompiles = $(patsubst %,./solgen/generated/%.go, $(precompile_names))

repo_dirs = arbos arbnode arbstate cmd precompiles solgen system_tests wavmio
go_source = $(wildcard $(patsubst %,%/*.go, $(repo_dirs)) $(patsubst %,%/*/*.go, $(repo_dirs)))

color_pink = "\e[38;5;161;1m"
color_reset = "\e[0;0m"

done = "%bdone!%b\n" $(color_pink) $(color_reset)

arbitrator_inputs=$(wildcard prover/test-cases/*.wat)
arbitrator_rust_bin_sources=$(wildcard prover/test-cases/rust/src/bin/*.rs)
arbitrator_generated_arbitrator_header=prover/generated-inc/arbitrator.h
arbitrator_outputs=$(patsubst prover/test-cases/%.wat,rollup/test/proofs/%.json, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,rollup/test/proofs/rust-%.json, $(rust_bin_sources)) rollup/test/proofs/go.json $(generated_arbitrator_header)
arbitrator_wasms=$(patsubst %.wat,%.wasm, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm, $(rust_bin_sources)) prover/test-cases/go/main

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot


# user targets

.DELETE_ON_ERROR: # causes a failure to delete its target
.PHONY: all clean

.make/all: always .make/solgen .make/solidity .make/test $(arbitrator_wasms) $(arbitrator_outputs)
	@printf "%bdone building %s%b\n" $(color_pink) $$(expr $$(echo $? | wc -w) - 1) $(color_reset)
	@touch .make/all

build: $(go_source) .make/solgen .make/solidity
	@printf $(done)

contracts: .make/solgen
	@printf $(done)

format fmt: .make/fmt
	@printf $(done)

lint: .make/lint
	@printf $(done)

test: .make/test
	gotestsum --format short-verbose
	@printf $(done)

push: .make/push
	@printf "%bready for push!%b\n" $(color_pink) $(color_reset)

clean:
	go clean -testcache
	rm -rf prover/test-cases/rust/target
	rm -f prover/test-cases/*.wasm
	rm -f prover/test-cases/go/main
	rm -rf `dirname $(generated_arbitrator_header)`
	rm -f rollup/test/proofs/*.json
	rm -rf wasm-libraries/target
	rm -f wasm-libraries/soft-float/soft-float.wasm
	rm -f wasm-libraries/soft-float/*.o
	rm -f wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.o
	rm -f wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.a
	@rm -rf solgen/artifacts solgen/cache solgen/go/
	@rm -f .make/*

docker:
	docker build -t nitro-node .

# regular build rules

arbitrator/prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm: prover/test-cases/rust/src/bin/%.rs prover/test-cases/rust/src/lib.rs
	cd prover/test-cases/rust && cargo build --target wasm32-wasi --bin $(patsubst arbitrator/prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm,%, $@)

arbitrator/prover/test-cases/go/main: prover/test-cases/go/main.go prover/test-cases/go/go.mod prover/test-cases/go/go.sum
	cd prover/test-cases/go && GOOS=js GOARCH=wasm go build main.go

arbitrator/$(arbitrator_generated_arbitrator_header): prover/src/lib.rs prover/src/utils.rs
	cbindgen --config cbindgen.toml --crate prover --output $(arbitrator_generated_arbitrator_header)

arbitrator/wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm: wasm-libraries/wasi-stub/src/**
	cd wasm-libraries && cargo build --target wasm32-unknown-unknown --package wasi-stub

arbitrator/wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/softfloat.a: \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/Makefile \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/platform.h \
		wasm-libraries/soft-float/SoftFloat-3e/source/*.c \
		wasm-libraries/soft-float/SoftFloat-3e/source/include/*.h \
		wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.c \
		wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.h
	cd wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang && make $(MAKEFLAGS)

arbitrator/wasm-libraries/soft-float/bindings%.o: wasm-libraries/soft-float/bindings%.c
	clang $< --sysroot $(WASI_SYSROOT) -I wasm-libraries/soft-float/SoftFloat-3e/source/include -target wasm32-wasi -Wconversion -c -o $@

arbitrator/wasm-libraries/soft-float/soft-float.wasm: \
		wasm-libraries/soft-float/bindings32.o \
		wasm-libraries/soft-float/bindings64.o \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/softfloat.a
	wasm-ld \
		wasm-libraries/soft-float/bindings32.o \
		wasm-libraries/soft-float/bindings64.o \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.o \
		--no-entry -o wasm-libraries/soft-float/soft-float.wasm \
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

arbitrator/wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm: wasm-libraries/go-stub/src/**
	cd wasm-libraries && cargo build --target wasm32-wasi --package go-stub

arbitrator/wasm-libraries/target/wasm32-wasi/debug/host_io.wasm: wasm-libraries/host-io/src/**
	cd wasm-libraries && cargo build --target wasm32-wasi --package host-io

arbitrator/prover/test-cases/%.wasm: prover/test-cases/%.wat
	wat2wasm $< -o $@

arbitrator/rollup/test/proofs/%.json: prover/test-cases/%.wasm prover/src/**
	cargo run -p prover -- $< -o $@ --always-merkleize

arbitrator/rollup/test/proofs/float%.json: prover/test-cases/float%.wasm wasm-libraries/soft-float/soft-float.wasm prover/src/**
	cargo run --release -p prover -- $< -l wasm-libraries/soft-float/soft-float.wasm -o $@ -b --always-merkleize

arbitrator/rollup/test/proofs/rust-%.json: \
		prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm \
		wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm \
		wasm-libraries/soft-float/soft-float.wasm prover/src/**
	cargo run --release -p prover -- $< -l wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm -l wasm-libraries/soft-float/soft-float.wasm -o $@ -b --allow-hostapi --inbox-add-stub-headers --inbox prover/test-cases/rust/messages/msg0.bin --inbox prover/test-cases/rust/messages/msg1.bin --delayed-inbox prover/test-cases/rust/messages/msg0.bin --delayed-inbox prover/test-cases/rust/messages/msg1.bin

arbitrator/rollup/test/proofs/go.json: \
		prover/test-cases/go/main \
		wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm \
		wasm-libraries/soft-float/soft-float.wasm prover/src/** \
		wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm \
		wasm-libraries/target/wasm32-wasi/debug/host_io.wasm
	cargo run --release -p prover -- $< -l wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm -l wasm-libraries/soft-float/soft-float.wasm -l wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm -o $@ -i 5000000


# strategic rules to minimize dependency building

.make/push: .make/lint | .make
	make $(MAKEFLAGS) .make/test
	@touch .make/push

.make/lint: .golangci.yml $(go_source) .make/solgen | .make
	golangci-lint run --fix
	@touch .make/lint

.make/fmt: .golangci.yml $(go_source) .make/solgen | .make
	golangci-lint run --disable-all -E gofmt --fix
	@touch .make/fmt

.make/test: $(go_source) .make/solgen .make/solidity | .make
	gotestsum --format short-verbose
	@touch .make/test

.make/solgen: solgen/gen.go .make/solidity | .make
	mkdir -p solgen/go/
	go run solgen/gen.go
	@touch .make/solgen

.make/solidity: solgen/src/*/*.sol .make/yarndeps | .make
	yarn --cwd solgen build
	@touch .make/solidity

.make/yarndeps: solgen/package.json solgen/yarn.lock | .make
	yarn --cwd solgen install
	@touch .make/yarndeps

.make:
	mkdir .make


# Makefile settings

always:              # use this to force other rules to always build
.DELETE_ON_ERROR:    # causes a failure to delete its target
