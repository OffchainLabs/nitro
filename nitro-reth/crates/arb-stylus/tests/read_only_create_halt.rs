//! End-to-end test for the read-only `create1`/`create2` halt.
//!
//! A read-only `CreateResponse::Fail` returns `Err` before `buy_gas`, halting
//! the WASM frame (`Failure`) with the ink preserved.
//!
//! In a `STATICCALL` context the `EvmApi` returns
//! `(CreateResponse::Fail("write protection"), 0, Gas(0), pages)`. These tests
//! drive the real `host::create1`/`create2` imports through `NativeInstance`
//! against an `EvmApi` that returns that exact tuple and assert the entrypoint
//! traps with the meter still ready (a frame `Failure`, not an out-of-ink and not
//! a success). The companion `Success` and `NormalFailure` arms show a `Success`
//! response (deployed or zero address) lets the program continue, so ONLY `Fail`
//! halts.
//!
//! This behavior is not ArbOS-version-gated — the read-only create prohibition
//! is unconditional in the EVM — so the tests run at a single version.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::{address, Address, B256, U256};
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

const PAGE_GAS: u16 = 1_000;
const PAGE_LIMIT: u16 = 128;
const SEED_INK: i64 = i64::MAX;
const DEPLOYED: Address = address!("00000000000000000000000000000000deadbeef");

/// What the injected `EvmApi` returns from `create1`/`create2`.
#[derive(Clone, Copy, Debug)]
enum CreateOutcome {
    /// The read-only short-circuit: cost 0, a `Fail` response.
    ReadOnlyFail,
    /// A normal successful create returning a deployed address.
    Success,
    /// A normal create whose inner deploy failed: a `Success(ZERO)` response, so
    /// the program continues with a zero address (matches the mutable-create
    /// failure path, which is distinct from the read-only halt).
    NormalFailure,
}

/// Program that calls `create1` once with in-bounds pointers and returns success.
/// `code_len = 0`, so every pointer (0) is valid inside page 0 and the host reads
/// no init code.
const WAT_CREATE1: &str = r#"
(module
    (import "vm_hooks" "create1"
        (func $create1 (param i32 i32 i32 i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $create1
            (i32.const 0)   ;; code_ptr
            (i32.const 0)   ;; code_len
            (i32.const 64)  ;; endowment_ptr
            (i32.const 96)  ;; contract_ptr
            (i32.const 128)) ;; ret_len_ptr
        i32.const 0
    )
)
"#;

/// Same as `WAT_CREATE1` but drives `create2` (extra salt pointer).
const WAT_CREATE2: &str = r#"
(module
    (import "vm_hooks" "create2"
        (func $create2 (param i32 i32 i32 i32 i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $create2
            (i32.const 0)    ;; code_ptr
            (i32.const 0)    ;; code_len
            (i32.const 64)   ;; endowment_ptr
            (i32.const 96)   ;; salt_ptr
            (i32.const 128)  ;; contract_ptr
            (i32.const 160)) ;; ret_len_ptr
        i32.const 0
    )
)
"#;

/// Result of invoking the entrypoint once.
#[derive(Debug, PartialEq, Eq)]
enum Outcome {
    /// Host returned `Err`, the frame trapped, and the meter was NOT exhausted
    /// (a `Failure`: the ink is preserved). This is the read-only halt.
    HaltWithInkPreserved,
    /// Host trapped with the meter exhausted (out-of-ink).
    OutOfInk,
    /// Entrypoint returned `status` (the host continued past the create).
    Returned(u32),
}

fn run(wat: &str, outcome: CreateOutcome) -> Outcome {
    let compile = CompileConfig::version(1, true).expect("compile config v1");
    let config = StylusConfig::default();
    let wasm = wat::parse_bytes(wat.as_bytes())
        .expect("wat compiles")
        .into_owned();

    let mut native = NativeInstance::from_bytes_with_pages(
        &wasm,
        ProbeEvmApi { outcome },
        evm_data_stub(),
        &compile,
        config,
        0,
        0,
        0,
        PAGE_GAS,
        PAGE_LIMIT,
        ARBOS_60,
    )
    .expect("instantiate");

    seed_meter(&mut native);

    let entrypoint: TypedFunction<u32, u32> = native
        .instance
        .exports
        .get_typed_function(&native.store, "user_entrypoint")
        .expect("entrypoint export");

    match entrypoint.call(&mut native.store, 0) {
        Ok(status) => Outcome::Returned(status),
        // Read the WASM ink-status global directly: after a trap the in-memory
        // `MeterData` is stale, but the global reflects the real meter. The host
        // sets it to 1 only on an out-of-ink; a `Fail`-driven `Err` leaves it 0.
        Err(_) => match ink_status(&mut native) {
            0 => Outcome::HaltWithInkPreserved,
            _ => Outcome::OutOfInk,
        },
    }
}

/// Read the WASM ink-status global directly (independent of `MeterData`).
fn ink_status(native: &mut NativeInstance<ProbeEvmApi>) -> u32 {
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

fn seed_meter(native: &mut NativeInstance<ProbeEvmApi>) {
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

// ── create1 ─────────────────────────────────────────────────────────

#[test]
fn create1_read_only_fail_halts_frame() {
    assert_eq!(
        run(WAT_CREATE1, CreateOutcome::ReadOnlyFail),
        Outcome::HaltWithInkPreserved,
        "a read-only create must halt the frame, not continue"
    );
}

#[test]
fn create1_success_continues() {
    assert_eq!(
        run(WAT_CREATE1, CreateOutcome::Success),
        Outcome::Returned(0),
        "a successful create must let the program continue"
    );
}

#[test]
fn create1_normal_failure_continues() {
    // A mutable-create failure surfaces as `Success(ZERO)`, so the program
    // continues with a zero address — only `Fail` (the read-only/write-protection
    // response) halts the frame.
    assert_eq!(
        run(WAT_CREATE1, CreateOutcome::NormalFailure),
        Outcome::Returned(0)
    );
}

// ── create2 ─────────────────────────────────────────────────────────

#[test]
fn create2_read_only_fail_halts_frame() {
    assert_eq!(
        run(WAT_CREATE2, CreateOutcome::ReadOnlyFail),
        Outcome::HaltWithInkPreserved
    );
}

#[test]
fn create2_success_continues() {
    assert_eq!(
        run(WAT_CREATE2, CreateOutcome::Success),
        Outcome::Returned(0)
    );
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
struct ProbeEvmApi {
    outcome: CreateOutcome,
}

impl ProbeEvmApi {
    fn create_response(&self, pages: (u16, u16)) -> (CreateResponse, u32, Gas, (u16, u16)) {
        match self.outcome {
            CreateOutcome::ReadOnlyFail => (
                CreateResponse::Fail("write protection".into()),
                0,
                Gas(0),
                pages,
            ),
            CreateOutcome::Success => (CreateResponse::Success(DEPLOYED), 0, Gas(0), pages),
            CreateOutcome::NormalFailure => {
                (CreateResponse::Success(Address::ZERO), 0, Gas(0), pages)
            }
        }
    }
}

impl EvmApi for ProbeEvmApi {
    fn get_bytes32(&mut self, _key: B256, _gas: Gas) -> eyre::Result<(B256, Gas)> {
        unreachable!()
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
        pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        Ok(self.create_response(pages))
    }
    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: U256,
        _salt: B256,
        _gas: Gas,
        pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        Ok(self.create_response(pages))
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
