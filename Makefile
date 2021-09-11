inputs=$(wildcard prover/test-cases/*.wat)
rust_inputs=$(wildcard prover/test-cases/rust/*.rs)
outputs=$(patsubst prover/test-cases/%.wat,rollup/test/proofs/%.json, $(inputs)) $(patsubst prover/test-cases/rust/%.rs,rollup/test/proofs/rust-%.json, $(rust_inputs))
wasms=$(patsubst %.wat,%.wasm, $(inputs)) prover/test-cases/rust/basics.wasm

all: $(wasms) $(outputs)
	@printf "\e[38;5;161;1mdone building %s\e[0;0m\n" $$(expr $$(echo $? | wc -w) - 1)

clean:
	rm -f prover/test-cases/**/*.wasm
	rm -f rollup/test/proofs/*.json

prover/test-cases/rust/%.wasm: prover/test-cases/rust/%.rs
	rustc +nightly $< --target wasm32-unknown-unknown -o $@

prover/test-cases/%.wasm: prover/test-cases/%.wat
	wat2wasm $< -o $@

rollup/test/proofs/%.json: prover/test-cases/%.wasm prover/src/**
	cargo run -p prover -- $< -o $@

rollup/test/proofs/rust-%.json: prover/test-cases/rust/%.wasm prover/src/**
	cargo run -p prover -- $< -o $@

.DELETE_ON_ERROR: # causes a failure to delete its target
