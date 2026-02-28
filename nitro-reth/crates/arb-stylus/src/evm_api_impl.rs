use alloy_primitives::{Address, Log, B256, U256};
use revm::context::journaled_state::entry::JournalEntry;
use revm::context::journal::inner::JournalInner;
use revm::Database;

use crate::evm_api::{CreateResponse, EvmApi, UserOutcomeKind};
use crate::ink::Gas;

/// EIP-2929 gas costs for storage operations.
const COLD_SLOAD_COST: u64 = 2100;
const WARM_STORAGE_READ_COST: u64 = 100;
const COLD_ACCOUNT_ACCESS_COST: u64 = 2600;
const WARM_ACCOUNT_ACCESS_COST: u64 = 100;

/// Concrete [`EvmApi`] bridging WASM host function calls to revm's journaled state.
///
/// Holds raw pointers to the revm journal and database. The pointers remain valid
/// for the lifetime of a single Stylus program execution within a precompile call.
///
/// # Safety
///
/// Wasmer executes WASM programs synchronously on the calling thread, so no
/// cross-thread sharing occurs despite the `Send` bound on [`EvmApi`].
pub struct StylusEvmApi<DB: Database> {
    /// Raw pointer to the journal (contains both JournalInner and DB).
    journal: *mut revm::Journal<DB>,
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
unsafe impl<DB: Database> Send for StylusEvmApi<DB> {}

impl<DB: Database> StylusEvmApi<DB> {
    /// Create a new StylusEvmApi from a raw pointer to the revm Journal.
    ///
    /// # Safety
    ///
    /// The `journal` pointer must remain valid for the lifetime of this struct.
    /// The caller must ensure exclusive mutable access through this pointer.
    pub unsafe fn new(
        journal: *mut revm::Journal<DB>,
        address: Address,
        read_only: bool,
    ) -> Self {
        Self {
            journal,
            address,
            storage_cache: Vec::new(),
            return_data: Vec::new(),
            read_only,
        }
    }

    /// Get mutable references to both the journal inner and the database.
    ///
    /// # Safety
    ///
    /// The raw pointer must be valid and the caller must ensure no aliased references exist.
    fn journal_and_db(&mut self) -> (&mut JournalInner<JournalEntry>, &mut DB) {
        unsafe {
            let journal = &mut *self.journal;
            (&mut journal.inner, &mut journal.database)
        }
    }

    /// Get a mutable reference to the journal inner only.
    fn journal_inner(&mut self) -> &mut JournalInner<JournalEntry> {
        unsafe { &mut (*self.journal).inner }
    }
}

impl<DB: Database> std::fmt::Debug for StylusEvmApi<DB> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusEvmApi")
            .field("address", &self.address)
            .field("read_only", &self.read_only)
            .field("cache_size", &self.storage_cache.len())
            .finish()
    }
}

impl<DB: Database + 'static> EvmApi for StylusEvmApi<DB>
where
    DB::Error: std::fmt::Display + std::fmt::Debug,
{
    fn get_bytes32(&mut self, key: B256, _evm_api_gas_to_use: Gas) -> eyre::Result<(B256, Gas)> {
        let storage_key = U256::from_be_bytes(key.0);
        let addr = self.address;
        let (journal, db) = self.journal_and_db();
        let result = journal
            .sload(db, addr, storage_key, false)
            .map_err(|e| eyre::eyre!("sload failed: {e:?}"))?;

        let value_u256: U256 = result.data;
        let value = B256::from(value_u256.to_be_bytes());
        let gas_cost = if result.is_cold {
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
            let (journal, db) = self.journal_and_db();
            let result = journal
                .sstore(db, addr, storage_key, storage_value, false)
                .map_err(|e| eyre::eyre!("sstore failed: {e:?}"))?;

            // Compute gas cost based on cold/warm and original/present/new values.
            let sstore_cost = sstore_gas_cost(result.is_cold, &result.data);
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
        let journal = self.journal_inner();
        let value = journal.tload(addr, storage_key);
        Ok(B256::from(value.to_be_bytes()))
    }

    fn set_transient_bytes32(&mut self, key: B256, value: B256) -> eyre::Result<UserOutcomeKind> {
        if self.read_only {
            return Ok(UserOutcomeKind::Failure);
        }
        let storage_key = U256::from_be_bytes(key.0);
        let storage_value = U256::from_be_bytes(value.0);
        let addr = self.address;
        let journal = self.journal_inner();
        journal.tstore(addr, storage_key, storage_value);
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
        // Sub-calls from Stylus programs require recursive EVM dispatch.
        // This is wired in Phase 2 when the full execution context is available.
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
        self.return_data = b"Stylus creates not yet wired".into();
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

        // The data layout from Stylus: first `topics * 32` bytes are topic hashes,
        // followed by the log data.
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
        let log = Log::new(
            addr,
            topic_list,
            log_data.into(),
        )
        .expect("too many log topics");

        self.journal_inner().log(log);
        Ok(())
    }

    fn account_balance(&mut self, address: Address) -> eyre::Result<(U256, Gas)> {
        let (journal, db) = self.journal_and_db();
        let result = journal
            .load_account(db, address)
            .map_err(|e| eyre::eyre!("load_account failed: {e:?}"))?;

        let balance = result.data.info.balance;
        let gas_cost = if result.is_cold {
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
        let (journal, db) = self.journal_and_db();
        let result = journal
            .load_code(db, address)
            .map_err(|e| eyre::eyre!("load_code failed: {e:?}"))?;

        let code = result
            .data
            .info
            .code
            .as_ref()
            .map(|c: &revm::bytecode::Bytecode| c.original_bytes().to_vec())
            .unwrap_or_default();
        let gas_cost = if result.is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((code, Gas(gas_cost)))
    }

    fn account_codehash(&mut self, address: Address) -> eyre::Result<(B256, Gas)> {
        let (journal, db) = self.journal_and_db();
        let result = journal
            .load_account(db, address)
            .map_err(|e| eyre::eyre!("load_account failed: {e:?}"))?;

        let hash = result.data.info.code_hash;
        let gas_cost = if result.is_cold {
            COLD_ACCOUNT_ACCESS_COST
        } else {
            WARM_ACCOUNT_ACCESS_COST
        };
        Ok((hash, Gas(gas_cost)))
    }

    fn add_pages(&mut self, _pages: u16) -> eyre::Result<Gas> {
        // Page cost is computed by the caller using the MemoryModel.
        // The thread-local page counter is updated by the dispatch layer.
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

/// Compute SSTORE gas cost from cold/warm status and the slot values.
///
/// Follows EIP-2929 + EIP-3529 (post-London) gas schedule.
fn sstore_gas_cost(
    is_cold: bool,
    result: &revm::context_interface::context::SStoreResult,
) -> u64 {
    // Base cost depends on whether the value is being set, reset, or cleared.
    let base = if result.original_value == result.new_value {
        // No-op store (value unchanged).
        WARM_STORAGE_READ_COST
    } else if result.original_value == result.present_value {
        // Fresh write.
        if result.original_value.is_zero() {
            20_000 // SSTORE_SET_GAS
        } else {
            2_900 // SSTORE_RESET_GAS (5000 - 2100)
        }
    } else {
        // Dirty write (slot already modified in this tx).
        WARM_STORAGE_READ_COST
    };

    let cold_cost = if is_cold { COLD_SLOAD_COST } else { 0 };
    base + cold_cost
}
