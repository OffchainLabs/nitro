use alloy_primitives::{Address, Log, B256, U256};
use arbos::programs::memory::MemoryModel;
use revm::Database;

use crate::evm_api::{CreateResponse, EvmApi, UserOutcomeKind};
use crate::ink::Gas;
use crate::pages;

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
        self.inner.state.contains_key(&addr)
            || self.inner.warm_addresses.is_warm(&addr)
    }

    fn add_address_to_access_list(&mut self, addr: Address) {
        // Load the account to mark it warm in the state map.
        let _ = self
            .inner
            .load_account(&mut self.database, addr);
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
/// Parameters: (ctx_ptr, call_type, contract, caller, input, gas, value)
/// call_type: 0=CALL, 1=DELEGATECALL, 2=STATICCALL
pub type DoCallFn =
    fn(*mut (), u8, Address, Address, &[u8], u64, U256) -> SubCallResult;

/// Type-erased function pointer for executing CREATE/CREATE2 from Stylus.
///
/// Parameters: (ctx_ptr, caller, code, gas, endowment, salt)
/// salt=None for CREATE, Some for CREATE2.
pub type DoCreateFn =
    fn(*mut (), Address, &[u8], u64, U256, Option<B256>) -> SubCreateResult;

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
    /// Cached storage writes (key, value pairs), flushed on demand.
    storage_cache: Vec<(B256, B256)>,
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
        Self {
            journal: journal as *mut dyn JournalAccess,
            address,
            caller,
            call_value,
            storage_cache: Vec::new(),
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
}

impl std::fmt::Debug for StylusEvmApi {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusEvmApi")
            .field("address", &self.address)
            .field("read_only", &self.read_only)
            .field("cache_size", &self.storage_cache.len())
            .finish()
    }
}

impl EvmApi for StylusEvmApi {
    fn get_bytes32(&mut self, key: B256, _evm_api_gas_to_use: Gas) -> eyre::Result<(B256, Gas)> {
        let storage_key = U256::from_be_bytes(key.0);
        let addr = self.address;
        let (value_u256, is_cold) = self.journal().sload(addr, storage_key)?;
        let value = B256::from(value_u256.to_be_bytes());
        let gas_cost = if is_cold {
            COLD_SLOAD_COST
        } else {
            WARM_STORAGE_READ_COST
        };
        Ok((value, Gas(gas_cost)))
    }

    fn cache_bytes32(&mut self, key: B256, value: B256) -> eyre::Result<Gas> {
        self.storage_cache.push((key, value));
        Ok(Gas(0))
    }

    fn flush_storage_cache(
        &mut self,
        clear: bool,
        gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)> {
        let entries: Vec<(B256, B256)> = if clear {
            std::mem::take(&mut self.storage_cache)
        } else {
            self.storage_cache.clone()
        };

        if self.read_only && !entries.is_empty() {
            return Ok((Gas(0), UserOutcomeKind::Failure));
        }

        let mut total_gas = 0u64;
        let mut remaining = gas_left.0;

        for (key, value) in &entries {
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
                return Ok((self.return_data.len() as u32, Gas(0), UserOutcomeKind::Failure));
            }
        };

        // WasmCallCost equivalent: warm/cold access + value transfer + new account
        let (base_cost, oog) =
            wasm_call_cost(self.journal(), contract, &value, gas_left.0);
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
            calldata,
            gas,
            value,
        );

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
                return Ok((self.return_data.len() as u32, Gas(0), UserOutcomeKind::Failure));
            }
        };

        // For DELEGATECALL, no value transfer cost
        let (base_cost, oog) =
            wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
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
            self.caller, // original caller
            calldata,
            gas,
            self.call_value, // forward current call value
        );

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
                return Ok((self.return_data.len() as u32, Gas(0), UserOutcomeKind::Failure));
            }
        };

        let (base_cost, oog) =
            wasm_call_cost(self.journal(), contract, &U256::ZERO, gas_left.0);
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
            self.address,
            calldata,
            gas,
            U256::ZERO,
        );

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
        let keccak_words = (code.len() as u64 + 31) / 32;
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
