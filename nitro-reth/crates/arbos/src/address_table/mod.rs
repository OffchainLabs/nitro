use alloy_primitives::{Address, B256, U256};
use alloy_rlp::{Decodable, Encodable};
use revm::Database;

use arb_storage::{Storage, StorageBackedUint64, StorageBackend};

mod error;
pub use error::AddressTableError;

/// A mapping between addresses and compact integer indices.
///
/// Allows compressing addresses to small integers for more efficient on-chain encoding.
/// Slot 0 = number of items, slots 1..N = address hashes.
/// Sub-storage at key [] maps address_hash → 1-based index.
pub struct AddressTable<D> {
    backing_storage: Storage<D>,
    by_address: Storage<D>,
    num_items: StorageBackedUint64,
}

pub fn initialize_address_table<D: Database>(_sto: &Storage<D>) {
    // no-op
}

pub fn open_address_table<D>(sto: Storage<D>) -> AddressTable<D> {
    let num_items = StorageBackedUint64::new(sto.base_key(), 0);
    let by_address = sto.open_sub_storage(&[]);
    AddressTable {
        backing_storage: sto,
        by_address,
        num_items,
    }
}

impl<D> AddressTable<D> {
    /// Registers `addr` if not already present and returns its 0-based index
    /// together with a flag indicating whether the address was already
    /// registered.
    pub fn register<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(u64, bool), AddressTableError> {
        let addr_hash = address_to_hash(addr);
        let rev = self.by_address_get(backend, addr_hash)?;

        if rev != B256::ZERO {
            return Ok((U256::from_be_bytes(rev.0).to::<u64>() - 1, true));
        }

        let current = self.num_items.get(backend)?;
        let new_num_items = current + 1;
        self.num_items.set(backend, new_num_items)?;

        self.backing_set_by_uint64(backend, new_num_items, addr_hash)?;
        self.by_address_set(backend, addr_hash, uint_to_hash(new_num_items))?;

        Ok((new_num_items - 1, false))
    }

    pub fn lookup<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(u64, bool), AddressTableError> {
        let addr_hash = address_to_hash(addr);
        let res_hash = self.by_address_get(backend, addr_hash)?;
        let res = U256::from_be_bytes(res_hash.0).to::<u64>();

        if res == 0 {
            Ok((0, false))
        } else {
            Ok((res - 1, true))
        }
    }

    pub fn address_exists<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<bool, AddressTableError> {
        let (_, exists) = self.lookup(backend, addr)?;
        Ok(exists)
    }

    pub fn size<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, AddressTableError> {
        Ok(self.num_items.get(backend)?)
    }

    pub fn lookup_index<B: StorageBackend>(
        &self,
        backend: &mut B,
        index: u64,
    ) -> Result<Option<Address>, AddressTableError> {
        let items = self.num_items.get(backend)?;
        if index >= items {
            return Ok(None);
        }
        let value = self.backing_get_by_uint64(backend, index + 1)?;
        let mut addr_bytes = [0u8; 20];
        addr_bytes.copy_from_slice(&value.0[12..32]);
        Ok(Some(Address::from(addr_bytes)))
    }

    /// Compress an address into an RLP-encoded index or raw address bytes.
    pub fn compress<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<Vec<u8>, AddressTableError> {
        let (index, exists) = self.lookup(backend, addr)?;
        if exists {
            let mut buf = Vec::new();
            index.encode(&mut buf);
            Ok(buf)
        } else {
            let mut buf = Vec::new();
            addr.as_slice().encode(&mut buf);
            Ok(buf)
        }
    }

    /// Decompress RLP-encoded data back to an address. Returns the
    /// resolved address, the number of bytes consumed, and whether the
    /// encoding was a raw 20-byte address (vs. a table index).
    pub fn decompress<B: StorageBackend>(
        &self,
        backend: &mut B,
        buf: &[u8],
    ) -> Result<(Address, u64, bool), AddressTableError> {
        let mut cursor = buf;
        let input = <Vec<u8> as Decodable>::decode(&mut cursor)
            .map_err(|_| AddressTableError::InvalidEncoding)?;
        let bytes_read = (buf.len() - cursor.len()) as u64;

        if input.len() == 20 {
            let mut addr_bytes = [0u8; 20];
            addr_bytes.copy_from_slice(&input);
            Ok((Address::from(addr_bytes), bytes_read, true))
        } else {
            let mut cursor = buf;
            let index = u64::decode(&mut cursor).map_err(|_| AddressTableError::InvalidEncoding)?;
            let bytes_read = (buf.len() - cursor.len()) as u64;
            let addr = self
                .lookup_index(backend, index)?
                .ok_or(AddressTableError::IndexOutOfRange(index))?;
            Ok((addr, bytes_read, false))
        }
    }

    fn by_address_get<B: StorageBackend>(
        &self,
        backend: &mut B,
        key: B256,
    ) -> Result<B256, AddressTableError> {
        let slot = self.by_address.slot_for_key(key);
        let value = backend
            .sload(self.by_address.account, slot)
            .map_err(Into::into)?;
        Ok(B256::from(value.to_be_bytes::<32>()))
    }

    fn by_address_set<B: StorageBackend>(
        &self,
        backend: &mut B,
        key: B256,
        value: B256,
    ) -> Result<(), AddressTableError> {
        let slot = self.by_address.slot_for_key(key);
        backend
            .sstore(self.by_address.account, slot, U256::from_be_bytes(value.0))
            .map_err(Into::into)?;
        Ok(())
    }

    fn backing_get_by_uint64<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
    ) -> Result<B256, AddressTableError> {
        let slot = self.backing_storage.new_slot(offset);
        let value = backend
            .sload(self.backing_storage.account, slot)
            .map_err(Into::into)?;
        Ok(B256::from(value.to_be_bytes::<32>()))
    }

    fn backing_set_by_uint64<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
        value: B256,
    ) -> Result<(), AddressTableError> {
        let slot = self.backing_storage.new_slot(offset);
        backend
            .sstore(
                self.backing_storage.account,
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
