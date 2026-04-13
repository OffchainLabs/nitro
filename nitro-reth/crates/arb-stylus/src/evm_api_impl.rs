use std::collections::HashMap;

use alloy_primitives::{Address, Log, B256, U256};
use arbos::programs::memory::MemoryModel;
use revm::Database;

use crate::{
    evm_api::{CreateResponse, EvmApi, UserOutcomeKind},
    ink::Gas,
    pages,
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
        Ok((result.data.info.code_hash, result.is_cold))
    }

    fn address_in_access_list(&self, addr: Address) -> bool {
        // An address is "warm" if it's in the loaded state or warm_addresses
        self.inner.state.contains_key(&addr) || self.inner.warm_addresses.is_warm(&addr)
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
}

/// Result from a CREATE/CREATE2 operation.
pub struct SubCreateResult {
    pub address: Option<Address>,
    pub output: Vec<u8>,
    pub gas_cost: u64,
}

/// Type-erased function pointer for executing sub-calls from Stylus.
///
/// Parameters: (ctx_ptr, call_type, contract, caller, storage_addr, input, gas, value)
/// - `caller`: msg.sender for the new frame (preserved for DELEGATECALL)
/// - `storage_addr`: address whose storage the new frame uses
///   (= current contract for CALL/STATICCALL, = preserved storage context for DELEGATECALL)
/// call_type: 0=CALL, 1=DELEGATECALL, 2=STATICCALL
pub type DoCallFn =
    fn(*mut (), u8, Address, Address, Address, &[u8], u64, U256) -> SubCallResult;

/// Type-erased function pointer for executing CREATE/CREATE2 from Stylus.
///
/// Parameters: (ctx_ptr, caller, code, gas, endowment, salt)
/// salt=None for CREATE, Some for CREATE2.
pub type DoCreateFn = fn(*mut (), Address, &[u8], u64, U256, Option<B256>) -> SubCreateResult;

/// Per-call storage cache entry, mirroring Nitro's `StorageWord`.
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
        Self {
            value,
            known: None,
        }
    }

    fn dirty(&self) -> bool {
        self.known.map_or(true, |k| k != self.value)
    }
}

/// Per-call storage cache matching Nitro's Rust-side cache.
/// Provides O(1) lookups to avoid redundant journal accesses and
/// correctly mirrors Nitro's gas charging (only charge evm_api_gas on miss).
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
    /// Per-call storage cache matching Nitro's Rust-side cache.
    storage_cache: StorageCache,
    /// Accumulated SSTORE refund (EIP-3529) from flush operations.
    sstore_refund: i64,
    /// Return data from the last sub-call.
    return_data: Vec<u8>,
    /// Whether the current execution context is read-only (STATICCALL).
    read_only: bool,
    /// MemoryModel params for add_pages gas computation.
    free_pages: u16,
    page_gas: u16,
    /// Type-erased context pointer and callbacks for sub-calls.
    ctx_ptr: *mut (),
    do_call: Option<DoCallFn>,
    do_create: Option<DoCreateFn>,
}

// Safety: Wasmer executes synchronously on the calling thread. No cross-thread access occurs.
unsafe impl Send for StylusEvmApi {}

impl StylusEvmApi {
    /// Create a new StylusEvmApi from a raw pointer to a revm Journal.
    ///
    /// # Safety
    ///
    /// The `journal` pointer must remain valid for the lifetime of this struct.
    /// The caller must ensure exclusive mutable access through this pointer.
    /// If `ctx_ptr` is provided, it must also remain valid.
    pub unsafe fn new<DB: Database>(
        journal: *mut revm::Journal<DB>,
        address: Address,
        caller: Address,
        call_value: U256,
        read_only: bool,
        free_pages: u16,
        page_gas: u16,
        ctx_ptr: *mut (),
        do_call: Option<DoCallFn>,
        do_create: Option<DoCreateFn>,
    ) -> Self {
        // Annotate with '_ to avoid the default 'static bound on dyn trait objects.
        // Raw pointers carry no lifetime; this is safe as long as the pointer
        // remains valid for the duration of StylusEvmApi's use (guaranteed by
        // synchronous WASM execution scoped within the caller).
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
            free_pages,
            page_gas,
            ctx_ptr,
            do_call,
            do_create,
        }
    }

    /// Get a mutable reference to the type-erased journal.
    fn journal(&mut self) -> &mut dyn JournalAccess {
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
            cost = Gas(cost.0.saturating_add(sload_cost).saturating_add(evm_api_gas_to_use.0));

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

        for (key, value) in &dirty {
            let storage_key = U256::from_be_bytes(key.0);
            let storage_value = U256::from_be_bytes(value.0);

            let addr = self.address;
            let info = self.journal().sstore(addr, storage_key, storage_value)?;

            let sstore_cost = sstore_gas_cost(&info);
            if sstore_cost > remaining {
                return Ok((Gas(total_gas), UserOutcomeKind::OutOfInk));
            }
            remaining -= sstore_cost;
            total_gas += sstore_cost;

            // Track SSTORE refunds (EIP-3529).
            self.sstore_refund += sstore_refund(&info);
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
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        if self.read_only && !value.is_zero() {
            self.return_data = Vec::new();
            return Ok((0, Gas(0), UserOutcomeKind::Failure));
        }

        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                ));
            }
        };

        // WasmCallCost equivalent: warm/cold access + value transfer + new account
        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &value, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure));
        }

        // 63/64ths rule
        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        // Stipend for value transfers
        let gas = if !value.is_zero() {
            gas.saturating_add(2300) // CallStipend
        } else {
            gas
        };

        let result = (do_call)(
            self.ctx_ptr,
            0, // CALL
            contract,
            self.address, // caller = current contract
            contract,     // storage_addr = target contract (CALL semantics)
            calldata,
            gas,
            value,
        );

        // Do NOT invalidate the per-call storage cache after a CALL.
        // Nitro's `EvmApiRequestor` (crates/arbutil/src/evm/req.rs) does not
        // invalidate either, and matching that behavior is required to keep
        // gas accounting in lockstep — invalidating clean entries forces the
        // next read of the same slot to pay full SLOAD + EVM_API_INK
        // (~60k gas legacy) instead of the cache-hit cost (0/2/10 gas).
        // CALL targets a different contract anyway, so its storage writes
        // can't affect this contract's cache.

        self.return_data = result.output;
        // cost = baseCost + (gas_given - gas_returned) = baseCost + gas_used
        let cost = base_cost.saturating_add(result.gas_cost);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((self.return_data.len() as u32, Gas(cost), outcome))
    }

    fn delegate_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                ));
            }
        };

        // For DELEGATECALL, no value transfer cost
        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure));
        }

        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        let result = (do_call)(
            self.ctx_ptr,
            1, // DELEGATECALL
            contract,
            self.caller,  // caller = preserved msg.sender
            self.address, // storage_addr = current contract (DELEGATECALL preserves storage context)
            calldata,
            gas,
            self.call_value, // forward current call value
        );

        // Invalidate storage cache: DELEGATECALL sub-call can write to our storage.
        self.storage_cache.slots.clear();

        self.return_data = result.output;
        // cost = baseCost + (gas_given - gas_returned) = baseCost + gas_used
        let cost = base_cost.saturating_add(result.gas_cost);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((self.return_data.len() as u32, Gas(cost), outcome))
    }

    fn static_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        let do_call = match self.do_call {
            Some(f) => f,
            None => {
                self.return_data = b"sub-calls not available".to_vec();
                return Ok((
                    self.return_data.len() as u32,
                    Gas(0),
                    UserOutcomeKind::Failure,
                ));
            }
        };

        let (base_cost, oog) = wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
        if oog {
            self.return_data = Vec::new();
            return Ok((0, Gas(gas_left.0), UserOutcomeKind::Failure));
        }

        let start_gas = gas_left.0.saturating_sub(base_cost) * 63 / 64;
        let gas = start_gas.min(gas_req.0);

        let result = (do_call)(
            self.ctx_ptr,
            2, // STATICCALL
            contract,
            self.address, // caller = current contract
            contract,     // storage_addr = target contract (STATICCALL semantics)
            calldata,
            gas,
            U256::ZERO,
        );

        // Invalidate storage cache: the sub-call may have written to storage
        // visible to us (e.g., via DELEGATECALL back to our address).
        self.storage_cache.slots.clear();

        self.return_data = result.output;
        // cost = baseCost + (gas_given - gas_returned) = baseCost + gas_used
        let cost = base_cost.saturating_add(result.gas_cost);

        let outcome = if result.success {
            UserOutcomeKind::Success
        } else {
            UserOutcomeKind::Failure
        };
        Ok((self.return_data.len() as u32, Gas(cost), outcome))
    }

    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)> {
        if self.read_only {
            self.return_data = Vec::new();
            return Ok((CreateResponse::Fail("write protection".into()), 0, Gas(0)));
        }

        let do_create = match self.do_create {
            Some(f) => f,
            None => {
                self.return_data = b"creates not available".to_vec();
                return Ok((
                    CreateResponse::Fail("not available".into()),
                    self.return_data.len() as u32,
                    Gas(0),
                ));
            }
        };

        // CREATE base cost = 32000
        let base_cost: u64 = 32000;
        if gas.0 < base_cost {
            self.return_data = Vec::new();
            return Ok((CreateResponse::Fail("out of gas".into()), 0, Gas(gas.0)));
        }
        let remaining = gas.0 - base_cost;

        // 63/64ths rule
        let one_64th = remaining / 64;
        let call_gas = remaining - one_64th;

        let result = (do_create)(
            self.ctx_ptr,
            self.address,
            &code,
            call_gas,
            endowment,
            None, // CREATE
        );

        self.return_data = result.output.clone();
        // cost = baseCost + gas_used (Go: startGas - returnGas - one_64th)
        let cost = base_cost.saturating_add(result.gas_cost);

        let response = match result.address {
            Some(addr) => CreateResponse::Success(addr),
            None => {
                // On non-revert failure, clear return data
                if self.return_data.is_empty() {
                    CreateResponse::Fail("create failed".into())
                } else {
                    CreateResponse::Fail("reverted".into())
                }
            }
        };

        Ok((response, self.return_data.len() as u32, Gas(cost)))
    }

    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        salt: B256,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)> {
        if self.read_only {
            self.return_data = Vec::new();
            return Ok((CreateResponse::Fail("write protection".into()), 0, Gas(0)));
        }

        let do_create = match self.do_create {
            Some(f) => f,
            None => {
                self.return_data = b"creates not available".to_vec();
                return Ok((
                    CreateResponse::Fail("not available".into()),
                    self.return_data.len() as u32,
                    Gas(0),
                ));
            }
        };

        // CREATE2 base cost = 32000 + keccak cost
        let keccak_words = (code.len() as u64).div_ceil(32);
        let keccak_cost = keccak_words.saturating_mul(6); // Keccak256WordGas
        let base_cost = 32000u64.saturating_add(keccak_cost);
        if gas.0 < base_cost {
            self.return_data = Vec::new();
            return Ok((CreateResponse::Fail("out of gas".into()), 0, Gas(gas.0)));
        }
        let remaining = gas.0 - base_cost;

        let one_64th = remaining / 64;
        let call_gas = remaining - one_64th;

        let result = (do_create)(
            self.ctx_ptr,
            self.address,
            &code,
            call_gas,
            endowment,
            Some(salt),
        );

        self.return_data = result.output.clone();
        // cost = baseCost + gas_used (Go: startGas - returnGas - one_64th)
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

        Ok((response, self.return_data.len() as u32, Gas(cost)))
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

    fn add_pages(&mut self, new_pages: u16) -> eyre::Result<Gas> {
        // add_stylus_pages returns previous (open, ever) before updating
        let (prev_open, prev_ever) = pages::add_stylus_pages(new_pages);
        let model = MemoryModel::new(self.free_pages, self.page_gas);
        let cost = model.gas_cost(new_pages, prev_open, prev_ever);
        Ok(Gas(cost))
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
