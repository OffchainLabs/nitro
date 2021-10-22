inputs=$(wildcard prover/test-cases/*.wat)
rust_bin_sources=$(wildcard prover/test-cases/rust/src/bin/*.rs)
generated_arbitrator_header=prover/generated-inc/arbitrator.h
outputs=$(patsubst prover/test-cases/%.wat,rollup/test/proofs/%.json, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,rollup/test/proofs/rust-%.json, $(rust_bin_sources)) rollup/test/proofs/go.json $(generated_arbitrator_header)
wasms=$(patsubst %.wat,%.wasm, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm, $(rust_bin_sources)) prover/test-cases/go/main

WASI_SYSROOT?=/opt/wasi-sdk/wasi-sysroot

all: $(wasms) $(outputs)
	@printf "\e[38;5;161;1mdone building %s\e[0;0m\n" $$(expr $$(echo $? | wc -w) - 1)

clean:
	rm -rf prover/test-cases/rust/target
	rm -f prover/test-cases/*.wasm
	rm -f prover/test-cases/go/main
	rm -rf `dirname prover/generated-inc $(generated_arbitrator_header)`
	rm -f rollup/test/proofs/*.json
	rm -rf wasm-libraries/target
	rm -f wasm-libraries/soft-float/soft-float.wasm
	rm -f wasm-libraries/soft-float/*.o
	rm -f wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.o
	rm -f wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/*.a

prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm: prover/test-cases/rust/src/bin/%.rs prover/test-cases/rust/src/lib.rs
	cd prover/test-cases/rust && cargo build --target wasm32-wasi --bin $(patsubst prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm,%, $@)

prover/test-cases/go/main: prover/test-cases/go/main.go prover/test-cases/go/go.mod prover/test-cases/go/go.sum
	cd prover/test-cases/go && GOOS=js GOARCH=wasm go build main.go

$(generated_arbitrator_header):
# TODO only gen if needed
	cbindgen --config cbindgen.toml --crate prover --output $(generated_arbitrator_header)

wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm: wasm-libraries/wasi-stub/src/**
	cd wasm-libraries && cargo build --target wasm32-unknown-unknown --package wasi-stub

wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/softfloat.a: \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/Makefile \
		wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang/platform.h \
		wasm-libraries/soft-float/SoftFloat-3e/source/*.c \
		wasm-libraries/soft-float/SoftFloat-3e/source/include/*.h \
		wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.c \
		wasm-libraries/soft-float/SoftFloat-3e/source/8086/*.h
	cd wasm-libraries/soft-float/SoftFloat-3e/build/Wasm-Clang && make $(MAKEFLAGS)

wasm-libraries/soft-float/bindings%.o: wasm-libraries/soft-float/bindings%.c
	clang $< --sysroot $(WASI_SYSROOT) -I wasm-libraries/soft-float/SoftFloat-3e/source/include -target wasm32-wasi -Wconversion -c -o $@

wasm-libraries/soft-float/soft-float.wasm: \
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

wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm: wasm-libraries/go-stub/src/**
	cd wasm-libraries && cargo build --target wasm32-wasi --package go-stub

prover/test-cases/%.wasm: prover/test-cases/%.wat
	wat2wasm $< -o $@

rollup/test/proofs/%.json: prover/test-cases/%.wasm prover/src/**
	cargo run -p prover -- $< -o $@ --always-merkleize

rollup/test/proofs/float%.json: prover/test-cases/float%.wasm wasm-libraries/soft-float/soft-float.wasm prover/src/**
	cargo run --release -p prover -- $< -l wasm-libraries/soft-float/soft-float.wasm -o $@ -b --always-merkleize

rollup/test/proofs/rust-%.json: \
		prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm \
		wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm \
		wasm-libraries/soft-float/soft-float.wasm prover/src/**
	cargo run --release -p prover -- $< -l wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm -l wasm-libraries/soft-float/soft-float.wasm -o $@ -b --always-merkleize

rollup/test/proofs/go.json: \
		prover/test-cases/go/main \
		wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm \
		wasm-libraries/soft-float/soft-float.wasm prover/src/** \
		wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm
	cargo run --release -p prover -- $< -l wasm-libraries/target/wasm32-unknown-unknown/debug/wasi_stub.wasm -l wasm-libraries/soft-float/soft-float.wasm -l wasm-libraries/target/wasm32-wasi/debug/go_stub.wasm -o $@ -i 5000000

.DELETE_ON_ERROR: # causes a failure to delete its target
.PHONY: all clean
