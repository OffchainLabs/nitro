//! Unit coverage for `BlockCtx`, `TxCtx`, and `ArbPrecompileCtx`.

use alloy_primitives::{address, b256, Address, B256, U256};
use arb_context::{ArbPrecompileCtx, BlockCtx, RecentWasms, TxCtx};
use std::sync::Arc;

// ── BlockCtx: L1/L2 block number and L2 block hash caches ───────────

#[test]
fn cached_l1_block_number_round_trips() {
    let block = BlockCtx::default();
    block.cache_l1_block_number(10, 1_000);
    block.cache_l1_block_number(11, 1_001);

    assert_eq!(block.cached_l1_block_number(10), Some(1_000));
    assert_eq!(block.cached_l1_block_number(11), Some(1_001));
    assert_eq!(block.cached_l1_block_number(12), None);
}

#[test]
fn cached_l1_block_number_retains_recent_window() {
    let block = BlockCtx::default();
    block.cache_l1_block_number(1, 100);
    // A height >100 triggers retention pruning of anything <(l2_block - 100).
    block.cache_l1_block_number(200, 2_000);
    assert!(block.cached_l1_block_number(1).is_none());
    assert_eq!(block.cached_l1_block_number(200), Some(2_000));
}

#[test]
fn cached_l2_block_hash_round_trips() {
    let block = BlockCtx::default();
    let hash = b256!("1111111111111111111111111111111111111111111111111111111111111111");
    block.cache_l2_block_hash(42, hash);

    assert_eq!(block.cached_l2_block_hash(42), Some(hash));
    assert_eq!(block.cached_l2_block_hash(43), None);
}

// ── BlockCtx: RecentWasms LRU ────────────────────────────────────────

#[test]
fn insert_recent_wasm_reports_first_insertion_as_new() {
    let block = BlockCtx::default();
    block.reset_recent_wasms(4);
    let h = b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");

    assert!(!block.insert_recent_wasm(h)); // not previously present
    assert!(block.insert_recent_wasm(h)); // re-insertion is a hit
}

#[test]
fn reset_recent_wasms_clears_entries_and_resizes() {
    let block = BlockCtx::default();
    block.reset_recent_wasms(8);
    for i in 0..6u8 {
        let mut bytes = [0u8; 32];
        bytes[0] = i;
        block.insert_recent_wasm(B256::from(bytes));
    }
    // Shrink to a smaller capacity; the LRU forgets everything.
    block.reset_recent_wasms(2);
    let h0 = b256!("0000000000000000000000000000000000000000000000000000000000000000");
    assert!(!block.insert_recent_wasm(h0));
}

#[test]
fn recent_wasms_evicts_oldest_when_capacity_exceeded() {
    let mut wasms = RecentWasms::new(2);
    let a = b256!("000000000000000000000000000000000000000000000000000000000000000a");
    let b = b256!("000000000000000000000000000000000000000000000000000000000000000b");
    let c = b256!("000000000000000000000000000000000000000000000000000000000000000c");

    assert!(!wasms.insert(a));
    assert!(!wasms.insert(b));
    // `c` evicts `a` (capacity 2).
    assert!(!wasms.insert(c));
    // `a` should be gone — re-inserting it is reported as new.
    assert!(!wasms.insert(a));
    // `b` is still cached (LRU touched it last when c was inserted? actually
    // insertion is append-after-remove; `b` came in before `c` so order is
    // [b, c]; inserting `a` evicts `b`. Verify `c` is still present.
    assert!(wasms.insert(c));
}

#[test]
fn recent_wasms_zero_capacity_disables_eviction_cap() {
    // A capacity of 0 means no upper bound enforcement in `insert`.
    let mut wasms = RecentWasms::new(0);
    for i in 0..10u8 {
        let mut bytes = [0u8; 32];
        bytes[31] = i;
        assert!(!wasms.insert(B256::from(bytes)));
    }
    // All ten entries are retained — re-inserting the first one is a hit.
    let mut bytes = [0u8; 32];
    bytes[31] = 0;
    assert!(wasms.insert(B256::from(bytes)));
}

// ── BlockCtx: gas backlog atomic ────────────────────────────────────

#[test]
fn current_gas_backlog_reads_and_writes() {
    let block = BlockCtx::default();
    assert_eq!(block.current_gas_backlog(), 0);
    block.set_current_gas_backlog(7_777);
    assert_eq!(block.current_gas_backlog(), 7_777);
}

// ── BlockCtx::new ───────────────────────────────────────────────────

#[test]
fn block_ctx_new_populates_fields() {
    let block = BlockCtx::new(60, 1_700_000_000, 19_000_000, 250_000, true);
    assert_eq!(block.arbos_version, 60);
    assert_eq!(block.block_timestamp, 1_700_000_000);
    assert_eq!(block.l1_block_number_for_evm, 19_000_000);
    assert_eq!(block.l2_block_number, 250_000);
    assert!(block.allow_debug_precompiles);
    assert_eq!(block.current_gas_backlog(), 0);
}

// ── ArbPrecompileCtx: caller stack ──────────────────────────────────

const A: Address = address!("aaaa000000000000000000000000000000000001");
const B: Address = address!("bbbb000000000000000000000000000000000002");
const C: Address = address!("cccc000000000000000000000000000000000003");

#[test]
fn caller_at_depth_indexes_from_one() {
    let ctx = ArbPrecompileCtx::default();
    ctx.push_caller(A);
    ctx.push_caller(B);
    ctx.push_caller(C);

    assert_eq!(ctx.caller_at_depth(0), None); // 0 is reserved
    assert_eq!(ctx.caller_at_depth(1), Some(A));
    assert_eq!(ctx.caller_at_depth(2), Some(B));
    assert_eq!(ctx.caller_at_depth(3), Some(C));
    assert_eq!(ctx.caller_at_depth(4), None); // out of bounds
}

#[test]
fn push_pop_caller_round_trips() {
    let ctx = ArbPrecompileCtx::default();
    for addr in [A, B, C] {
        ctx.push_caller(addr);
    }
    ctx.pop_caller();
    assert_eq!(ctx.caller_at_depth(3), None);
    assert_eq!(ctx.caller_at_depth(2), Some(B));
    ctx.pop_caller();
    ctx.pop_caller();
    assert_eq!(ctx.caller_at_depth(1), None);
}

#[test]
fn reset_caller_stack_clears_all_frames() {
    let ctx = ArbPrecompileCtx::default();
    ctx.push_caller(A);
    ctx.push_caller(B);
    ctx.reset_caller_stack();
    assert_eq!(ctx.caller_at_depth(1), None);
    assert_eq!(ctx.caller_at_depth(2), None);
}

#[test]
fn pop_caller_on_empty_stack_is_a_noop() {
    let ctx = ArbPrecompileCtx::default();
    ctx.pop_caller(); // must not panic
    assert_eq!(ctx.caller_at_depth(1), None);
}

// ── ArbPrecompileCtx: EVM depth ─────────────────────────────────────

#[test]
fn evm_depth_atomic_round_trips() {
    let ctx = ArbPrecompileCtx::default();
    assert_eq!(ctx.evm_depth(), 0);
    ctx.set_evm_depth(5);
    assert_eq!(ctx.evm_depth(), 5);
}

// ── ArbPrecompileCtx: Stylus program counters ───────────────────────

#[test]
fn push_stylus_program_signals_reentrant_call_on_second_entry() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(A)); // first entry: fresh
    assert!(ctx.push_stylus_program(A)); // second entry: reentrant
    assert_eq!(ctx.stylus_program_count(A), 2);
}

#[test]
fn push_distinct_stylus_programs_are_not_reentrant() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(A));
    assert!(!ctx.push_stylus_program(B));
    assert_eq!(ctx.stylus_program_count(A), 1);
    assert_eq!(ctx.stylus_program_count(B), 1);
}

#[test]
fn pop_stylus_program_decrements_and_removes_at_zero() {
    let ctx = ArbPrecompileCtx::default();
    ctx.push_stylus_program(A);
    ctx.push_stylus_program(A);
    assert_eq!(ctx.stylus_program_count(A), 2);
    ctx.pop_stylus_program(A);
    assert_eq!(ctx.stylus_program_count(A), 1);
    ctx.pop_stylus_program(A);
    assert_eq!(ctx.stylus_program_count(A), 0);
    // Popping below zero saturates without panicking.
    ctx.pop_stylus_program(A);
    assert_eq!(ctx.stylus_program_count(A), 0);
}

#[test]
fn pop_stylus_program_unknown_address_is_a_noop() {
    let ctx = ArbPrecompileCtx::default();
    ctx.pop_stylus_program(A);
    assert_eq!(ctx.stylus_program_count(A), 0);
}

// ── TxCtx: round-trip every setter through the lock ─────────────────

#[test]
fn tx_ctx_setters_round_trip_every_field() {
    let ctx = ArbPrecompileCtx::default();

    let sender = address!("1111111111111111111111111111111111111111");
    let redeemer = address!("2222222222222222222222222222222222222222");
    let activation_addr = address!("3333333333333333333333333333333333333333");
    let retryable = b256!("4444444444444444444444444444444444444444444444444444444444444444");
    let keepalive = b256!("5555555555555555555555555555555555555555555555555555555555555555");
    let activation_fee = U256::from(99_999u64);
    let call_value = U256::from(1_234_567u64);

    ctx.set_sender(sender);
    ctx.set_effective_gas_price(7_777);
    ctx.set_poster_fee(8_888);
    ctx.set_poster_balance_correction(9_999);
    ctx.set_retryable_id(retryable);
    ctx.set_redeemer(redeemer);
    ctx.set_tx_is_aliased(true);
    ctx.set_stylus_activation_addr(Some(activation_addr));
    ctx.set_stylus_keepalive_hash(Some(keepalive));
    ctx.set_stylus_activation_data_fee(activation_fee);
    ctx.set_stylus_call_value(call_value);

    let snap = ctx.tx_snapshot();
    assert_eq!(snap.sender, sender);
    assert_eq!(snap.effective_gas_price, 7_777);
    assert_eq!(snap.poster_fee, 8_888);
    assert_eq!(snap.poster_balance_correction, 9_999);
    assert_eq!(snap.retryable_id, retryable);
    assert_eq!(snap.redeemer, redeemer);
    assert!(snap.tx_is_aliased);
    assert_eq!(snap.stylus_activation_addr, Some(activation_addr));
    assert_eq!(snap.stylus_keepalive_hash, Some(keepalive));
    assert_eq!(snap.stylus_activation_data_fee, activation_fee);
    assert_eq!(snap.stylus_call_value, call_value);
    assert!(ctx.tx_is_aliased());
    assert_eq!(ctx.stylus_call_value(), call_value);
}

#[test]
fn take_stylus_activation_fields_consume_state() {
    let ctx = ArbPrecompileCtx::default();
    let addr = address!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    let hash = b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    let fee = U256::from(42u64);

    ctx.set_stylus_activation_addr(Some(addr));
    ctx.set_stylus_keepalive_hash(Some(hash));
    ctx.set_stylus_activation_data_fee(fee);

    assert_eq!(ctx.take_stylus_activation_addr(), Some(addr));
    assert_eq!(ctx.take_stylus_activation_addr(), None); // consumed
    assert_eq!(ctx.take_stylus_keepalive_hash(), Some(hash));
    assert_eq!(ctx.take_stylus_keepalive_hash(), None);
    assert_eq!(ctx.take_stylus_activation_data_fee(), fee);
    assert_eq!(ctx.take_stylus_activation_data_fee(), U256::ZERO);
}

#[test]
fn reset_tx_restores_every_field_to_default() {
    let ctx = ArbPrecompileCtx::default();
    ctx.set_sender(address!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"));
    ctx.set_effective_gas_price(1);
    ctx.set_poster_fee(2);
    ctx.set_poster_balance_correction(3);
    ctx.set_retryable_id(b256!(
        "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
    ));
    ctx.set_redeemer(address!("dddddddddddddddddddddddddddddddddddddddd"));
    ctx.set_tx_is_aliased(true);
    ctx.set_stylus_activation_addr(Some(address!("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")));
    ctx.set_stylus_keepalive_hash(Some(b256!(
        "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
    )));
    ctx.set_stylus_activation_data_fee(U256::from(99u64));
    ctx.set_stylus_call_value(U256::from(123u64));
    ctx.push_stylus_program(A);

    ctx.reset_tx();
    let snap = ctx.tx_snapshot();
    assert_eq!(snap.sender, Address::ZERO);
    assert_eq!(snap.effective_gas_price, 0);
    assert_eq!(snap.poster_fee, 0);
    assert_eq!(snap.poster_balance_correction, 0);
    assert_eq!(snap.retryable_id, B256::ZERO);
    assert_eq!(snap.redeemer, Address::ZERO);
    assert!(!snap.tx_is_aliased);
    assert_eq!(snap.stylus_activation_addr, None);
    assert_eq!(snap.stylus_keepalive_hash, None);
    assert_eq!(snap.stylus_activation_data_fee, U256::ZERO);
    assert_eq!(snap.stylus_call_value, U256::ZERO);
    assert!(snap.stylus_program_counts.is_empty());
    assert_eq!(snap.stylus_pages_open, 0);
    assert_eq!(snap.stylus_pages_ever, 0);
}

#[test]
fn tx_ctx_redeemer_word_left_pads_address_to_32_bytes() {
    let tx = TxCtx {
        redeemer: address!("1234567890abcdef1234567890abcdef12345678"),
        ..TxCtx::default()
    };
    // Address (20 bytes) is left-padded with 12 zero bytes into a 32-byte word.
    let mut expected_bytes = [0u8; 32];
    expected_bytes[12..].copy_from_slice(tx.redeemer.as_slice());
    let expected = U256::from_be_bytes(expected_bytes);
    assert_eq!(tx.redeemer_word(), expected);
}

#[test]
fn tx_ctx_redeemer_word_zero_address_is_zero() {
    let tx = TxCtx::default();
    assert_eq!(tx.redeemer_word(), U256::ZERO);
}

// ── ArbPrecompileCtx::with_block shares the same BlockCtx ───────────

#[test]
fn arb_precompile_ctx_with_block_shares_arc() {
    let block = Arc::new(BlockCtx::new(60, 1, 2, 3, false));
    let ctx_a = ArbPrecompileCtx::with_block(Arc::clone(&block));
    let ctx_b = ArbPrecompileCtx::with_block(Arc::clone(&block));

    // A write through ctx_a is observable through ctx_b.
    ctx_a.block.set_current_gas_backlog(4_242);
    assert_eq!(ctx_b.block.current_gas_backlog(), 4_242);
}

// ── Compile-time guarantees for cross-thread plumbing ───────────────

fn _assert_send_sync_static<T: Send + Sync + 'static>() {}

#[test]
fn context_types_are_send_sync_static() {
    // Compile-time assertion only: monomorphisation fails to link if any of
    // these types lose the bounds threaded through executor task spawning.
    _assert_send_sync_static::<ArbPrecompileCtx>();
    _assert_send_sync_static::<BlockCtx>();
    _assert_send_sync_static::<TxCtx>();
}
