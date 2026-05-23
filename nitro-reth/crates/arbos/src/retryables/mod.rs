use alloy_primitives::{keccak256, Address, B256, U256};
use revm::Database;

use arb_storage::{
    initialize_queue, open_queue, Queue, Storage, StorageBackedAddress, StorageBackedAddressOrNil,
    StorageBackedBigUint, StorageBackedBytes, StorageBackedUint64, StorageBackend,
};

mod error;
pub use error::RetryableError;

pub const RETRYABLE_LIFETIME_SECONDS: u64 = 7 * 24 * 60 * 60; // one week
pub const RETRYABLE_REAP_PRICE: u64 = 58000;

pub const TIMEOUT_QUEUE_KEY: &[u8] = &[0];
pub const CALLDATA_KEY: &[u8] = &[1];

// Storage offsets for Retryable fields.
pub const NUM_TRIES_OFFSET: u64 = 0;
pub const FROM_OFFSET: u64 = 1;
pub const TO_OFFSET: u64 = 2;
pub const CALLVALUE_OFFSET: u64 = 3;
pub const BENEFICIARY_OFFSET: u64 = 4;
pub const TIMEOUT_OFFSET: u64 = 5;
pub const TIMEOUT_WINDOWS_LEFT_OFFSET: u64 = 6;

/// Manages the collection of retryable tickets.
pub struct RetryableState<D> {
    retryables: Storage<D>,
    pub timeout_queue: Queue,
}

/// A single retryable ticket.
pub struct Retryable<D> {
    pub id: B256,
    #[allow(dead_code)]
    backing_storage: Storage<D>,
    num_tries: StorageBackedUint64,
    from: StorageBackedAddress,
    to: StorageBackedAddressOrNil,
    callvalue: StorageBackedBigUint,
    beneficiary: StorageBackedAddress,
    calldata: StorageBackedBytes,
    timeout: StorageBackedUint64,
    timeout_windows_left: StorageBackedUint64,
}

pub fn initialize_retryable_state<D: Database>(sto: &Storage<D>) -> Result<(), RetryableError> {
    Ok(initialize_queue(&sto.open_sub_storage(TIMEOUT_QUEUE_KEY))?)
}

pub fn open_retryable_state<D>(sto: Storage<D>) -> RetryableState<D> {
    let queue_sto = sto.open_sub_storage(TIMEOUT_QUEUE_KEY);
    RetryableState {
        timeout_queue: open_queue(queue_sto),
        retryables: sto,
    }
}

impl<D> RetryableState<D> {
    pub fn open(sto: Storage<D>) -> Self {
        open_retryable_state(sto)
    }
}

impl<D: Database> RetryableState<D> {
    pub fn initialize(sto: &Storage<D>) -> Result<(), RetryableError> {
        initialize_retryable_state(sto)
    }

    /// Creates a new retryable ticket. The id must be unique.
    pub fn create_retryable<B: StorageBackend>(
        &self,
        backend: &mut B,
        id: B256,
        timeout: u64,
        from: Address,
        to: Option<Address>,
        callvalue: U256,
        beneficiary: Address,
        calldata: &[u8],
    ) -> Result<Retryable<D>, RetryableError> {
        let ret = self.internal_open(id);
        ret.num_tries.set(backend, 0)?;
        ret.from.set(backend, from)?;
        ret.to.set(backend, to)?;
        ret.callvalue.set(backend, callvalue)?;
        ret.beneficiary.set(backend, beneficiary)?;
        ret.calldata.set(backend, calldata)?;
        ret.timeout.set(backend, timeout)?;
        ret.timeout_windows_left.set(backend, 0)?;
        self.timeout_queue.put(backend, id)?;
        Ok(ret)
    }

    /// Opens an existing retryable if it exists and hasn't expired.
    pub fn open_retryable<B: StorageBackend>(
        &self,
        backend: &mut B,
        id: B256,
        current_timestamp: u64,
    ) -> Result<Option<Retryable<D>>, RetryableError> {
        let sto = self.retryables.open_sub_storage(id.as_slice());
        let timeout_storage = StorageBackedUint64::new(sto.base_key(), TIMEOUT_OFFSET);
        let timeout = timeout_storage.get(backend)?;
        if timeout == 0 || timeout < current_timestamp {
            return Ok(None);
        }
        Ok(Some(self.internal_open(id)))
    }

    /// Gets the size in bytes a retryable occupies in storage.
    pub fn retryable_size_bytes<B: StorageBackend>(
        &self,
        backend: &mut B,
        id: B256,
        current_time: u64,
    ) -> Result<u64, RetryableError> {
        let retryable = self.open_retryable(backend, id, current_time)?;
        match retryable {
            None => Ok(0),
            Some(ret) => {
                let size = ret.calldata_size(backend)?;
                let calldata_slots = 32 + 32 * words_for_bytes(size);
                Ok(6 * 32 + calldata_slots)
            }
        }
    }

    /// Deletes a retryable and returns whether it existed.
    /// Moves the escrow's entire balance to the beneficiary via the provided closures.
    pub fn delete_retryable<F, G, B>(
        &self,
        backend: &mut B,
        id: B256,
        mut transfer_fn: F,
        mut balance_of: G,
    ) -> Result<bool, RetryableError>
    where
        F: FnMut(Address, Address, U256) -> Result<(), ()>,
        G: FnMut(Address) -> U256,
        B: StorageBackend,
    {
        let ret_storage = self.retryables.open_sub_storage(id.as_slice());
        let timeout_val = ret_storage.get_by_uint64(TIMEOUT_OFFSET)?;
        if timeout_val == B256::ZERO {
            return Ok(false);
        }

        let beneficiary_val = ret_storage.get_by_uint64(BENEFICIARY_OFFSET)?;
        let escrow_address = retryable_escrow_address(id);
        let beneficiary_address = Address::from_slice(&beneficiary_val[12..]);
        let amount = balance_of(escrow_address);
        transfer_fn(escrow_address, beneficiary_address, amount)
            .map_err(|()| RetryableError::TransferFailed)?;

        ret_storage.set_by_uint64(NUM_TRIES_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(FROM_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(TO_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(CALLVALUE_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(BENEFICIARY_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(TIMEOUT_OFFSET, B256::ZERO)?;
        ret_storage.set_by_uint64(TIMEOUT_WINDOWS_LEFT_OFFSET, B256::ZERO)?;
        let bytes_storage =
            StorageBackedBytes::new(ret_storage.open_sub_storage(CALLDATA_KEY).base_key());
        bytes_storage.clear(backend)?;
        Ok(true)
    }

    /// Extends the lifetime of a retryable ticket.
    pub fn keepalive<B: StorageBackend>(
        &self,
        backend: &mut B,
        ticket_id: B256,
        current_timestamp: u64,
        limit_before_add: u64,
        _time_to_add: u64,
    ) -> Result<u64, RetryableError> {
        let retryable = self
            .open_retryable(backend, ticket_id, current_timestamp)?
            .ok_or(RetryableError::NoTicketWithId)?;
        let timeout = retryable.calculate_timeout(backend)?;
        if timeout > limit_before_add {
            return Err(RetryableError::TimeoutTooFarFuture);
        }
        self.timeout_queue.put(backend, retryable.id)?;
        retryable.increment_timeout_windows(backend)?;
        let new_timeout = timeout + RETRYABLE_LIFETIME_SECONDS;
        Ok(new_timeout)
    }

    /// Tries to reap one expired retryable from the timeout queue.
    pub fn try_to_reap_one_retryable<F, G, B>(
        &self,
        backend: &mut B,
        current_timestamp: u64,
        mut transfer_fn: F,
        mut balance_of: G,
    ) -> Result<(), RetryableError>
    where
        F: FnMut(Address, Address, U256) -> Result<(), ()>,
        G: FnMut(Address) -> U256,
        B: StorageBackend,
    {
        let id = self.timeout_queue.peek(backend)?;
        let id = match id {
            None => return Ok(()),
            Some(id) => id,
        };

        let ret_storage = self.retryables.open_sub_storage(id.as_slice());
        let timeout_storage = StorageBackedUint64::new(ret_storage.base_key(), TIMEOUT_OFFSET);
        let timeout = timeout_storage.get(backend)?;

        if timeout == 0 {
            self.timeout_queue.get(backend)?;
            return Ok(());
        }

        let windows_left_storage =
            StorageBackedUint64::new(ret_storage.base_key(), TIMEOUT_WINDOWS_LEFT_OFFSET);
        let windows_left = windows_left_storage.get(backend)?;

        if timeout >= current_timestamp {
            return Ok(());
        }

        self.timeout_queue.get(backend)?;

        if windows_left == 0 {
            self.delete_retryable(backend, id, &mut transfer_fn, &mut balance_of)?;
            return Ok(());
        }

        timeout_storage.set(backend, timeout + RETRYABLE_LIFETIME_SECONDS)?;
        windows_left_storage.set(backend, windows_left - 1)?;
        Ok(())
    }

    /// Total number of pending retryables in the timeout queue.
    pub fn queue_size<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, RetryableError> {
        Ok(self.timeout_queue.size(backend)?)
    }

    /// Walk the timeout queue and yield `(ticket_id, timeout_seconds)`
    /// for each non-expired retryable.
    pub fn snapshot_queue<B: StorageBackend>(
        &self,
        backend: &mut B,
        current_time: u64,
        max_entries: usize,
    ) -> Result<Vec<(B256, u64)>, RetryableError> {
        let ids: Vec<B256> = {
            let mut collected = Vec::new();
            self.timeout_queue
                .for_each(backend, |id| -> Result<(), RetryableError> {
                    collected.push(id);
                    Ok(())
                })?;
            collected
        };
        let mut out = Vec::new();
        for id in ids {
            if out.len() >= max_entries {
                break;
            }
            if let Some(retryable) = self.open_retryable(backend, id, current_time)? {
                let timeout = retryable.calculate_timeout(backend)?;
                out.push((id, timeout));
            }
        }
        Ok(out)
    }

    fn internal_open(&self, id: B256) -> Retryable<D> {
        let sto = self.retryables.open_sub_storage(id.as_slice());
        let base_key = sto.base_key();
        let calldata_key = sto.open_sub_storage(CALLDATA_KEY).base_key();
        Retryable {
            id,
            num_tries: StorageBackedUint64::new(base_key, NUM_TRIES_OFFSET),
            from: StorageBackedAddress::new(base_key, FROM_OFFSET),
            to: StorageBackedAddressOrNil::new(base_key, TO_OFFSET),
            callvalue: StorageBackedBigUint::new(base_key, CALLVALUE_OFFSET),
            beneficiary: StorageBackedAddress::new(base_key, BENEFICIARY_OFFSET),
            calldata: StorageBackedBytes::new(calldata_key),
            timeout: StorageBackedUint64::new(base_key, TIMEOUT_OFFSET),
            timeout_windows_left: StorageBackedUint64::new(base_key, TIMEOUT_WINDOWS_LEFT_OFFSET),
            backing_storage: sto,
        }
    }
}

impl<D: Database> Retryable<D> {
    pub fn num_tries<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, RetryableError> {
        Ok(self.num_tries.get(backend)?)
    }

    pub fn increment_num_tries<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, RetryableError> {
        let current = self.num_tries.get(backend)?;
        let new_val = current + 1;
        self.num_tries.set(backend, new_val)?;
        Ok(new_val)
    }

    pub fn beneficiary<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Address, RetryableError> {
        Ok(self.beneficiary.get(backend)?)
    }

    pub fn calculate_timeout<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, RetryableError> {
        let timeout = self.timeout.get(backend)?;
        let windows = self.timeout_windows_left.get(backend)?;
        Ok(timeout + windows * RETRYABLE_LIFETIME_SECONDS)
    }

    pub fn set_timeout<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), RetryableError> {
        Ok(self.timeout.set(backend, val)?)
    }

    pub fn timeout_windows_left<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, RetryableError> {
        Ok(self.timeout_windows_left.get(backend)?)
    }

    fn increment_timeout_windows<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, RetryableError> {
        let current = self.timeout_windows_left.get(backend)?;
        let new_val = current + 1;
        self.timeout_windows_left.set(backend, new_val)?;
        Ok(new_val)
    }

    pub fn from<B: StorageBackend>(&self, backend: &mut B) -> Result<Address, RetryableError> {
        Ok(self.from.get(backend)?)
    }

    pub fn to<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Option<Address>, RetryableError> {
        Ok(self.to.get(backend)?)
    }

    pub fn callvalue<B: StorageBackend>(&self, backend: &mut B) -> Result<U256, RetryableError> {
        Ok(self.callvalue.get(backend)?)
    }

    pub fn calldata<B: StorageBackend>(&self, backend: &mut B) -> Result<Vec<u8>, RetryableError> {
        Ok(self.calldata.get(backend)?)
    }

    pub fn calldata_size<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, RetryableError> {
        Ok(self.calldata.size(backend)?)
    }

    /// Constructs a retry transaction from this retryable's stored fields
    /// combined with the provided runtime parameters.
    pub fn make_tx<B: StorageBackend>(
        &self,
        backend: &mut B,
        chain_id: U256,
        nonce: u64,
        gas_fee_cap: U256,
        gas: u64,
        ticket_id: B256,
        refund_to: Address,
        max_refund: U256,
        submission_fee_refund: U256,
    ) -> Result<arb_alloy_consensus::tx::ArbRetryTx, RetryableError> {
        Ok(arb_alloy_consensus::tx::ArbRetryTx {
            chain_id,
            nonce,
            from: self.from(backend)?,
            gas_fee_cap,
            gas,
            to: self.to(backend)?,
            value: self.callvalue(backend)?,
            data: self.calldata(backend)?.into(),
            ticket_id,
            refund_to,
            max_refund,
            submission_fee_refund,
        })
    }
}

/// Computes the escrow address for a retryable ticket.
pub fn retryable_escrow_address(ticket_id: B256) -> Address {
    let mut data = Vec::with_capacity(16 + 32);
    data.extend_from_slice(b"retryable escrow");
    data.extend_from_slice(ticket_id.as_slice());
    let hash = keccak256(&data);
    Address::from_slice(&hash[12..])
}

/// Submission fee for a retryable ticket: `(1400 + 6 * len) * l1_base_fee`,
/// computed with big-integer arithmetic to prevent overflow.
pub fn retryable_submission_fee(calldata_length: usize, l1_base_fee: U256) -> U256 {
    let factor = U256::from(1400u64)
        .saturating_add(U256::from(6u64).saturating_mul(U256::from(calldata_length as u128)));
    l1_base_fee.saturating_mul(factor)
}

/// Rounds up byte count to number of 32-byte words.
fn words_for_bytes(bytes: u64) -> u64 {
    bytes.div_ceil(32)
}
