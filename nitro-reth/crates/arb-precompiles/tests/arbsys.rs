//! Integration tests for ArbSys precompile (address 0x64).
//!
//! Each test mirrors a method of `precompiles/ArbSys.go` in Nitro and asserts
//! that our handler returns the same value, modifies the same state, and emits
//! the same events as Nitro's reference implementation.

mod common;

use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::create_arbsys_precompile;
use common::{calldata, decode_address, decode_u256, word_address, word_u256, PrecompileTest};

const ARBOS_V11: u64 = 11;
const ARBOS_V30: u64 = 30;

fn arbsys() -> alloy_evm::precompiles::DynPrecompile {
    create_arbsys_precompile()
}

#[test]
fn arb_block_number_returns_l2_block() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_number(98_765)
        .arbos_state()
        .call(&arbsys(), &calldata("arbBlockNumber()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(98_765));
}

#[test]
fn arb_chain_id_returns_configured_chain_id() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .chain_id(421_614)
        .arbos_state()
        .call(&arbsys(), &calldata("arbChainID()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(421_614));
}

#[test]
fn arbos_version_returns_55_plus_raw() {
    // Nitro's ArbOSVersion (precompiles/ArbSys.go:67) returns 55 + raw stored value.
    // The protocol stores the raw version (e.g. 30); the precompile reports 85.
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(&arbsys(), &calldata("arbOSVersion()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(55 + ARBOS_V30));
}

#[test]
fn get_storage_gas_available_returns_zero() {
    // Nitro precompiles/ArbSys.go:73 always returns 0.
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(&arbsys(), &calldata("getStorageGasAvailable()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn is_top_level_call_at_depth_one() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .evm_depth(1)
        .arbos_state()
        .call(&arbsys(), &calldata("isTopLevelCall()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn is_top_level_call_at_depth_three() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .evm_depth(3)
        .arbos_state()
        .call(&arbsys(), &calldata("isTopLevelCall()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn map_l1_sender_low_byte_carry() {
    // 0x0123...4567 + 0x1111...1111
    //   low bytes: 0x67 + 0x11 = 0x78 (no carry)
    //   high bytes: 0x01 + 0x11 = 0x12 (no carry into byte -1)
    //   middle bytes unchanged because the offset's middle is all zero
    let l1: Address = address!("0123456789abcdef0123456789abcdef01234567");
    let expected: Address = address!("1234456789abcdef0123456789abcdef01235678");
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbsys(),
            &calldata(
                "mapL1SenderContractAddressToL2Alias(address,address)",
                &[word_address(l1), word_address(Address::ZERO)],
            ),
        );
    assert_eq!(decode_address(run.output()), expected);
}

#[test]
fn map_l1_sender_with_carry_propagation() {
    // Per-byte addition would give 0x1111 ef 00... wrong.
    // True 160-bit integer addition gives 0x12 00 00... 1111 (carry from byte 1 into byte 0).
    let l1: Address = address!("00ef000000000000000000000000000000000000");
    let expected: Address = address!("1200000000000000000000000000000000001111");
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbsys(),
            &calldata(
                "mapL1SenderContractAddressToL2Alias(address,address)",
                &[word_address(l1), word_address(Address::ZERO)],
            ),
        );
    assert_eq!(decode_address(run.output()), expected);
}

#[test]
fn was_aliased_returns_false_when_tx_not_aliased() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .evm_depth(2)
        .tx_is_aliased(false)
        .arbos_state()
        .call(&arbsys(), &calldata("wasMyCallersAddressAliased()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn was_aliased_returns_true_when_top_level_aliased() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .evm_depth(2)
        .tx_is_aliased(true)
        .caller(address!("00000000000000000000000000000000000000aa"))
        .arbos_state()
        .call(&arbsys(), &calldata("wasMyCallersAddressAliased()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn arb_block_hash_returns_cached_hash_for_recent_block() {
    let target_hash =
        B256::from_slice(&[0x42; 32]);
    arb_precompiles::set_l2_block_hash(99, target_hash);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_number(100) // current = 100, requested = 99 (within 256 window, < current)
        .arbos_state()
        .call(
            &arbsys(),
            &calldata("arbBlockHash(uint256)", &[word_u256(U256::from(99))]),
        );
    let returned = B256::from(decode_u256(run.output()).to_be_bytes::<32>());
    assert_eq!(returned, target_hash);
}

#[test]
fn arb_block_hash_reverts_for_future_block_arbos11() {
    // requestedBlockNum >= currentNumber must revert with InvalidBlockNumberError on ArbOS >= 11.
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V11)
        .block_number(100)
        .arbos_state()
        .call(
            &arbsys(),
            &calldata("arbBlockHash(uint256)", &[word_u256(U256::from(100))]),
        );
    let out = run.assert_ok();
    assert_eq!(out.reverted, true, "expected reverted output");
    // Selector for InvalidBlockNumberError(uint256,uint256) is the first 4 bytes of revert data.
    assert_eq!(&out.bytes[..4], &[0xd5, 0xdc, 0x64, 0x2d]);
}

#[test]
fn arb_block_hash_reverts_for_too_old_block_arbos11() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V11)
        .block_number(1000)
        .arbos_state()
        .call(
            &arbsys(),
            &calldata("arbBlockHash(uint256)", &[word_u256(U256::from(500))]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
}
