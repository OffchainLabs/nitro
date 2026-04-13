mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbaddresstable_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ADDRESS_TABLE_SUBSPACE,
        ARBOS_STATE_ADDRESS, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, word_address, PrecompileTest};

fn arbaddresstable() -> DynPrecompile {
    create_arbaddresstable_precompile()
}

fn table_key() -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE)
}

fn size_slot() -> U256 {
    map_slot(table_key().as_slice(), 0)
}

fn by_address_slot(addr: Address) -> U256 {
    let by_address_key = derive_subspace_key(table_key().as_slice(), &[]);
    map_slot_b256(
        by_address_key.as_slice(),
        &B256::left_padding_from(addr.as_slice()),
    )
}

#[test]
fn size_returns_zero_for_empty_table() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&arbaddresstable(), &calldata("size()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn size_returns_stored_value() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, size_slot(), U256::from(42))
        .call(&arbaddresstable(), &calldata("size()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(42));
}

#[test]
fn address_exists_returns_false_for_unregistered() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata("addressExists(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn address_exists_returns_true_for_registered() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, by_address_slot(addr), U256::from(1))
        .call(
            &arbaddresstable(),
            &calldata("addressExists(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

// ── Nitro TestArbAddressTableInit / TestAddressTable1 ports ──────────

/// Decode the dynamic `bytes` ABI return into a Vec<u8>.
fn decode_dynamic_bytes(out: &alloy_primitives::Bytes) -> Vec<u8> {
    let len = U256::from_be_slice(&out[32..64]).to::<usize>();
    out[64..64 + len].to_vec()
}

#[test]
fn lookup_unregistered_address_reverts() {
    // gas_check converts PrecompileError::Other to a reverted output at ArbOS >= 11.
    let addr: Address = address!("00000000000000000000000000000000000000bb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata("lookup(address)", &[word_address(addr)]),
        );
    let out = run.assert_ok();
    assert!(out.reverted, "lookup of unregistered must revert");
}

#[test]
fn lookup_index_zero_in_empty_table_reverts() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata(
                "lookupIndex(uint256)",
                &[B256::from(U256::ZERO.to_be_bytes::<32>())],
            ),
        );
    let out = run.assert_ok();
    assert!(out.reverted, "lookupIndex into empty table must revert");
}

#[test]
fn register_returns_zero_for_first_address_and_increments_size() {
    // Mirrors Nitro TestAddressTable1: registering the first address yields
    // slot index 0 and increments size to 1.
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata("register(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, size_slot()),
        U256::from(1u64)
    );
}

#[test]
fn compress_unregistered_returns_21_byte_raw_format() {
    // Mirrors Nitro TestAddressTableCompressNotInTable: unknown addresses
    // round-trip via the 21-byte RLP raw format.
    let addr: Address = address!("0123456789abcdef0123456789abcdef01234567");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata("compress(address)", &[word_address(addr)]),
        );
    let body = decode_dynamic_bytes(run.output());
    assert_eq!(body.len(), 21, "unknown addr compresses to 21 bytes");
    // The first byte is the RLP "20-byte string" tag (0x80 + 20 = 0x94).
    assert_eq!(body[0], 0x94);
    assert_eq!(&body[1..], addr.as_slice());
}

#[test]
fn compress_registered_returns_short_format() {
    // Mirrors Nitro TestAddressTableCompressInTable: known addresses should
    // compress to <= 9 bytes (1-byte RLP int prefix + up to 8-byte index).
    // We seed the by-address slot directly so we don't need to call register.
    let addr: Address = address!("0123456789abcdef0123456789abcdef01234567");
    let test = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        // Slot value = 1-based index → index 0 in Compress's view.
        .storage(ARBOS_STATE_ADDRESS, by_address_slot(addr), U256::from(1));
    let run = test.call(
        &arbaddresstable(),
        &calldata("compress(address)", &[word_address(addr)]),
    );
    let body = decode_dynamic_bytes(run.output());
    assert!(
        body.len() <= 9,
        "registered addr should compress to <= 9 bytes, got {}",
        body.len()
    );
}
