use alloy_primitives::{keccak256, B256};
use revm::Database;

use arb_storage::{Storage, StorageBackedUint64, StorageBackend};

mod error;
pub use error::BlockhashesError;

pub struct Blockhashes<D> {
    backing_storage: Storage<D>,
    l1_block_number: StorageBackedUint64,
}

pub fn initialize_blockhashes<D: Database>(_backing_storage: &Storage<D>) {
    // no-op: next_block_number is already zero
}

pub fn open_blockhashes<D: Database>(backing_storage: Storage<D>) -> Blockhashes<D> {
    let l1_block_number = StorageBackedUint64::new(backing_storage.base_key(), 0);
    Blockhashes {
        backing_storage,
        l1_block_number,
    }
}

impl<D: Database> Blockhashes<D> {
    pub fn l1_block_number<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, BlockhashesError> {
        Ok(self.l1_block_number.get(backend)?)
    }

    pub fn block_hash<B: StorageBackend>(
        &self,
        backend: &mut B,
        number: u64,
    ) -> Result<Option<B256>, BlockhashesError> {
        let current_number = self.l1_block_number.get(backend)?;
        if number >= current_number || number + 256 < current_number {
            return Ok(None);
        }
        let hash = self.backing_storage.get_by_uint64(1 + (number % 256))?;
        Ok(Some(hash))
    }

    pub fn record_new_l1_block<B: StorageBackend>(
        &self,
        backend: &mut B,
        number: u64,
        block_hash: B256,
        arbos_version: u64,
    ) -> Result<(), BlockhashesError> {
        let mut next_number = self.l1_block_number.get(backend)?;

        if number < next_number {
            return Ok(());
        }

        if next_number + 256 < number {
            next_number = number - 256;
        }

        while next_number + 1 < number {
            next_number += 1;

            let mut next_num_buf = [0u8; 8];
            if arbos_version >= 8 {
                next_num_buf.copy_from_slice(&next_number.to_le_bytes());
            }

            let mut combined = Vec::with_capacity(40);
            combined.extend_from_slice(block_hash.as_slice());
            combined.extend_from_slice(&next_num_buf);
            let fill = keccak256(&combined);

            self.backing_storage
                .set_by_uint64(1 + (next_number % 256), fill)?;
        }

        self.backing_storage
            .set_by_uint64(1 + (number % 256), block_hash)?;
        Ok(self.l1_block_number.set(backend, number + 1)?)
    }
}
