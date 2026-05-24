//! Page accounting and reentrancy tests for the Stylus runtime.
//!
//! After the migration off thread-local globals, page tracking lives on
//! `WasmEnv` (per-instance, threaded across sub-calls explicitly) and the
//! reentrancy counter lives on `TxCtx`. These tests cover both surfaces.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::{address, Address, B256, U256};
use arb_context::ArbPrecompileCtx;
use arb_stylus::{
    config::{CompileConfig, StylusConfig},
    env::WasmEnv,
    evm_api::{CreateResponse, EvmApi, UserOutcomeKind},
    ink::{Gas, Ink},
};
use arbos::programs::{memory::MemoryModel, types::EvmData};

// ── MemoryModel: shared between WasmEnv and the precompile path ─────

#[test]
fn memory_model_returns_zero_below_free_pages() {
    let model = MemoryModel::new(2, 1_000);
    assert_eq!(model.gas_cost(2, 0, 0), 0);
}

#[test]
fn memory_model_charges_linear_plus_exponential_beyond_free() {
    let model = MemoryModel::new(2, 1_000);
    // open=0, ever=0, allocate 4 pages: 2 free + 2 paid linear + exp(4)-exp(0).
    let cost_first = model.gas_cost(4, 0, 0);
    assert!(cost_first > 0);
    // Re-allocating without freeing piles on more cost.
    let cost_second = model.gas_cost(2, 4, 4);
    assert!(cost_second > 0);
}

#[test]
fn memory_model_reuses_ever_high_water_mark_when_reopening() {
    let model = MemoryModel::new(0, 100);
    let full = model.gas_cost(10, 0, 0);
    // Closing and reopening the same pages skips the exponential delta because
    // `ever` already covers them; only the linear portion is charged.
    let reopen = model.gas_cost(10, 0, 10);
    assert!(reopen < full);
}

// ── WasmEnv page tracking ───────────────────────────────────────────

fn fresh_env() -> WasmEnv<NoopEvmApi> {
    WasmEnv::new(
        CompileConfig::default(),
        Some(StylusConfig::default()),
        NoopEvmApi,
        evm_data_stub(),
    )
}

#[test]
fn set_pages_seeds_open_and_ever_for_subcall() {
    let mut env = fresh_env();
    env.set_pages(3, 7, 2, 1_000);
    assert_eq!(env.pages_open, 3);
    assert_eq!(env.pages_ever, 7);
    assert_eq!(env.free_pages, 2);
    assert_eq!(env.page_gas, 1_000);
}

#[test]
fn add_pages_charge_advances_open_and_ever_for_fresh_env() {
    let mut env = fresh_env();
    env.set_pages(0, 0, 0, 100);
    env.add_pages_charge(5);
    assert_eq!(env.pages_open, 5);
    assert_eq!(env.pages_ever, 5);
}

#[test]
fn add_pages_charge_accumulates_open() {
    let mut env = fresh_env();
    env.set_pages(0, 0, 0, 100);
    env.add_pages_charge(5);
    env.add_pages_charge(3);
    assert_eq!(env.pages_open, 8);
    assert_eq!(env.pages_ever, 8);
}

#[test]
fn add_pages_charge_saturates_on_overflow() {
    let mut env = fresh_env();
    env.set_pages(u16::MAX - 5, u16::MAX - 5, 0, 0);
    env.add_pages_charge(100);
    assert_eq!(env.pages_open, u16::MAX);
    assert_eq!(env.pages_ever, u16::MAX);
}

#[test]
fn pages_ever_is_high_water_mark_after_freeing() {
    let mut env = fresh_env();
    env.set_pages(0, 0, 0, 100);
    env.add_pages_charge(10);
    // Simulate a sub-call freeing memory by writing the lower open count back.
    env.pages_open = 2;
    assert_eq!(env.pages_ever, 10);
    // Re-allocating beyond the previous high-water mark advances ever.
    env.add_pages_charge(20);
    assert_eq!(env.pages_open, 22);
    assert_eq!(env.pages_ever, 22);
}

#[test]
fn add_pages_charge_below_free_pages_is_free() {
    let mut env = fresh_env();
    env.set_pages(0, 0, 4, 500);
    let cost = env.add_pages_charge(3); // still within free window
    assert_eq!(cost, 0);
    assert_eq!(env.pages_open, 3);
    assert_eq!(env.pages_ever, 3);
}

#[test]
fn add_pages_charge_matches_memory_model_for_paid_pages() {
    let mut env = fresh_env();
    env.set_pages(0, 0, 2, 1_000);
    let cost = env.add_pages_charge(5);
    let expected = MemoryModel::new(2, 1_000).gas_cost(5, 0, 0);
    assert_eq!(cost, expected);
}

// ── Reentrancy counter (now on TxCtx via ArbPrecompileCtx) ──────────

const PROG_A: Address = address!("aaaa000000000000000000000000000000000000");
const PROG_B: Address = address!("bbbb000000000000000000000000000000000000");

#[test]
fn push_stylus_program_signals_reentrancy_on_second_entry() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(PROG_A));
    assert!(ctx.push_stylus_program(PROG_A));
    assert!(ctx.push_stylus_program(PROG_A));
}

#[test]
fn push_distinct_addresses_are_not_reentrant() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(PROG_A));
    assert!(!ctx.push_stylus_program(PROG_B));
}

#[test]
fn pop_stylus_program_decrements_then_removes_at_zero() {
    let ctx = ArbPrecompileCtx::default();
    ctx.push_stylus_program(PROG_A);
    ctx.push_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 2);
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 1);
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 0);
    // Extra pops saturate without panicking.
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 0);
}

// ── Test fixtures ───────────────────────────────────────────────────

fn evm_data_stub() -> EvmData {
    EvmData {
        arbos_version: 0,
        block_basefee: B256::ZERO,
        chain_id: 0,
        block_coinbase: Address::ZERO,
        block_gas_limit: 0,
        block_number: 0,
        block_timestamp: 0,
        contract_address: Address::ZERO,
        module_hash: B256::ZERO,
        msg_sender: Address::ZERO,
        msg_value: B256::ZERO,
        tx_gas_price: B256::ZERO,
        tx_origin: Address::ZERO,
        reentrant: 0,
        cached: false,
        tracing: false,
    }
}

/// Minimal `EvmApi` impl whose methods are unreachable in these tests —
/// page accounting paths never call into the bridge.
#[derive(Debug)]
struct NoopEvmApi;

impl EvmApi for NoopEvmApi {
    fn get_bytes32(&mut self, _key: B256, _gas: Gas) -> eyre::Result<(B256, Gas)> {
        unreachable!("page accounting must not touch the EVM bridge")
    }
    fn cache_bytes32(&mut self, _key: B256, _value: B256) -> eyre::Result<Gas> {
        unreachable!()
    }
    fn flush_storage_cache(
        &mut self,
        _clear: bool,
        _gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)> {
        unreachable!()
    }
    fn get_transient_bytes32(&mut self, _key: B256) -> eyre::Result<B256> {
        unreachable!()
    }
    fn set_transient_bytes32(&mut self, _key: B256, _value: B256) -> eyre::Result<UserOutcomeKind> {
        unreachable!()
    }
    fn contract_call(
        &mut self,
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
        _value: U256,
        _pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        unreachable!()
    }
    fn delegate_call(
        &mut self,
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
        _pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        unreachable!()
    }
    fn static_call(
        &mut self,
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
        _pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        unreachable!()
    }
    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: U256,
        _gas: Gas,
        _pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        unreachable!()
    }
    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: U256,
        _salt: B256,
        _gas: Gas,
        _pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        unreachable!()
    }
    fn get_return_data(&self) -> Vec<u8> {
        vec![]
    }
    fn emit_log(&mut self, _data: Vec<u8>, _topics: u32) -> eyre::Result<()> {
        unreachable!()
    }
    fn account_balance(&mut self, _address: Address) -> eyre::Result<(U256, Gas)> {
        unreachable!()
    }
    fn account_code(
        &mut self,
        _arbos_version: u64,
        _address: Address,
        _gas_left: Gas,
    ) -> eyre::Result<(Vec<u8>, Gas)> {
        unreachable!()
    }
    fn account_codehash(&mut self, _address: Address) -> eyre::Result<(B256, Gas)> {
        unreachable!()
    }
    fn capture_hostio(
        &mut self,
        _name: &str,
        _args: &[u8],
        _outs: &[u8],
        _start_ink: Ink,
        _end_ink: Ink,
    ) {
    }
}
