use std::collections::HashMap;

use alloy_primitives::{Address, Log, B256, U256};
use arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_LAST_CODE_CACHE_FIX;
use revm::Database;

use crate::{
    evm_api::{CreateResponse, EvmApi, UserOutcomeKind},
    ink::Gas,
};

/// EIP-2929 gas costs for storage operations.
const COLD_SLOAD_COST: u64 = 2100;
const WARM_STORAGE_READ_COST: u64 = 100;
const COLD_ACCOUNT_ACCESS_COST: u64 = 2600;
const WARM_ACCOUNT_ACCESS_COST: u64 = 100;

/// Extra gas charged when loading account code in Stylus.
/// Matches Go: `cfg.MaxCodeSize() / params.DefaultMaxCodeSize * params.ExtcodeSizeGasEIP150`
/// = 24576 / 24576 * 700 = 700.
const WASM_EXT_CODE_COST: u64 = 700;

// ── Type-erased journal access ──────────────────────────────────────

/// Flattened SSTORE result without revm generics.
pub struct SStoreInfo {
    pub is_cold: bool,
    pub original_value: U256,
    pub present_value: U256,
    pub new_value: U256,
}

/// Object-safe trait wrapping `Journal<DB>` operations needed by Stylus.
///
/// By erasing the `DB` type parameter, [`StylusEvmApi`] becomes non-generic
/// and trivially satisfies `'static` (required by wasmer's `FunctionEnv`).
pub trait JournalAccess {
    fn sload(&mut self, addr: Address, key: U256) -> eyre::Result<(U256, bool)>;
    fn sstore(&mut self, addr: Address, key: U256, value: U256) -> eyre::Result<SStoreInfo>;
    fn tload(&mut self, addr: Address, key: U256) -> U256;
    fn tstore(&mut self, addr: Address, key: U256, value: U256);
    fn log(&mut self, log: Log);
    fn account_balance(&mut self, addr: Address) -> eyre::Result<(U256, bool)>;
    fn account_code(&mut self, addr: Address) -> eyre::Result<(Vec<u8>, bool)>;
    fn account_codehash(&mut self, addr: Address) -> eyre::Result<(B256, bool)>;
    fn address_in_access_list(&self, addr: Address) -> bool;
    fn add_address_to_access_list(&mut self, addr: Address);
    fn is_account_empty(&mut self, addr: Address) -> eyre::Result<bool>;
}

impl<DB: Database> JournalAccess for revm::Journal<DB> {
    fn sload(&mut self, addr: Address, key: U256) -> eyre::Result<(U256, bool)> {
        let result = self
            .inner
            .sload(&mut self.database, addr, key, false)
            .map_err(|e| eyre::eyre!("sload failed: {e:?}"))?;
        Ok((result.data, result.is_cold))
    }

    fn sstore(&mut self, addr: Address, key: U256, value: U256) -> eyre::Result<SStoreInfo> {
        let result = self
            .inner
            .sstore(&mut self.database, addr, key, value, false)
            .map_err(|e| eyre::eyre!("sstore failed: {e:?}"))?;
        Ok(SStoreInfo {
            is_cold: result.is_cold,
            original_value: result.data.original_value,
            present_value: result.data.present_value,
            new_value: result.data.new_value,
        })
    }

    fn tload(&mut self, addr: Address, key: U256) -> U256 {
        self.inner.tload(addr, key)
    }

    fn tstore(&mut self, addr: Address, key: U256, value: U256) {
        self.inner.tstore(addr, key, value);
    }

    fn log(&mut self, log: Log) {
        self.inner.log(log);
    }

    fn account_balance(&mut self, addr: Address) -> eyre::Result<(U256, bool)> {
        let result = self
            .inner
            .load_account(&mut self.database, addr)
            .map_err(|e| eyre::eyre!("load_account failed: {e:?}"))?;
        Ok((result.data.info.balance, result.is_cold))
    }

    fn account_code(&mut self, addr: Address) -> eyre::Result<(Vec<u8>, bool)> {
        let result = self
            .inner
            .load_code(&mut self.database, addr)
            .map_err(|e| eyre::eyre!("load_code failed: {e:?}"))?;
        let code = result
            .data
            .info
            .code
            .as_ref()
            .map(|c: &revm::bytecode::Bytecode| c.original_bytes().to_vec())
            .unwrap_or_default();
        Ok((code, result.is_cold))
    }

    fn account_codehash(&mut self, addr: Address) -> eyre::Result<(B256, bool)> {
        let result = self
            .inner
            .load_account(&mut self.database, addr)
            .map_err(|e| eyre::eyre!("load_account failed: {e:?}"))?;
        let is_cold = result.is_cold;
        // EIP-1052: an empty account's code hash is zero, not `keccak("")`.
        let hash = if result.data.info.is_empty() {
            B256::ZERO
        } else {
            result.data.info.code_hash
        };
        Ok((hash, is_cold))
    }

    fn address_in_access_list(&self, addr: Address) -> bool {
        // Pre-warmed: precompiles, coinbase, EIP-2930 access list.
        if self.inner.warm_addresses.is_warm(&addr) {
            return true;
        }
        // EIP-2929 access list lives on each account as a (transaction_id, Cold flag)
        // pair. A reverted sub-call leaves the account in the state HashMap but
        // re-marked Cold via JournalEntry::AccountWarmed::revert, so a plain
        // contains_key check would miss the revert and report stale warmth.
        if let Some(account) = self.inner.state.get(&addr) {
            return !account.is_cold_transaction_id(self.inner.transaction_id);
        }
        false
    }

    fn add_address_to_access_list(&mut self, addr: Address) {
        // Load the account to mark it warm in the state map.
        let _ = self.inner.load_account(&mut self.database, addr);
    }

    fn is_account_empty(&mut self, addr: Address) -> eyre::Result<bool> {
        let result = self
            .inner
            .load_account(&mut self.database, addr)
            .map_err(|e| eyre::eyre!("load_account failed: {e:?}"))?;
        let acc = result.data;
        Ok(acc.info.balance.is_zero()
            && acc.info.nonce == 0
            && acc.info.code_hash == revm::primitives::KECCAK_EMPTY)
    }
}

// ── StylusEvmApi ────────────────────────────────────────────────────

/// Result from a sub-call (CALL, DELEGATECALL, STATICCALL).
pub struct SubCallResult {
    pub output: Vec<u8>,
    pub gas_cost: u64,
    pub success: bool,
    /// Gas refund accumulated during the sub-call (EIP-3529 SSTORE refunds).
    pub refund: i64,
    /// Page counters after the sub-call returns.
    pub pages: (u16, u16),
}

/// Result from a CREATE/CREATE2 operation.
pub struct SubCreateResult {
    pub address: Option<Address>,
    pub output: Vec<u8>,
    pub gas_cost: u64,
    pub pages: (u16, u16),
}

/// Type-erased function pointer for executing sub-calls from Stylus.
///
/// `call_type`: `0=CALL`, `1=DELEGATECALL`, `2=STATICCALL`.
/// `pages` carries the parent's (open, ever) counters into the new frame.
///
/// The first pointer is the type-erased revm `Context`; the second points at
/// the `ArbPrecompileCtx` shared by the precompile closures and dispatch
/// path. Both must remain live for the duration of the trampoline call.
pub type DoCallFn = fn(
    *mut (),
    *const (),
    u8,
    Address,
    Address,
    Address,
    &[u8],
    u64,
    U256,
    (u16, u16),
) -> SubCallResult;

/// Type-erased function pointer for executing CREATE/CREATE2 from Stylus.
pub type DoCreateFn =
    fn(*mut (), *const (), Address, &[u8], u64, U256, Option<B256>, (u16, u16)) -> SubCreateResult;

/// Per-call storage cache entry.
struct StorageCacheEntry {
    /// Current value (may be dirty from a write).
    value: B256,
    /// Original value from the journal (None = written before first read).
    known: Option<B256>,
}

impl StorageCacheEntry {
    fn known(value: B256) -> Self {
        Self {
            value,
            known: Some(value),
        }
    }

    fn unknown(value: B256) -> Self {
        Self { value, known: None }
    }

    fn dirty(&self) -> bool {
        self.known != Some(self.value)
    }
}

/// Per-call storage cache: avoids repeat journal hits and charges the
/// `evm_api_gas` surcharge only on the first miss per slot.
struct StorageCache {
    slots: HashMap<B256, StorageCacheEntry>,
    reads: u32,
    writes: u32,
}

impl StorageCache {
    fn new() -> Self {
        Self {
            slots: HashMap::new(),
            reads: 0,
            writes: 0,
        }
    }

    fn read_gas(&mut self) -> Gas {
        self.reads += 1;
        match self.reads {
            0..=32 => Gas(0),
            33..=128 => Gas(2),
            _ => Gas(10),
        }
    }

    fn write_gas(&mut self) -> Gas {
        self.writes += 1;
        match self.writes {
            0..=8 => Gas(0),
            9..=64 => Gas(7),
            _ => Gas(10),
        }
    }
}

/// Concrete [`EvmApi`] bridging WASM host function calls to revm's journaled state.
///
/// Uses a type-erased raw pointer to [`JournalAccess`] so that the `DB` generic
/// parameter is erased. This allows `StylusEvmApi` to be `'static` without
/// requiring `DB: 'static`, which is needed for wasmer's `FunctionEnv`.
///
/// # Safety
///
/// Wasmer executes WASM programs synchronously on the calling thread, so no
/// cross-thread sharing occurs despite the `Send` bound on [`EvmApi`].
/// The raw pointer must remain valid for the lifetime of the Stylus execution.
pub struct StylusEvmApi {
    /// Type-erased raw pointer to the journal (implements [`JournalAccess`]).
    journal: *mut dyn JournalAccess,
    /// The contract address being executed.
    address: Address,
    /// The caller (msg.sender) of the current contract.
    caller: Address,
    /// Value of the current call (needed for DELEGATECALL forwarding).
    call_value: U256,
    /// Per-call storage cache.
    storage_cache: StorageCache,
    /// Accumulated SSTORE refund (EIP-3529) from flush operations.
    sstore_refund: i64,
    /// Return data from the last sub-call.
    return_data: Vec<u8>,
    /// Whether the current execution context is read-only (STATICCALL).
    read_only: bool,
    /// ArbOS version — flush semantics are version-gated at v50.
    arbos_version: u64,
    /// Type-erased context pointer and callbacks for sub-calls.
    ctx_ptr: *mut (),
    /// Type-erased pointer to the `ArbPrecompileCtx` carried alongside
    /// `ctx_ptr` so the trampoline can access per-block / per-tx state
    /// without going through a thread-local.
    precompile_ctx_ptr: *const (),
    do_call: Option<DoCallFn>,
    do_create: Option<DoCreateFn>,
    /// Most recent `account_code` result; a same-address repeat read is free.
    last_code: Option<(Address, Vec<u8>)>,
}

// SAFETY: `wasmer::FunctionEnv::new<T>` requires `T: Send + 'static`, so
// `StylusEvmApi` (held inside a `WasmEnv` that is passed to wasmer host
// functions) must be `Send`. The `*mut dyn JournalAccess` and the
// `*mut ()` / `*const ()` context handles inside this struct make it
// `!Send` by default. They are only ever dereferenced by host functions
// invoked synchronously on the same thread that built and is currently
// driving the `Instance` — wasmer never moves the env across threads
// while host calls are in flight, so the raw pointers are never observed
// from any thread other than the one that constructed them.
unsafe impl Send for StylusEvmApi {}

impl StylusEvmApi {
    /// Create a new StylusEvmApi from a raw pointer to a revm Journal.
    ///
    /// # Safety
    ///
    /// The `journal` pointer must remain valid for the lifetime of this struct.
    /// The caller must ensure exclusive mutable access through this pointer.
    /// If `ctx_ptr` is provided, it must also remain valid.
    #[allow(clippy::too_many_arguments)]
    pub unsafe fn new<DB: Database>(
        journal: *mut revm::Journal<DB>,
        address: Address,
        caller: Address,
        call_value: U256,
        read_only: bool,
        arbos_version: u64,
        ctx_ptr: *mut (),
        precompile_ctx_ptr: *const (),
        do_call: Option<DoCallFn>,
        do_create: Option<DoCreateFn>,
    ) -> Self {
        let journal: *mut dyn JournalAccess = {
            let r: &mut (dyn JournalAccess + '_) = &mut *journal;
            #[allow(clippy::unnecessary_cast)]
            {
                r as *mut (dyn JournalAccess + '_) as *mut dyn JournalAccess
            }
        };
        Self {
            journal,
            address,
            caller,
            call_value,
            storage_cache: StorageCache::new(),
            sstore_refund: 0,
            return_data: Vec::new(),
            read_only,
            arbos_version,
            ctx_ptr,
            precompile_ctx_ptr,
            do_call,
            do_create,
            last_code: None,
        }
    }

    /// Get a mutable reference to the type-erased journal.
    fn journal(&mut self) -> &mut dyn JournalAccess {
        // SAFETY: `self.journal` was set by `Self::new`, whose caller
        // contract requires the pointer to remain valid for the lifetime
        // of this struct and grants exclusive access. The dispatch path
        // upholds this: the EVM context owning the journal is kept alive
        // and is not borrowed elsewhere while host functions run.
        unsafe { &mut *self.journal }
    }

    /// Return the accumulated SSTORE refund from flush operations.
    pub fn sstore_refund(&self) -> i64 {
        self.sstore_refund
    }
}

impl std::fmt::Debug for StylusEvmApi {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusEvmApi")
            .field("address", &self.address)
            .field("read_only", &self.read_only)
            .field("cache_size", &self.storage_cache.slots.len())
            .finish()
    }
}

impl EvmApi for StylusEvmApi {
    fn get_bytes32(&mut self, key: B256, evm_api_gas_to_use: Gas) -> eyre::Result<(B256, Gas)> {
        let mut cost = self.storage_cache.read_gas();

        let value = if let Some(entry) = self.storage_cache.slots.get(&key) {
            entry.value
        } else {
            let storage_key = U256::from_be_bytes(key.0);
            let addr = self.address;
            let (value_u256, is_cold) = self.journal().sload(addr, storage_key)?;
            let value = B256::from(value_u256.to_be_bytes());

            let sload_cost = if is_cold {
                COLD_SLOAD_COST
            } else {
                WARM_STORAGE_READ_COST
            };
            cost = Gas(cost
                .0
                .saturating_add(sload_cost)
                .saturating_add(evm_api_gas_to_use.0));

            self.storage_cache
                .slots
                .insert(key, StorageCacheEntry::known(value));
            value
        };

        Ok((value, cost))
    }

    fn cache_bytes32(&mut self, key: B256, value: B256) -> eyre::Result<Gas> {
        let cost = self.storage_cache.write_gas();
        match self.storage_cache.slots.get_mut(&key) {
            Some(entry) => entry.value = value,
            None => {
                self.storage_cache
                    .slots
                    .insert(key, StorageCacheEntry::unknown(value));
            }
        }
        Ok(cost)
    }

    fn flush_storage_cache(
        &mut self,
        clear: bool,
        gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)> {
        // Collect dirty entries
        let dirty: Vec<(B256, B256)> = self
            .storage_cache
            .slots
            .iter()
            .filter(|(_, v)| v.dirty())
            .map(|(k, v)| (*k, v.value))
            .collect();

        if clear {
            self.storage_cache.slots.clear();
        } else {
            // Mark all entries as known (clean)
            for entry in self.storage_cache.slots.values_mut() {
                entry.known = Some(entry.value);
            }
        }

        if dirty.is_empty() {
            return Ok((Gas(0), UserOutcomeKind::Success));
        }

        if self.read_only {
            return Ok((Gas(0), UserOutcomeKind::Failure));
        }

        let mut total_gas = 0u64;
        let mut remaining = gas_left.0;
        let mut is_out_of_gas = false;

        for (key, value) in &dirty {
            let storage_key = U256::from_be_bytes(key.0);
            let storage_value = U256::from_be_bytes(value.0);

            let addr = self.address;
            let info = self.journal().sstore(addr, storage_key, storage_value)?;

            let sstore_cost = sstore_gas_cost(&info);
            if sstore_cost > remaining {
                is_out_of_gas = true;
                total_gas = gas_left.0;
                break;
            }
            remaining -= sstore_cost;
            total_gas += sstore_cost;
            self.sstore_refund += sstore_refund(&info);
        }

        // A budget that was exhausted — by partial OOG or by hitting exactly
        // zero — must surface as a non-Success outcome so the caller traps.
        if is_out_of_gas || remaining == 0 {
            const ARBOS_VERSION_DIA: u64 = 50;
            let outcome = if self.arbos_version < ARBOS_VERSION_DIA {
                UserOutcomeKind::Failure
            } else {
                UserOutcomeKind::OutOfInk
            };
            return Ok((Gas(total_gas), outcome));
        }

        Ok((Gas(total_gas), UserOutcomeKind::Success))
    }

    fn get_transient_bytes32(&mut self, key: B256) -> eyre::Result<B256> {
        let storage_key = U256::from_be_bytes(key.0);
        let addr = self.address;
        let value = self.journal().tload(addr, storage_key);
        Ok(B256::from(value.to_be_bytes()))
    }

    fn set_transient_bytes32(&mut self, key: B256, value: B256) -> eyre::Result<UserOutcomeKind> {
        if self.read_only {
            return Ok(UserOutcomeKind::Failure);
        }
        let storage_key = U256::from_be_bytes(key.0);
        let storage_value = U256::from_be_bytes(value.0);
        let addr = self.address;
        self.journal().tstore(addr, storage_key, storage_value);
        Ok(UserOutcomeKind::Success)
    }

    fn contract_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        value: U256,
        pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        if self.read_only && !value.is_zero() {
            self.return_data = Vec::new();
            return Ok((0, Gas(0), UserOutcomeKind::Failure, pages));
        }

        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                    pages,
                ));
            }
        };

        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &value, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure, pages));
        }

        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        let gas = if !value.is_zero() {
            gas.saturating_add(2300)
        } else {
            gas
        };

        let result = (do_call)(
            self.ctx_ptr,
            self.precompile_ctx_ptr,
            0,
            contract,
            self.address,
            contract,
            calldata,
            gas,
            value,
            pages,
        );

        self.return_data = result.output;
        let cost = base_cost.saturating_add(result.gas_cost);
        self.sstore_refund = self.sstore_refund.saturating_add(result.refund);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((
            self.return_data.len() as u32,
            Gas(cost),
            outcome,
            result.pages,
        ))
    }

    fn delegate_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                    pages,
                ));
            }
        };

        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure, pages));
        }

        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        let result = (do_call)(
            self.ctx_ptr,
            self.precompile_ctx_ptr,
            1,
            contract,
            self.caller,
            self.address,
            calldata,
            gas,
            self.call_value,
            pages,
        );

        self.return_data = result.output;
        let cost = base_cost.saturating_add(result.gas_cost);
        self.sstore_refund = self.sstore_refund.saturating_add(result.refund);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((
            self.return_data.len() as u32,
            Gas(cost),
            outcome,
            result.pages,
        ))
    }

    fn static_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        pages: (u16, u16),
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind, (u16, u16))> {
        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                    pages,
                ));
            }
        };

        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure, pages));
        }

        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        let result = (do_call)(
            self.ctx_ptr,
            self.precompile_ctx_ptr,
            2,
            contract,
            self.address,
            contract,
            calldata,
            gas,
            U256::ZERO,
            pages,
        );

        self.return_data = result.output;
        let cost = base_cost.saturating_add(result.gas_cost);
        self.sstore_refund = self.sstore_refund.saturating_add(result.refund);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((
            self.return_data.len() as u32,
            Gas(cost),
            outcome,
            result.pages,
        ))
    }

    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        gas: Gas,
        pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        if self.read_only {
            self.return_data = Vec::new();
            return Ok((
                CreateResponse::Fail("write protection".into()),
                0,
                Gas(0),
                pages,
            ));
        }

        let do_create = match self.do_create {
            Some(f) => f,
            None => {
                self.return_data = b"creates not available".to_vec();
                return Ok((
                    CreateResponse::Fail("not available".into()),
                    self.return_data.len() as u32,
                    Gas(0),
                    pages,
                ));
            }
        };

        let base_cost: u64 = 32000;
        if gas.0 < base_cost {
            self.return_data = Vec::new();
            return Ok((
                CreateResponse::Fail("out of gas".into()),
                0,
                Gas(gas.0),
                pages,
            ));
        }
        let remaining = gas.0 - base_cost;
        let one_64th = remaining / 64;
        let call_gas = remaining - one_64th;

        let result = (do_create)(
            self.ctx_ptr,
            self.precompile_ctx_ptr,
            self.address,
            &code,
            call_gas,
            endowment,
            None,
            pages,
        );

        self.return_data = result.output.clone();
        let cost = base_cost.saturating_add(result.gas_cost);

        let response = match result.address {
            Some(addr) => CreateResponse::Success(addr),
            None => {
                if self.return_data.is_empty() {
                    CreateResponse::Fail("create failed".into())
                } else {
                    CreateResponse::Fail("reverted".into())
                }
            }
        };

        Ok((
            response,
            self.return_data.len() as u32,
            Gas(cost),
            result.pages,
        ))
    }

    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        salt: B256,
        gas: Gas,
        pages: (u16, u16),
    ) -> eyre::Result<(CreateResponse, u32, Gas, (u16, u16))> {
        if self.read_only {
            self.return_data = Vec::new();
            return Ok((
                CreateResponse::Fail("write protection".into()),
                0,
                Gas(0),
                pages,
            ));
        }

        let do_create = match self.do_create {
            Some(f) => f,
            None => {
                self.return_data = b"creates not available".to_vec();
                return Ok((
                    CreateResponse::Fail("not available".into()),
                    self.return_data.len() as u32,
                    Gas(0),
                    pages,
                ));
            }
        };

        let keccak_words = (code.len() as u64).div_ceil(32);
        let keccak_cost = keccak_words.saturating_mul(6);
        let base_cost = 32000u64.saturating_add(keccak_cost);
        if gas.0 < base_cost {
            self.return_data = Vec::new();
            return Ok((
                CreateResponse::Fail("out of gas".into()),
                0,
                Gas(gas.0),
                pages,
            ));
        }
        let remaining = gas.0 - base_cost;
        let one_64th = remaining / 64;
        let call_gas = remaining - one_64th;

        let result = (do_create)(
            self.ctx_ptr,
            self.precompile_ctx_ptr,
            self.address,
            &code,
            call_gas,
            endowment,
            Some(salt),
            pages,
        );

        self.return_data = result.output.clone();
        let cost = base_cost.saturating_add(result.gas_cost);

        let response = match result.address {
            Some(addr) => CreateResponse::Success(addr),
            None => {
                if self.return_data.is_empty() {
                    CreateResponse::Fail("create failed".into())
                } else {
                    CreateResponse::Fail("reverted".into())
                }
            }
        };

        Ok((
            response,
            self.return_data.len() as u32,
            Gas(cost),
            result.pages,
        ))
    }

    fn get_return_data(&self) -> Vec<u8> {
        self.return_data.clone()
    }

    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> eyre::Result<()> {
        if self.read_only {
            return Err(eyre::eyre!("cannot emit log in static context"));
        }

        let topic_bytes = (topics as usize) * 32;
        if data.len() < topic_bytes {
            return Err(eyre::eyre!("log data too short for {topics} topics"));
        }

        let mut topic_list = Vec::with_capacity(topics as usize);
        for i in 0..topics as usize {
            let start = i * 32;
            let mut bytes = [0u8; 32];
            bytes.copy_from_slice(&data[start..start + 32]);
            topic_list.push(B256::from(bytes));
        }

        let log_data = data[topic_bytes..].to_vec();

        let addr = self.address;
        let log = Log::new(addr, topic_list, log_data.into()).expect("too many log topics");

        self.journal().log(log);
        Ok(())
    }

    fn account_balance(&mut self, address: Address) -> eyre::Result<(U256, Gas)> {
        let (balance, is_cold) = self.journal().account_balance(address)?;
        // WasmAccountTouchCost(withCode=false): cold/warm access cost
        let gas_cost = if is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((balance, Gas(gas_cost)))
    }

    fn account_code(
        &mut self,
        _arbos_version: u64,
        address: Address,
        gas_left: Gas,
    ) -> eyre::Result<(Vec<u8>, Gas)> {
        if let Some((stored, data)) = self.last_code.as_ref() {
            if *stored == address {
                return Ok((data.clone(), Gas(0)));
            }
        }
        let (code, is_cold) = self.journal().account_code(address)?;
        // WasmAccountTouchCost(withCode=true): extCodeCost + cold/warm access cost
        let access_cost = if is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        let gas_cost = WASM_EXT_CODE_COST + access_cost;
        // If insufficient gas, return empty code but still charge
        if gas_left.0 < gas_cost {
            return Ok((Vec::new(), Gas(gas_cost)));
        }
        if !code.is_empty() || self.arbos_version < ARBOS_VERSION_STYLUS_LAST_CODE_CACHE_FIX {
            self.last_code = Some((address, code.clone()));
        }
        Ok((code, Gas(gas_cost)))
    }

    fn account_codehash(&mut self, address: Address) -> eyre::Result<(B256, Gas)> {
        let (hash, is_cold) = self.journal().account_codehash(address)?;
        // WasmAccountTouchCost(withCode=false)
        let gas_cost = if is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((hash, Gas(gas_cost)))
    }

    fn capture_hostio(
        &mut self,
        _name: &str,
        _args: &[u8],
        _outs: &[u8],
        _start_ink: crate::ink::Ink,
        _end_ink: crate::ink::Ink,
    ) {
        // Debug tracing — no-op in production.
    }
}

/// Compute the base gas cost for a CALL from Stylus.
///
/// Matches Go's `WasmCallCost`: EIP-2929 warm/cold access + value transfer +
/// new account creation cost. Returns `(cost, out_of_gas)`.
fn wasm_call_cost(
    journal: &mut dyn JournalAccess,
    contract: Address,
    value: &U256,
    budget: u64,
) -> (u64, bool) {
    let mut total: u64 = 0;

    // Static cost: warm storage read (computation)
    total += WARM_ACCOUNT_ACCESS_COST; // 100
    if total > budget {
        return (total, true);
    }

    // Cold access cost
    let warm = journal.address_in_access_list(contract);
    if !warm {
        journal.add_address_to_access_list(contract);
        let cold_cost = COLD_ACCOUNT_ACCESS_COST - WARM_ACCOUNT_ACCESS_COST; // 2500
        total = total.saturating_add(cold_cost);
        if total > budget {
            return (total, true);
        }
    }

    let transfers_value = !value.is_zero();
    if transfers_value {
        // Check if target is empty (for new account cost)
        if let Ok(empty) = journal.is_account_empty(contract) {
            if empty {
                total = total.saturating_add(25000); // CallNewAccountGas
                if total > budget {
                    return (total, true);
                }
            }
        }
        // Value transfer cost
        total = total.saturating_add(9000); // CallValueTransferGas
        if total > budget {
            return (total, true);
        }
    }

    (total, false)
}

/// EIP-3529 SSTORE refund constants (post-London).
const SSTORE_CLEARS_SCHEDULE: i64 = 4_800; // WARM_SSTORE_RESET(2900) + ACCESS_LIST_STORAGE_KEY(1900)
const SSTORE_SET_REFUND: i64 = 19_900; // SSTORE_SET(20000) - WARM_STORAGE_READ(100)
const SSTORE_RESET_REFUND: i64 = 2_800; // WARM_SSTORE_RESET(2900) - WARM_STORAGE_READ(100)

/// Compute SSTORE refund following revm's `sstore_refund` formula (Istanbul+/EIP-3529).
fn sstore_refund(info: &SStoreInfo) -> i64 {
    let original = info.original_value;
    let present = info.present_value;
    let new = info.new_value;

    // No-op: new equals current value
    if new == present {
        return 0;
    }

    // Refund for clearing on first write to a slot whose original is non-zero
    if original == present && new.is_zero() {
        return SSTORE_CLEARS_SCHEDULE;
    }

    let mut refund: i64 = 0;

    // If original is non-zero, track clearing/un-clearing of the slot
    if !original.is_zero() {
        if present.is_zero() {
            // Slot was previously cleared in this tx; un-clear it now
            refund -= SSTORE_CLEARS_SCHEDULE;
        } else if new.is_zero() {
            // Now clearing a previously non-zero slot
            refund += SSTORE_CLEARS_SCHEDULE;
        }
    }

    // Refund for restoring the slot to its original value
    if original == new {
        if original.is_zero() {
            refund += SSTORE_SET_REFUND;
        } else {
            refund += SSTORE_RESET_REFUND;
        }
    }

    refund
}

/// Compute SSTORE gas cost following EIP-2929 + EIP-3529 (post-London).
fn sstore_gas_cost(info: &SStoreInfo) -> u64 {
    let base = if info.original_value == info.new_value {
        WARM_STORAGE_READ_COST
    } else if info.original_value == info.present_value {
        if info.original_value.is_zero() {
            20_000 // SSTORE_SET_GAS
        } else {
            2_900 // SSTORE_RESET_GAS (5000 - 2100)
        }
    } else {
        WARM_STORAGE_READ_COST
    };

    let cold_cost = if info.is_cold { COLD_SLOAD_COST } else { 0 };
    base + cold_cost
}

// SSTORE gas + refund parity tests for the 9 canonical EIP-2200 cases
// plus the EIP-3529 refund schedule.
#[cfg(test)]
mod sstore_parity_tests {
    use super::{sstore_gas_cost, sstore_refund, SStoreInfo};
    use alloy_primitives::U256;

    fn info(original: u64, present: u64, new: u64, is_cold: bool) -> SStoreInfo {
        SStoreInfo {
            is_cold,
            original_value: U256::from(original),
            present_value: U256::from(present),
            new_value: U256::from(new),
        }
    }

    // ── EIP-2200 base costs (warm-access, EIP-2929/3529 adjusted) ─────

    /// Case 1 (noop on untouched slot): `current == value`, `original == current`.
    /// Expected: `WarmStorageReadCostEIP2929 = 100`.
    #[test]
    fn case_1_noop_untouched_warm() {
        assert_eq!(sstore_gas_cost(&info(5, 5, 5, false)), 100);
        assert_eq!(sstore_refund(&info(5, 5, 5, false)), 0);
    }

    /// Case 2.1.1 (create slot): `original == current == 0`, `value != 0`.
    /// Expected: `SstoreSetGasEIP2200 = 20_000`.
    #[test]
    fn case_2_1_1_create_slot() {
        assert_eq!(sstore_gas_cost(&info(0, 0, 5, false)), 20_000);
        assert_eq!(sstore_refund(&info(0, 0, 5, false)), 0);
    }

    /// Case 2.1.2 (update clean): `original == current != 0`, `value != 0`, `value != original`.
    /// Expected: `SstoreResetGasEIP2200 - ColdSloadCostEIP2929 = 2_900`.
    #[test]
    fn case_2_1_2_update_clean() {
        assert_eq!(sstore_gas_cost(&info(5, 5, 10, false)), 2_900);
        assert_eq!(sstore_refund(&info(5, 5, 10, false)), 0);
    }

    /// Case 2.1.2b (delete clean): original == current != 0, value == 0.
    /// Same base cost as 2.1.2 plus an EIP-3529 `SstoreClearsScheduleRefundEIP3529 = 4_800` refund.
    #[test]
    fn case_2_1_2b_delete_clean() {
        assert_eq!(sstore_gas_cost(&info(5, 5, 0, false)), 2_900);
        assert_eq!(sstore_refund(&info(5, 5, 0, false)), 4_800);
    }

    /// Case 2.2 (dirty update, no restore): `original != current`, `value`
    /// matches neither original nor a clearing pattern. Expected: 100.
    #[test]
    fn case_2_2_dirty_update() {
        // original=5, current=10, new=15 — pure dirty update
        assert_eq!(sstore_gas_cost(&info(5, 10, 15, false)), 100);
        assert_eq!(sstore_refund(&info(5, 10, 15, false)), 0);
    }

    /// Case 2.2.1.1 (un-clear): `original != 0`, `current == 0`, `value != 0`.
    /// Expected: 100 gas, `-clearingRefund` (−4_800).
    #[test]
    fn case_2_2_1_1_un_clear_dirty() {
        assert_eq!(sstore_gas_cost(&info(5, 0, 10, false)), 100);
        assert_eq!(sstore_refund(&info(5, 0, 10, false)), -4_800);
    }

    /// Case 2.2.1.2 (clear dirty): `original != 0`, `current != 0`, `value == 0`.
    /// Expected: 100 gas, `+clearingRefund` (+4_800).
    #[test]
    fn case_2_2_1_2_clear_dirty() {
        // original=5, present=10, new=0 — value becomes zero from a dirty state
        assert_eq!(sstore_gas_cost(&info(5, 10, 0, false)), 100);
        assert_eq!(sstore_refund(&info(5, 10, 0, false)), 4_800);
    }

    /// Case 2.2.2.1 (restore to inexistent original): `original == 0`, `value == 0`, `current !=
    /// 0`. Expected: 100 gas, refund `SstoreSetGasEIP2200 - WarmStorageReadCostEIP2929 =
    /// 19_900`.
    #[test]
    fn case_2_2_2_1_restore_to_zero_original() {
        assert_eq!(sstore_gas_cost(&info(0, 5, 0, false)), 100);
        assert_eq!(sstore_refund(&info(0, 5, 0, false)), 19_900);
    }

    /// Case 2.2.2.2 (restore to non-zero original): `original != 0`,
    /// `value == original`, `current != value`, `current != 0`.
    /// Expected: 100 gas, refund
    /// `SstoreResetGasEIP2200 - ColdSloadCostEIP2929 - WarmStorageReadCostEIP2929 = 2_800`.
    #[test]
    fn case_2_2_2_2_restore_to_nonzero_original() {
        assert_eq!(sstore_gas_cost(&info(5, 10, 5, false)), 100);
        assert_eq!(sstore_refund(&info(5, 10, 5, false)), 2_800);
    }

    // ── Cold-access surcharge (EIP-2929) ─────────────────────────────

    /// Cold slot access adds `ColdSloadCostEIP2929 = 2_100` to the base.
    #[test]
    fn cold_access_adds_2100() {
        assert_eq!(sstore_gas_cost(&info(5, 5, 5, true)), 100 + 2_100);
        assert_eq!(sstore_gas_cost(&info(5, 5, 10, true)), 2_900 + 2_100);
        assert_eq!(sstore_gas_cost(&info(0, 0, 5, true)), 20_000 + 2_100);
        assert_eq!(sstore_gas_cost(&info(5, 10, 0, true)), 100 + 2_100);
    }

    // ── Combined refund patterns (step 4 + step 5 can stack) ────────

    /// `original != 0`, `current == 0` (un-clear refund of −4_800) AND
    /// `value == original` (restore refund of +2_800) stack, netting −2_000.
    /// Observable if an SSTORE clears then restores a slot within one tx.
    #[test]
    fn un_clear_plus_restore_stacks() {
        // original=5, current=0, new=5 — effectively restoring after a delete
        assert_eq!(sstore_gas_cost(&info(5, 0, 5, false)), 100);
        assert_eq!(sstore_refund(&info(5, 0, 5, false)), -4_800 + 2_800);
    }

    /// original != 0, current != 0, value == 0 (clear refund of +4_800),
    /// AND NOT value == original (so no restore refund).
    #[test]
    fn clear_dirty_without_restore() {
        // original=5, current=10, new=0 — plain clear from a dirty state
        assert_eq!(sstore_refund(&info(5, 10, 0, false)), 4_800);
    }

    // ── Multi-slot flush totals ──────────────────────────────────────

    /// An 8-slot flush exercising every case above: the gas + refund totals
    /// must equal the sum of per-case expectations.
    #[test]
    fn mixed_eight_slot_flush_totals() {
        let slots = [
            info(0x12e, 0x12f, 0x130, false),        // dirty update
            info(0x880f, 0x880f, 0x23b5, false),     // reset clean
            info(0x10906e, 0x10906e, 0x275b, false), // reset clean
            info(0, 0, 1, false),                    // create
            info(0x2fea, 0x2fea, 0xf8e2, false),     // reset clean
            info(0x26658a, 0x26658a, 0x27bd, false), // reset clean
            info(0x7e51, 0x2160, 0x7e51, false),     // restore to original
            info(0x1a5f, 0x1a5f, 0x1c50, false),     // reset clean
        ];
        let total_cost: u64 = slots.iter().map(sstore_gas_cost).sum();
        let total_refund: i64 = slots.iter().map(sstore_refund).sum();
        assert_eq!(total_cost, 100 + 2_900 * 5 + 20_000 + 100);
        assert_eq!(total_cost, 34_700);
        assert_eq!(total_refund, 2_800);
    }
}
