inputs=$(wildcard prover/test-cases/*.wat)
rust_bin_sources=$(wildcard prover/test-cases/rust/src/bin/*.rs)
outputs=$(patsubst prover/test-cases/%.wat,rollup/test/proofs/%.json, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,rollup/test/proofs/rust-%.json, $(rust_bin_sources))
wasms=$(patsubst %.wat,%.wasm, $(inputs)) $(patsubst prover/test-cases/rust/src/bin/%.rs,prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm, $(rust_bin_sources))

all: $(wasms) $(outputs)
	@printf "\e[38;5;161;1mdone building %s\e[0;0m\n" $$(expr $$(echo $? | wc -w) - 1)

clean:
	rm -rf prover/test-cases/rust/target
	rm -f prover/test-cases/**/*.wasm
	rm -f rollup/test/proofs/*.json

prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm: prover/test-cases/rust/src/bin/%.rs prover/test-cases/rust/src/lib.rs
	cd prover/test-cases/rust && cargo build --target wasm32-wasi --bin $(patsubst prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm,%, $@)

prover/test-cases/%.wasm: prover/test-cases/%.wat
	wat2wasm $< -o $@

rollup/test/proofs/%.json: prover/test-cases/%.wasm prover/src/**
	cargo run -p prover -- $< -o $@

rollup/test/proofs/rust-%.json: prover/test-cases/rust/target/wasm32-wasi/debug/%.wasm prover/src/**
	cargo run --release -p prover -- $< -o $@ -b

.DELETE_ON_ERROR: # causes a failure to delete its target
