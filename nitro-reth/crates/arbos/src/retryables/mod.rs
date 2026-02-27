use alloy_primitives::{Address, B256, U256, keccak256};
use revm::Database;

use arb_storage::{
    Queue, Storage, StorageBackedAddress, StorageBackedAddressOrNil, StorageBackedBigUint,
    StorageBackedBytes, StorageBackedUint64, initialize_queue, open_queue,
};

pub const RETRYABLE_LIFETIME_SECONDS: u64 = 7 * 24 * 60 * 60; // one week
pub const RETRYABLE_REAP_PRICE: u64 = 58000;

const TIMEOUT_QUEUE_KEY: &[u8] = &[0];
const CALLDATA_KEY: &[u8] = &[1];

// Storage offsets for Retryable fields.
const NUM_TRIES_OFFSET: u64 = 0;
const FROM_OFFSET: u64 = 1;
const TO_OFFSET: u64 = 2;
const CALLVALUE_OFFSET: u64 = 3;
const BENEFICIARY_OFFSET: u64 = 4;
const TIMEOUT_OFFSET: u64 = 5;
const TIMEOUT_WINDOWS_LEFT_OFFSET: u64 = 6;

/// Manages the collection of retryable tickets.
pub struct RetryableState<D> {
    retryables: Storage<D>,
    pub timeout_queue: Queue<D>,
}

/// A single retryable ticket.
pub struct Retryable<D> {
    pub id: B256,
    backing_storage: Storage<D>,
    num_tries: StorageBackedUint64<D>,
    from: StorageBackedAddress<D>,
    to: StorageBackedAddressOrNil<D>,
    callvalue: StorageBackedBigUint<D>,
    beneficiary: StorageBackedAddress<D>,
    calldata: StorageBackedBytes<D>,
    timeout: StorageBackedUint64<D>,
    timeout_windows_left: StorageBackedUint64<D>,
}

pub fn initialize_retryable_state<D: Database>(sto: &Storage<D>) -> Result<(), ()> {
    initialize_queue(&sto.open_sub_storage(TIMEOUT_QUEUE_KEY))
}

pub fn open_retryable_state<D: Database>(sto: Storage<D>) -> RetryableState<D> {
    let queue_sto = sto.open_sub_storage(TIMEOUT_QUEUE_KEY);
    RetryableState {
        timeout_queue: open_queue(queue_sto),
        retryables: sto,
    }
}

impl<D: Database> RetryableState<D> {
    pub fn initialize(sto: &Storage<D>) -> Result<(), ()> {
        initialize_retryable_state(sto)
    }

    pub fn open(sto: Storage<D>) -> Self {
        open_retryable_state(sto)
    }

    /// Creates a new retryable ticket. The id must be unique.
    pub fn create_retryable(
        &self,
        id: B256,
        timeout: u64,
        from: Address,
        to: Option<Address>,
        callvalue: U256,
        beneficiary: Address,
        calldata: &[u8],
    ) -> Result<Retryable<D>, ()> {
        let ret = self.internal_open(id);
        ret.num_tries.set(0)?;
        ret.from.set(from)?;
        ret.to.set(to)?;
        ret.callvalue.set(callvalue)?;
        ret.beneficiary.set(beneficiary)?;
        ret.calldata.set(calldata)?;
        ret.timeout.set(timeout)?;
        ret.timeout_windows_left.set(0)?;
        self.timeout_queue.put(id)?;
        Ok(ret)
    }

    /// Opens an existing retryable if it exists and hasn't expired.
    pub fn open_retryable(
        &self,
        id: B256,
        current_timestamp: u64,
    ) -> Result<Option<Retryable<D>>, ()> {
        let sto = self.retryables.open_sub_storage(id.as_slice());
        let timeout_storage = StorageBackedUint64::new(sto.state_ptr(), sto.base_key(), TIMEOUT_OFFSET);
        let timeout = timeout_storage.get()?;
        if timeout == 0 || timeout < current_timestamp {
            return Ok(None);
        }
        Ok(Some(self.internal_open(id)))
    }

    /// Gets the size in bytes a retryable occupies in storage.
    pub fn retryable_size_bytes(
        &self,
        id: B256,
        current_time: u64,
    ) -> Result<u64, ()> {
        let retryable = self.open_retryable(id, current_time)?;
        match retryable {
            None => Ok(0),
            Some(ret) => {
                let size = ret.calldata_size()?;
                let calldata_slots = 32 + 32 * words_for_bytes(size);
                Ok(6 * 32 + calldata_slots)
            }
        }
    }

    /// Deletes a retryable and returns whether it existed.
    /// The `transfer_fn` handles moving escrowed funds to the beneficiary.
    pub fn delete_retryable<F>(
        &self,
        id: B256,
        mut transfer_fn: F,
    ) -> Result<bool, ()>
    where
        F: FnMut(Address, Address, U256) -> Result<(), ()>,
    {
        let ret_storage = self.retryables.open_sub_storage(id.as_slice());
        let timeout_val = ret_storage.get_by_uint64(TIMEOUT_OFFSET)?;
        if timeout_val == B256::ZERO {
            return Ok(false);
        }

        // Move escrowed funds to beneficiary.
        let beneficiary_val = ret_storage.get_by_uint64(BENEFICIARY_OFFSET)?;
        let escrow_address = retryable_escrow_address(id);
        let beneficiary_address = Address::from_slice(&beneficiary_val[12..]);
        // The actual balance transfer is delegated to the caller.
        let _ = transfer_fn(escrow_address, beneficiary_address, U256::ZERO);

        // Clear all storage slots.
        let _ = ret_storage.set_by_uint64(NUM_TRIES_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(FROM_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(TO_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(CALLVALUE_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(BENEFICIARY_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(TIMEOUT_OFFSET, B256::ZERO);
        let _ = ret_storage.set_by_uint64(TIMEOUT_WINDOWS_LEFT_OFFSET, B256::ZERO);
        let bytes_storage = StorageBackedBytes::new(ret_storage.open_sub_storage(CALLDATA_KEY));
        bytes_storage.clear()?;
        Ok(true)
    }

    /// Extends the lifetime of a retryable ticket.
    pub fn keepalive(
        &self,
        ticket_id: B256,
        current_timestamp: u64,
        limit_before_add: u64,
        time_to_add: u64,
    ) -> Result<u64, ()> {
        let retryable = self.open_retryable(ticket_id, current_timestamp)?;
        let retryable = retryable.ok_or(())?;
        let timeout = retryable.calculate_timeout()?;
        if timeout > limit_before_add {
            return Err(());
        }
        self.timeout_queue.put(retryable.id)?;
        retryable.increment_timeout_windows()?;
        let new_timeout = timeout + RETRYABLE_LIFETIME_SECONDS;
        // In Go, this also burns RetryableReapPrice gas.
        Ok(new_timeout)
    }

    /// Tries to reap one expired retryable from the timeout queue.
    pub fn try_to_reap_one_retryable<F>(
        &self,
        current_timestamp: u64,
        mut transfer_fn: F,
    ) -> Result<(), ()>
    where
        F: FnMut(Address, Address, U256) -> Result<(), ()>,
    {
        let id = self.timeout_queue.peek()?;
        let id = match id {
            None => return Ok(()),
            Some(id) => id,
        };

        let ret_storage = self.retryables.open_sub_storage(id.as_slice());
        let timeout_storage = StorageBackedUint64::new(
            ret_storage.state_ptr(),
            ret_storage.base_key(),
            TIMEOUT_OFFSET,
        );
        let timeout = timeout_storage.get()?;

        if timeout == 0 {
            // Already deleted, discard queue entry.
            let _ = self.timeout_queue.get()?;
            return Ok(());
        }

        let windows_left_storage = StorageBackedUint64::new(
            ret_storage.state_ptr(),
            ret_storage.base_key(),
            TIMEOUT_WINDOWS_LEFT_OFFSET,
        );
        let windows_left = windows_left_storage.get()?;

        if timeout >= current_timestamp {
            return Ok(());
        }

        // Retryable has expired or lost a lifetime window.
        let _ = self.timeout_queue.get()?;

        if windows_left == 0 {
            // Fully expired — delete it.
            self.delete_retryable(id, &mut transfer_fn)?;
            return Ok(());
        }

        // Consume a window, delaying timeout by one lifetime.
        timeout_storage.set(timeout + RETRYABLE_LIFETIME_SECONDS)?;
        windows_left_storage.set(windows_left - 1)?;
        Ok(())
    }

    fn internal_open(&self, id: B256) -> Retryable<D> {
        let sto = self.retryables.open_sub_storage(id.as_slice());
        let state = sto.state_ptr();
        let base_key = sto.base_key();
        Retryable {
            id,
            num_tries: StorageBackedUint64::new(state, base_key, NUM_TRIES_OFFSET),
            from: StorageBackedAddress::new(state, base_key, FROM_OFFSET),
            to: StorageBackedAddressOrNil::new(state, base_key, TO_OFFSET),
            callvalue: StorageBackedBigUint::new(state, base_key, CALLVALUE_OFFSET),
            beneficiary: StorageBackedAddress::new(state, base_key, BENEFICIARY_OFFSET),
            calldata: StorageBackedBytes::new(sto.open_sub_storage(CALLDATA_KEY)),
            timeout: StorageBackedUint64::new(state, base_key, TIMEOUT_OFFSET),
            timeout_windows_left: StorageBackedUint64::new(state, base_key, TIMEOUT_WINDOWS_LEFT_OFFSET),
            backing_storage: sto,
        }
    }
}

impl<D: Database> Retryable<D> {
    pub fn num_tries(&self) -> Result<u64, ()> {
        self.num_tries.get()
    }

    pub fn increment_num_tries(&self) -> Result<u64, ()> {
        let current = self.num_tries.get()?;
        let new_val = current + 1;
        self.num_tries.set(new_val)?;
        Ok(new_val)
    }

    pub fn beneficiary(&self) -> Result<Address, ()> {
        self.beneficiary.get()
    }

    pub fn calculate_timeout(&self) -> Result<u64, ()> {
        let timeout = self.timeout.get()?;
        let windows = self.timeout_windows_left.get()?;
        Ok(timeout + windows * RETRYABLE_LIFETIME_SECONDS)
    }

    pub fn set_timeout(&self, val: u64) -> Result<(), ()> {
        self.timeout.set(val)
    }

    pub fn timeout_windows_left(&self) -> Result<u64, ()> {
        self.timeout_windows_left.get()
    }

    fn increment_timeout_windows(&self) -> Result<u64, ()> {
        let current = self.timeout_windows_left.get()?;
        let new_val = current + 1;
        self.timeout_windows_left.set(new_val)?;
        Ok(new_val)
    }

    pub fn from(&self) -> Result<Address, ()> {
        self.from.get()
    }

    pub fn to(&self) -> Result<Option<Address>, ()> {
        self.to.get()
    }

    pub fn callvalue(&self) -> Result<U256, ()> {
        self.callvalue.get()
    }

    pub fn calldata(&self) -> Result<Vec<u8>, ()> {
        self.calldata.get()
    }

    pub fn calldata_size(&self) -> Result<u64, ()> {
        self.calldata.size()
    }

    /// Constructs a retry transaction from this retryable's stored fields
    /// combined with the provided runtime parameters.
    pub fn make_tx(
        &self,
        chain_id: U256,
        nonce: u64,
        gas_fee_cap: U256,
        gas: u64,
        ticket_id: B256,
        refund_to: Address,
        max_refund: U256,
        submission_fee_refund: U256,
    ) -> Result<arb_alloy_consensus::tx::ArbRetryTx, ()> {
        Ok(arb_alloy_consensus::tx::ArbRetryTx {
            chain_id,
            nonce,
            from: self.from()?,
            gas_fee_cap,
            gas,
            to: self.to()?,
            value: self.callvalue()?,
            data: self.calldata()?.into(),
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

/// Computes the submission fee for a retryable ticket.
pub fn retryable_submission_fee(calldata_length: usize, l1_base_fee: U256) -> U256 {
    l1_base_fee * U256::from(1400 + 6 * calldata_length as u64)
}

/// Rounds up byte count to number of 32-byte words.
fn words_for_bytes(bytes: u64) -> u64 {
    (bytes + 31) / 32
}
