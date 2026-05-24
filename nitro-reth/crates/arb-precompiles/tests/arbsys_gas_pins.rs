//! Per-selector gas pins for ArbSys (0x64).
//!
//! Locks the exact `PrecompileOutput::gas_used` for every selector. ArbSys
//! does not use the standard `init_precompile_gas` framework — each handler
//! builds its own accumulator from scratch, so per-selector pins are the
//! only safe guard against silent drift in the bespoke schedules.

mod common;

use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::create_arbsys_precompile;
use common::{calldata, word_address, word_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;
const ARBOS_V11: u64 = 11;

const SLOAD: u64 = 800;
const COPY: u64 = 3;

fn arbsys(
    ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>,
) -> alloy_evm::precompiles::DynPrecompile {
    create_arbsys_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

// ── pure-ish view functions: SLOAD + argsCost + resultCost ─────────────

#[test]
fn arb_block_number_v30_gas_pin() {
    let run = fixture(ARBOS_V30)
        .block_number(98_765)
        .call(arbsys, &calldata("arbBlockNumber()", &[]));
    // STORAGE_READ_COST(800) + argsCost(0 words) + resultCost(1 word) = 803
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn arb_chain_id_v30_gas_pin() {
    let run = fixture(ARBOS_V30)
        .chain_id(421_614)
        .call(arbsys, &calldata("arbChainID()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn arbos_version_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbsys, &calldata("arbOSVersion()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn get_storage_gas_available_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbsys, &calldata("getStorageGasAvailable()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn is_top_level_call_v30_gas_pin() {
    let run = fixture(ARBOS_V30)
        .evm_depth(1)
        .call(arbsys, &calldata("isTopLevelCall()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn was_my_callers_address_aliased_v30_gas_pin() {
    let run = fixture(ARBOS_V30)
        .evm_depth(2)
        .call(arbsys, &calldata("wasMyCallersAddressAliased()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn my_callers_address_without_aliasing_v30_gas_pin() {
    let run = fixture(ARBOS_V30)
        .evm_depth(1)
        .call(arbsys, &calldata("myCallersAddressWithoutAliasing()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn map_l1_sender_v30_gas_pin() {
    // Pure function: charges only argsCost + resultCost, no SLOAD.
    let l1: Address = address!("0123456789abcdef0123456789abcdef01234567");
    let run = fixture(ARBOS_V30).call(
        arbsys,
        &calldata(
            "mapL1SenderContractAddressToL2Alias(address,address)",
            &[word_address(l1), word_address(Address::ZERO)],
        ),
    );
    // argsCost(2 words) + resultCost(1 word) = 9
    assert_eq!(run.gas_used(), 2 * COPY + COPY);
}

// ── arbBlockHash: success returns 1 SLOAD+args+result; revert path
//    (ArbOS>=11) emits InvalidBlockNumber as a sol-error revert ──────

#[test]
fn arb_block_hash_recent_v30_gas_pin() {
    let target_hash = B256::from_slice(&[0x42; 32]);
    let run = fixture(ARBOS_V30)
        .block_number(100)
        .cache_l2_block_hash(99, target_hash)
        .call(
            arbsys,
            &calldata("arbBlockHash(uint256)", &[word_u256(U256::from(99))]),
        );
    // STORAGE_READ_COST(800) + argsCost(1 word) + resultCost(1 word) = 806
    assert_eq!(run.gas_used(), SLOAD + 2 * COPY);
}

#[test]
fn arb_block_hash_future_block_revert_arbos11_gas_pin() {
    // ArbOS >= 11 emits an `InvalidBlockNumber(uint256,uint256)` sol-error
    // revert. Encoded payload is 4 (selector) + 2 * 32 = 68 bytes → 3 words.
    let run = fixture(ARBOS_V11).block_number(100).call(
        arbsys,
        &calldata("arbBlockHash(uint256)", &[word_u256(U256::from(100))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // STORAGE_READ_COST(800) + argsCost(1 word) + resultCost(3 words) = 812
    assert_eq!(out.gas_used, SLOAD + COPY + 3 * COPY);
}

// ── send_merkle_tree_state: caller-zero, accumulator-size-zero path ──

#[test]
fn send_merkle_tree_state_empty_v30_gas_pin() {
    // Caller must be Address::ZERO. Accumulator is empty → only 1 size SLOAD
    // and no partial reads. Output is 4 head words (size + root + offset +
    // count), so resultCost = 4 * COPY.
    let run = fixture(ARBOS_V30)
        .caller(Address::ZERO)
        .call(arbsys, &calldata("sendMerkleTreeState()", &[]));
    // STORAGE_READ_COST(800 body size read) + argsCost(0) + resultCost(4 words) = 812
    assert_eq!(run.gas_used(), SLOAD + 4 * COPY);
}

// ── L2→L1 send paths. The full append-emit schedule includes keccak,
//    merkle partial writes, and per-event log gas — these pins lock the
//    *complete* observed value rather than re-deriving it. ────────────

#[test]
fn withdraw_eth_to_l1_v30_gas_pin() {
    let dest: Address = address!("000000000000000000000000000000000000bbbb");
    let run = fixture(ARBOS_V30)
        .caller(address!("00000000000000000000000000000000000000aa"))
        .block_number(1_000)
        .block_timestamp(1_700_000_000)
        .gas(2_000_000)
        .call(
            arbsys,
            &calldata("withdrawEth(address)", &[word_address(dest)]),
        );
    // Empty accumulator (0 merge events). Decomposition in arbsys.rs's
    // do_send_tx_to_l1:
    //   argsCost(1 word)                       = 3
    //   OpenArbosState SLOAD                   = 800
    //   pre-append size SLOAD                  = 800
    //   send_hash keccak(168 bytes = 6 words)  = 30 + 6*6 = 66
    //   per_merge_gas * 0                      = 0
    //   terminator_gas (n_events == num_partials_old = 0) = SSTORE_SET = 20_000
    //   size.set SSTORE                        = 20_000
    //   phantom post-append size SLOAD         = 800
    //   L2ToL1Tx LOG4+data (7 head words + 0 payload + 0 pad = 224 bytes):
    //     375 + 4*375 + 8*224                  = 3_667
    //   resultCost(1 word)                     = 3
    //   Total                                  = 46_139
    assert_eq!(run.gas_used(), 46_139);
}

#[test]
fn send_tx_to_l1_with_calldata_v30_gas_pin() {
    let dest: Address = address!("000000000000000000000000000000000000cccc");
    let payload = vec![0xab; 32];
    let mut buf = Vec::with_capacity(4 + 4 * 32);
    buf.extend_from_slice(&common::selector("sendTxToL1(address,bytes)"));
    buf.extend_from_slice(word_address(dest).as_slice());
    buf.extend_from_slice(word_u256(U256::from(64u64)).as_slice());
    buf.extend_from_slice(word_u256(U256::from(payload.len() as u64)).as_slice());
    buf.extend_from_slice(&payload);
    let run = fixture(ARBOS_V30)
        .caller(address!("00000000000000000000000000000000000000aa"))
        .block_number(1_000)
        .block_timestamp(1_700_000_000)
        .gas(2_000_000)
        .call(arbsys, &alloy_primitives::Bytes::from(buf));
    // Empty accumulator (0 merge events) + 32-byte calldata payload.
    // Decomposition matches do_send_tx_to_l1: argsCost(4 words) + OpenArbosState
    // + size SLOAD + keccak(200) + terminator SSTORE_SET + size SSTORE_SET
    // + phantom size SLOAD + L2ToL1Tx LOG4 over 7-head + 1-payload + 0-pad
    // (256 bytes) + resultCost.
    assert_eq!(run.gas_used(), 46_410);
}
