//! End-to-end test for the `pay_for_memory_grow` operand width.
//!
//! The host import param is `u32`. These tests instantiate a real WASM module
//! against the actual `host::pay_for_memory_grow` import (registered by
//! `NativeInstance::from_bytes_with_pages`) and call it with an `i32` operand of
//! 65536. At ArbOS >= 59 the operand exceeds `u16::MAX`, so `buy_gas(u64::MAX)`
//! exhausts the meter and traps. At ArbOS 58 the guard is inert, the host
//! truncates 65536 -> 0, and the cheap base-ink path returns success. The
//! boundary operand 65535 (== u16::MAX) succeeds at every version because the
//! guard is strict (`> u16::MAX`).

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::{Address, B256, U256};
use arb_stylus::{
    config::{CompileConfig, StylusConfig},
    evm_api::{CreateResponse, EvmApi, UserOutcomeKind},
    ink::{Gas, Ink},
    meter::{MachineMeter, MeteredMachine, STYLUS_INK_LEFT, STYLUS_INK_STATUS, STYLUS_STACK_LEFT},
    native::NativeInstance,
};
use arbos::programs::types::EvmData;
use wasmer::{TypedFunction, Value};

const ARBOS_60: u64 = 60;
const ARBOS_58: u64 = 58;

const PAGE_GAS: u16 = 1_000;
// Page limit disabled so the open-page gate (and the memory-model exponent
// saturation past 128 pages) cannot interfere: only the width guard, which runs
// before any page math, decides the outcome here.
const PAGE_LIMIT: u16 = 0;
const SEED_INK: i64 = i64::MAX;

/// Direct-call program: forwards a constant page operand to `pay_for_memory_grow`
/// and returns success. No calldata read keeps the metering deterministic.
fn wat_grow(operand: u32) -> String {
    format!(
        r#"
(module
    (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $pay_for_memory_grow (i32.const {operand}))
        i32.const 0
    )
)
"#
    )
}

/// Outcome of invoking the entrypoint once.
#[derive(Debug, PartialEq, Eq)]
enum Outcome {
    /// Host trapped and the meter was exhausted (the OOG path).
    OutOfInk,
    /// Entrypoint returned `status` with ink still available.
    Returned(u32),
}

/// Compile `wat`, register the real host imports, seed the meter, and call the
/// entrypoint once at the given ArbOS version.
fn run(operand: u32, arbos_version: u64) -> Outcome {
    // Version 1 installs the metering middleware so the ink globals exist and
    // the host charges land on the same path the runtime uses.
    let compile = CompileConfig::version(1, true).expect("compile config v1");
    let config = StylusConfig::default();
    let wasm = wat::parse_bytes(wat_grow(operand).as_bytes())
        .expect("wat compiles")
        .into_owned();

    let mut native = NativeInstance::from_bytes_with_pages(
        &wasm,
        NoopEvmApi,
        evm_data_stub(arbos_version),
        &compile,
        config,
        0,
        0,
        0,
        PAGE_GAS,
        PAGE_LIMIT,
        arbos_version,
    )
    .expect("instantiate");

    seed_meter(&mut native);

    let entrypoint = entrypoint_of(&native);
    match entrypoint.call(&mut native.store, 0) {
        Ok(status) => Outcome::Returned(status),
        Err(_) => {
            assert_eq!(
                ink_status(&mut native),
                1,
                "trap must be an out-of-ink (status 1), not another fault"
            );
            Outcome::OutOfInk
        }
    }
}

/// Resolve the typed `user_entrypoint` export.
fn entrypoint_of(native: &NativeInstance<NoopEvmApi>) -> TypedFunction<u32, u32> {
    native
        .instance
        .exports
        .get_typed_function(&native.store, "user_entrypoint")
        .expect("entrypoint export")
}

/// Seed both the WASM ink globals (read by the host's `buy_ink`) and `MeterData`.
fn seed_meter(native: &mut NativeInstance<NoopEvmApi>) {
    native
        .set_global(STYLUS_INK_LEFT, Value::I64(SEED_INK))
        .expect("seed ink global");
    native
        .set_global(STYLUS_INK_STATUS, Value::I32(0))
        .expect("seed ink status global");
    native
        .set_global(STYLUS_STACK_LEFT, Value::I32(i32::MAX))
        .expect("seed stack global");
    native.set_meter(MachineMeter::Ready(Ink(SEED_INK as u64)));
}

/// Read the WASM ink-left global directly (independent of `MeterData`).
fn ink_global(native: &mut NativeInstance<NoopEvmApi>) -> u64 {
    match native
        .instance
        .exports
        .get_global(STYLUS_INK_LEFT)
        .expect("ink global")
        .get(&mut native.store)
    {
        Value::I64(v) => v as u64,
        _ => unreachable!("ink global is i64"),
    }
}

/// Read the WASM ink-status global directly. The host sets it to 1 when the
/// meter is exhausted, so a trap with status 1 is an out-of-ink.
fn ink_status(native: &mut NativeInstance<NoopEvmApi>) -> u32 {
    match native
        .instance
        .exports
        .get_global(STYLUS_INK_STATUS)
        .expect("ink status global")
        .get(&mut native.store)
    {
        Value::I32(v) => v as u32,
        _ => unreachable!("ink status global is i32"),
    }
}

// ── 65536 (> u16::MAX): traps at ArbOS >= 59, succeeds before ───────

#[test]
fn grow_65536_traps_out_of_ink_at_v60() {
    assert_eq!(run(65_536, ARBOS_60), Outcome::OutOfInk);
}

#[test]
fn grow_65536_succeeds_at_v58() {
    // Guard inert: the host truncates 65536 -> 0 and takes the base-ink path.
    assert_eq!(run(65_536, ARBOS_58), Outcome::Returned(0));
}

#[test]
fn grow_65536_at_v58_barely_consumes_ink() {
    let compile = CompileConfig::version(1, true).expect("compile config v1");
    let wasm = wat::parse_bytes(wat_grow(65_536).as_bytes())
        .expect("wat compiles")
        .into_owned();
    let mut native = NativeInstance::from_bytes_with_pages(
        &wasm,
        NoopEvmApi,
        evm_data_stub(ARBOS_58),
        &compile,
        StylusConfig::default(),
        0,
        0,
        0,
        PAGE_GAS,
        PAGE_LIMIT,
        ARBOS_58,
    )
    .expect("instantiate");
    seed_meter(&mut native);

    let before = ink_global(&mut native);
    let entrypoint = entrypoint_of(&native);
    let status = entrypoint.call(&mut native.store, 0).expect("must succeed");
    assert_eq!(status, 0);
    let after = ink_global(&mut native);
    // The base-ink charge is a tiny constant; a full-budget burn would be ~i64::MAX.
    let consumed = before - after;
    assert!(
        consumed < 1_000_000_000,
        "v58/65536 must take the cheap base-ink path, consumed {consumed}"
    );
}

// ── small operand: identical cheap success on both versions ─────────

#[test]
fn grow_16_succeeds_on_both_versions() {
    assert_eq!(run(16, ARBOS_60), Outcome::Returned(0));
    assert_eq!(run(16, ARBOS_58), Outcome::Returned(0));
}

// ── Test fixtures ───────────────────────────────────────────────────

fn evm_data_stub(arbos_version: u64) -> EvmData {
    EvmData {
        arbos_version,
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
