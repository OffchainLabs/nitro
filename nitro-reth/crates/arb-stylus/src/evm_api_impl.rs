use alloy_primitives::{Address, Log, B256, U256};
use revm::Database;

use crate::evm_api::{CreateResponse, EvmApi, UserOutcomeKind};
use crate::ink::Gas;

/// EIP-2929 gas costs for storage operations.
const COLD_SLOAD_COST: u64 = 2100;
const WARM_STORAGE_READ_COST: u64 = 100;
const COLD_ACCOUNT_ACCESS_COST: u64 = 2600;
const WARM_ACCOUNT_ACCESS_COST: u64 = 100;

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
}

// ── StylusEvmApi ────────────────────────────────────────────────────

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
    /// Cached storage writes (key, value pairs), flushed on demand.
    storage_cache: Vec<(B256, B256)>,
    /// Return data from the last sub-call.
    return_data: Vec<u8>,
    /// Whether the current execution context is read-only (STATICCALL).
    read_only: bool,
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
    pub unsafe fn new<DB: Database>(
        journal: *mut revm::Journal<DB>,
        address: Address,
        read_only: bool,
    ) -> Self {
        Self {
            journal: journal as *mut dyn JournalAccess,
            address,
            storage_cache: Vec::new(),
            return_data: Vec::new(),
            read_only,
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
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
        _value: U256,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        self.return_data = b"Stylus sub-calls not yet wired".to_vec();
        Ok((
            self.return_data.len() as u32,
            Gas(0),
            UserOutcomeKind::Revert,
        ))
    }

    fn delegate_call(
        &mut self,
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        self.return_data = b"Stylus sub-calls not yet wired".to_vec();
        Ok((
            self.return_data.len() as u32,
            Gas(0),
            UserOutcomeKind::Revert,
        ))
    }

    fn static_call(
        &mut self,
        _contract: Address,
        _calldata: &[u8],
        _gas_left: Gas,
        _gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)> {
        self.return_data = b"Stylus sub-calls not yet wired".to_vec();
        Ok((
            self.return_data.len() as u32,
            Gas(0),
            UserOutcomeKind::Revert,
        ))
    }

    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: U256,
        _gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)> {
        self.return_data = b"Stylus creates not yet wired".to_vec();
        Ok((
            CreateResponse::Fail("not yet wired".into()),
            self.return_data.len() as u32,
            Gas(0),
        ))
    }

    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: U256,
        _salt: B256,
        _gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)> {
        self.return_data = b"Stylus creates not yet wired".to_vec();
        Ok((
            CreateResponse::Fail("not yet wired".into()),
            self.return_data.len() as u32,
            Gas(0),
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
        _gas_left: Gas,
    ) -> eyre::Result<(Vec<u8>, Gas)> {
        let (code, is_cold) = self.journal().account_code(address)?;
        let gas_cost = if is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((code, Gas(gas_cost)))
    }

    fn account_codehash(&mut self, address: Address) -> eyre::Result<(B256, Gas)> {
        let (hash, is_cold) = self.journal().account_codehash(address)?;
        let gas_cost = if is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((hash, Gas(gas_cost)))
    }

    fn add_pages(&mut self, _pages: u16) -> eyre::Result<Gas> {
        Ok(Gas(0))
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
