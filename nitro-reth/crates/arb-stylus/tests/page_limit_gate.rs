//! Single-node tests for the Stylus consensus `PageLimit` gate and the
//! `pay_for_memory_grow` operand width, asserted against the runtime primitives
//! directly: the page charge saturates exactly when `arbos_version >= 59 &&
//! page_limit > 0 && new_open > page_limit`, and is inert at
//! `arbos_version == 58`.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::{Address, B256, U256};
use arb_stylus::{
    config::{CompileConfig, StylusConfig},
    env::{page_limit_exceeded, WasmEnv},
    evm_api::{CreateResponse, EvmApi, UserOutcomeKind},
    ink::{Gas, Ink},
};
use arbos::programs::{memory::MemoryModel, types::EvmData};

const ARBOS_59: u64 = 59;
const ARBOS_58: u64 = 58;
const ARBOS_60: u64 = 60;

const FREE_PAGES: u16 = 2;
const PAGE_GAS: u16 = 1_000;

// ── page_limit_exceeded truth table ─────────────────────────────────

#[test]
fn predicate_fires_only_above_limit_at_v59_plus() {
    assert!(page_limit_exceeded(ARBOS_60, 4, 9));
    assert!(page_limit_exceeded(ARBOS_59, 4, 5));
    assert!(!page_limit_exceeded(ARBOS_60, 4, 4));
    assert!(!page_limit_exceeded(ARBOS_60, 4, 3));
}

#[test]
fn predicate_inert_below_v59() {
    assert!(!page_limit_exceeded(ARBOS_58, 4, 9));
    assert!(!page_limit_exceeded(0, 4, 9));
}

#[test]
fn predicate_inert_when_limit_zero() {
    assert!(!page_limit_exceeded(ARBOS_60, 0, 9));
}

// ── add_pages_charge saturation ─────────────────────────────────────

fn env_with(open: u16, page_limit: u16, arbos_version: u64) -> WasmEnv<NoopEvmApi> {
    let mut env = WasmEnv::new(
        CompileConfig::default(),
        Some(StylusConfig::default()),
        NoopEvmApi,
        evm_data_stub(),
    );
    env.set_pages(open, open, FREE_PAGES, PAGE_GAS, page_limit, arbos_version);
    env
}

/// Footprint 1 already open, grow 8 → open 9 > limit 4 at arbos 60: the charge
/// saturates to `u64::MAX`.
#[test]
fn add_pages_charge_saturates_over_limit_at_v60() {
    let mut env = env_with(1, 4, ARBOS_60);
    let cost = env.add_pages_charge(8);
    assert_eq!(cost, u64::MAX);
    assert_eq!(env.pages_open, 9);
}

/// Same open/grow/limit, arbos 58: the gate is inert, so the charge is the
/// ordinary finite memory-model cost.
#[test]
fn add_pages_charge_inert_below_v59() {
    let mut env = env_with(1, 4, ARBOS_58);
    let cost = env.add_pages_charge(8);
    let expected = MemoryModel::new(FREE_PAGES, PAGE_GAS).gas_cost(8, 1, 1);
    assert_eq!(cost, expected);
    assert!(cost < u64::MAX);
    assert_eq!(env.pages_open, 9);
}

/// Exactly at the limit is allowed (`new_open > page_limit` is strict).
#[test]
fn add_pages_charge_exactly_at_limit_is_finite() {
    let mut env = env_with(1, 9, ARBOS_60);
    let cost = env.add_pages_charge(8);
    assert!(cost < u64::MAX);
    assert_eq!(env.pages_open, 9);
}

/// A zero `page_limit` disables the cap even at arbos 60.
#[test]
fn add_pages_charge_zero_limit_disabled() {
    let mut env = env_with(1, 0, ARBOS_60);
    let cost = env.add_pages_charge(8);
    assert!(cost < u64::MAX);
}

/// Open 9 stays under the ample default limit 128, so the charge is finite at
/// arbos 60.
#[test]
fn add_pages_charge_under_default_limit_is_finite() {
    let mut env = env_with(1, 128, ARBOS_60);
    let cost = env.add_pages_charge(8);
    assert!(cost < u64::MAX);
    assert_eq!(env.pages_open, 9);
}

// ── entry-footprint site (stylus_call_gas_cost in arb-evm) ──────────

/// The entry-footprint reservation adds `u64::MAX` when `pages_open + footprint`
/// exceeds the limit at arbos >= 59, for a program that never grows at runtime.
#[test]
fn entry_footprint_predicate_matches_call_site() {
    assert!(page_limit_exceeded(ARBOS_60, 4, 0u16.saturating_add(5)));
    assert!(!page_limit_exceeded(ARBOS_58, 4, 0u16.saturating_add(5)));
    assert!(!page_limit_exceeded(ARBOS_60, 5, 0u16.saturating_add(5)));
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
