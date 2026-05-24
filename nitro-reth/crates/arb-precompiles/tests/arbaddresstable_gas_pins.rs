//! Per-selector gas pins for ArbAddressTable (0x66).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbaddresstable_precompile;
use common::{calldata, word_address, word_u256, PrecompileTest};

const SLOAD: u64 = 800;
const SSTORE_SET: u64 = 20_000;
const COPY: u64 = 3;

fn arbaddresstable() -> DynPrecompile {
    create_arbaddresstable_precompile()
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(30).arbos_state()
}

#[test]
fn size_v30_gas_pin() {
    // 2 * SLOAD + COPY = 1603.
    let run = fixture().call(&arbaddresstable(), &calldata("size()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn address_exists_v30_gas_pin() {
    // 2 * SLOAD + 2 * COPY = 1606.
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().call(
        &arbaddresstable(),
        &calldata("addressExists(address)", &[word_address(addr)]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn lookup_unknown_reverts_v30_gas_pin() {
    // Unknown → empty_revert with the boilerplate gas accumulated.
    // init_precompile_gas charged 800 + 1*3 = 803 before the body ran.
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().gas(50_000).call(
        &arbaddresstable(),
        &calldata("lookup(address)", &[word_address(addr)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, SLOAD + COPY);
}

#[test]
fn lookup_index_unknown_reverts_v30_gas_pin() {
    // Unknown index → empty_revert with init_precompile_gas accumulated.
    let run = fixture().gas(50_000).call(
        &arbaddresstable(),
        &calldata("lookupIndex(uint256)", &[word_u256(U256::from(0u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, SLOAD + COPY);
}

#[test]
fn register_new_address_v30_gas_pin() {
    // 803 boilerplate + (2*SLOAD + 3*SSTORE_SET + COPY) = 803 + 61_603 = 62_406.
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().call(
        &arbaddresstable(),
        &calldata("register(address)", &[word_address(addr)]),
    );
    assert_eq!(
        run.gas_used(),
        SLOAD + COPY + 2 * SLOAD + 3 * SSTORE_SET + COPY,
    );
}

#[test]
fn compress_unregistered_v30_gas_pin() {
    // Unregistered addr → raw 21-byte RLP: [0x94, addr]. Output buffer:
    // 32 (offset) + 32 (length) + 21 (data) + 11 (pad) = 96 bytes = 3 words.
    // body charges 2*SLOAD + COPY + 3*COPY = 1612.
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().call(
        &arbaddresstable(),
        &calldata("compress(address)", &[word_address(addr)]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY + 3 * COPY);
}

#[test]
fn decompress_raw_address_v30_gas_pin() {
    // First-byte > 0x80 = raw 21-byte address path: 1 body SLOAD + (args+2) COPY.
    // Calldata: selector(4) + offset(32) + length(32) + buf(32) +
    //           [21 byte raw], padded to 32 → 5 head words.
    let mut buf = Vec::with_capacity(4 + 5 * 32);
    buf.extend_from_slice(&common::selector("decompress(bytes,uint256)"));
    buf.extend_from_slice(word_u256(U256::from(64u64)).as_slice()); // offset to bytes
    buf.extend_from_slice(word_u256(U256::ZERO).as_slice()); // offset arg
    buf.extend_from_slice(word_u256(U256::from(21u64)).as_slice()); // bytes length
    let mut raw = vec![0u8; 32];
    raw[0] = 0x94; // raw 21-byte address marker
    for (i, b) in raw[1..21].iter_mut().enumerate() {
        *b = (i + 1) as u8;
    }
    buf.extend_from_slice(&raw);
    let run = fixture().call(&arbaddresstable(), &alloy_primitives::Bytes::from(buf));
    // args=4 words (offset, offset_arg, length, 32-byte raw padded buf) → 128 bytes.
    // body_sloads = 1 (raw 21-byte). resultCost = (4+2)*3 = 18. Plus 1*SLOAD = 800.
    assert_eq!(run.gas_used(), SLOAD + (4 + 2) * COPY);
}
