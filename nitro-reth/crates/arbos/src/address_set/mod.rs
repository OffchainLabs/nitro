use alloy_primitives::{Address, B256, U256};
use revm::Database;

use arb_storage::{
    Storage, StorageBackedAddress, StorageBackedUint64, StorageBackend, SystemStateBackend,
};

mod error;
pub use error::AddressSetError;

/// Flat ArbOS storage gas per slot read/write (no EVM cold/warm or refunds).
const STORAGE_READ_COST: u64 = 800;
const STORAGE_WRITE_COST: u64 = 20_000;
const STORAGE_WRITE_ZERO_COST: u64 = 5_000;

fn write_cost(value: B256) -> u64 {
    if value == B256::ZERO {
        STORAGE_WRITE_ZERO_COST
    } else {
        STORAGE_WRITE_COST
    }
}

/// A set of addresses backed by ArbOS storage.
///
/// Layout: slot 0 = size, slots 1..size = addresses (as StorageBackedAddress).
/// Sub-storage at key ]0\] maps address_hash → slot index.
pub struct AddressSet<'a, D> {
    backing_storage: Storage<'a, D>,
    size: StorageBackedUint64,
    by_address: Storage<'a, D>,
}

pub fn initialize_address_set<D: Database>(sto: &Storage<'_, D>) -> Result<(), AddressSetError> {
    Ok(sto.set_by_uint64(0, B256::ZERO)?)
}

pub fn open_address_set<D>(sto: Storage<'_, D>) -> AddressSet<'_, D> {
    let size = StorageBackedUint64::new(sto.base_key(), 0);
    let by_address = sto.open_sub_storage(&[0u8]);
    AddressSet {
        backing_storage: sto,
        size,
        by_address,
    }
}

impl<D> AddressSet<'_, D> {
    pub fn size<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u64, AddressSetError> {
        Ok(self.size.get(backend)?)
    }

    pub fn is_member<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<bool, AddressSetError> {
        let value = self.by_address_get(backend, address_to_hash(addr))?;
        Ok(value != B256::ZERO)
    }

    pub fn get_any_member<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Option<Address>, AddressSetError> {
        let size = self.size.get(backend)?;
        if size == 0 {
            return Ok(None);
        }
        let sba = StorageBackedAddress::new(self.backing_storage.base_key(), 1);
        Ok(sba.get(backend).map(Some)?)
    }

    pub fn clear<B: StorageBackend>(&self, backend: &mut B) -> Result<(), AddressSetError> {
        let size = self.size.get(backend)?;
        if size == 0 {
            return Ok(());
        }
        for i in 1..=size {
            let contents = self.backing_get_by_uint64(backend, i)?;
            self.backing_set_by_uint64(backend, i, B256::ZERO)?;
            self.by_address_set(backend, contents, B256::ZERO)?;
        }
        Ok(self.size.set(backend, 0)?)
    }

    pub fn all_members<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        max_num: u64,
    ) -> Result<Vec<Address>, AddressSetError> {
        let mut size = self.size.get(backend)?;
        if size > max_num {
            size = max_num;
        }
        let mut ret = Vec::with_capacity(size as usize);
        for i in 0..size {
            let sba = StorageBackedAddress::new(self.backing_storage.base_key(), i + 1);
            ret.push(sba.get(backend)?);
        }
        Ok(ret)
    }

    pub fn clear_list<B: StorageBackend>(&self, backend: &mut B) -> Result<(), AddressSetError> {
        let size = self.size.get(backend)?;
        if size == 0 {
            return Ok(());
        }
        for i in 1..=size {
            self.backing_set_by_uint64(backend, i, B256::ZERO)?;
        }
        Ok(self.size.set(backend, 0)?)
    }

    pub fn rectify_mapping<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(), AddressSetError> {
        if !self.is_member(backend, addr)? {
            return Err(AddressSetError::NotMember);
        }

        let addr_as_hash = address_to_hash(addr);
        let slot = hash_to_uint64(self.by_address_get(backend, addr_as_hash)?);
        let at_slot = self.backing_get_by_uint64(backend, slot)?;
        let size = self.size.get(backend)?;

        if at_slot == addr_as_hash && slot <= size {
            return Err(AddressSetError::MappingAlreadyConsistent);
        }

        self.by_address_set(backend, addr_as_hash, B256::ZERO)?;
        self.add(backend, addr)
    }

    pub fn add<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(), AddressSetError> {
        let present = self.is_member(backend, addr)?;
        if present {
            return Ok(());
        }

        let size = self.size.get(backend)?;
        let slot = uint_to_hash(1 + size);
        let addr_as_hash = address_to_hash(addr);

        self.by_address_set(backend, addr_as_hash, slot)?;

        let sba = StorageBackedAddress::new(self.backing_storage.base_key(), 1 + size);
        sba.set(backend, addr)?;

        Ok(self.size.set(backend, size + 1)?)
    }

    /// Removes `addr`, adding the value-dependent storage gas it consumes to `gas`.
    pub fn remove<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
        arbos_version: u64,
        gas: &mut u64,
    ) -> Result<(), AddressSetError> {
        let addr_as_hash = address_to_hash(addr);
        let slot_hash = self.by_address_get(backend, addr_as_hash)?;
        *gas += STORAGE_READ_COST;
        let slot = hash_to_uint64(slot_hash);

        if slot == 0 {
            return Ok(());
        }

        self.by_address_set(backend, addr_as_hash, B256::ZERO)?;
        *gas += STORAGE_WRITE_ZERO_COST;

        let size = self.size.get(backend)?;
        *gas += STORAGE_READ_COST;
        if slot < size {
            let at_size = self.backing_get_by_uint64(backend, size)?;
            *gas += STORAGE_READ_COST;
            self.backing_set_by_uint64(backend, slot, at_size)?;
            *gas += write_cost(at_size);

            if arbos_version >= 11 {
                self.by_address_set(backend, at_size, uint_to_hash(slot))?;
                *gas += write_cost(uint_to_hash(slot));
            }
        }

        self.backing_set_by_uint64(backend, size, B256::ZERO)?;
        *gas += STORAGE_WRITE_ZERO_COST;

        let new_size = size - 1;
        *gas += STORAGE_READ_COST;
        self.size.set(backend, new_size)?;
        *gas += write_cost(uint_to_hash(new_size));
        Ok(())
    }

    fn by_address_get<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        key: B256,
    ) -> Result<B256, AddressSetError> {
        let slot = self.by_address.slot_for_key(key);
        let value = backend
            .sload_system(self.by_address.account(), slot)
            .map_err(Into::into)?;
        Ok(B256::from(value.to_be_bytes::<32>()))
    }

    fn by_address_set<B: StorageBackend>(
        &self,
        backend: &mut B,
        key: B256,
        value: B256,
    ) -> Result<(), AddressSetError> {
        let slot = self.by_address.slot_for_key(key);
        backend
            .sstore(
                self.by_address.account(),
                slot,
                U256::from_be_bytes(value.0),
            )
            .map_err(Into::into)?;
        Ok(())
    }

    fn backing_get_by_uint64<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        offset: u64,
    ) -> Result<B256, AddressSetError> {
        let slot = self.backing_storage.new_slot(offset);
        let value = backend
            .sload_system(self.backing_storage.account(), slot)
            .map_err(Into::into)?;
        Ok(B256::from(value.to_be_bytes::<32>()))
    }

    fn backing_set_by_uint64<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
        value: B256,
    ) -> Result<(), AddressSetError> {
        let slot = self.backing_storage.new_slot(offset);
        backend
            .sstore(
                self.backing_storage.account(),
                slot,
                U256::from_be_bytes(value.0),
            )
            .map_err(Into::into)?;
        Ok(())
    }
}

fn address_to_hash(addr: Address) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[12..32].copy_from_slice(addr.as_slice());
    B256::from(bytes)
}

fn uint_to_hash(val: u64) -> B256 {
    B256::from(U256::from(val))
}

fn hash_to_uint64(hash: B256) -> u64 {
    U256::from_be_bytes(hash.0).to::<u64>()
}
