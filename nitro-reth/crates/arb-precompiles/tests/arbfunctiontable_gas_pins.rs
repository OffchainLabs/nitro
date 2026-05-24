//! Per-selector gas pins for ArbFunctionTable (0x68).
//!
//! All three selectors are degenerate (the table is empty / no-op) but each
//! still goes through the precompile framework's argsCost + OpenArbosState
//! accounting, plus a per-selector body cost. These pins lock in that
//! framework overhead.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{Address, U256};
use arb_precompiles::create_arbfunctiontable_precompile;
use common::{calldata, word_address, word_u256, PrecompileTest};

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

fn arbfunctiontable() -> DynPrecompile {
    create_arbfunctiontable_precompile()
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(30).arbos_state()
}

#[test]
fn upload_v30_gas_pin() {
    // upload(bytes) — no-op. Cost = OpenArbosState(800) + argsCost.
    // Calldata: selector (4 bytes) + dynamic-bytes head (offset + length) +
    // data tail. We pass an empty `bytes` arg: offset=32, length=0 → 2 words
    // of args after the selector → argsCost = 2 * COPY_GAS.
    let mut buf = Vec::with_capacity(4 + 64);
    buf.extend_from_slice(&common::selector("upload(bytes)"));
    buf.extend_from_slice(word_u256(U256::from(32u64)).as_slice());
    buf.extend_from_slice(word_u256(U256::ZERO).as_slice());
    let run = fixture().call(&arbfunctiontable(), &alloy_primitives::Bytes::from(buf));
    // OpenArbosState(800) + argsCost(2 * 3) = 806
    assert_eq!(run.gas_used(), SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn size_v30_gas_pin() {
    // size(address) → uint. Cost = OpenArbosState + 1 arg word + 1 result word.
    let run = fixture().call(
        &arbfunctiontable(),
        &calldata("size(address)", &[word_address(Address::ZERO)]),
    );
    // 800 + 3 + 3 = 806
    assert_eq!(run.gas_used(), SLOAD_GAS + COPY_GAS + COPY_GAS);
}

#[test]
fn get_reverts_burning_accumulated_gas_v30_gas_pin() {
    // get(address,uint) unconditionally reverts; gas_check returns the
    // accumulated OpenArbosState + argsCost on the revert path.
    let run = fixture().call(
        &arbfunctiontable(),
        &calldata(
            "get(address,uint256)",
            &[word_address(Address::ZERO), word_u256(U256::ZERO)],
        ),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // OpenArbosState(800) + argsCost(2 * 3) = 806
    assert_eq!(out.gas_used, SLOAD_GAS + 2 * COPY_GAS);
}
