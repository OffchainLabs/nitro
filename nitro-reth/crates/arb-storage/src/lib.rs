//! Storage-backed types for ArbOS state.
//!
//! Provides typed wrappers over raw storage slots: integers, addresses,
//! byte arrays, queues, and vectors that persist in the state trie.

mod backed_types;
mod bytes_storage;
mod extra_types;
pub mod queue;
mod slot;
mod state_ops;
mod storage;
pub mod vector;

pub use backed_types::{
    StorageBackedAddress, StorageBackedAddressOrNil, StorageBackedBigInt, StorageBackedBigUint,
    StorageBackedInt64, StorageBackedUint64,
};
pub use bytes_storage::StorageBackedBytes;
pub use extra_types::{
    StorageBackedBips, StorageBackedUBips, StorageBackedUint16, StorageBackedUint24,
    StorageBackedUint32,
};
pub use queue::{initialize_queue, open_queue, Queue};
pub use slot::storage_key_map;
pub use state_ops::{
    ensure_account_in_bundle, ensure_arbos_account_in_bundle, get_account_balance,
    read_arbos_storage, read_storage_at, set_account_code, set_account_nonce, write_arbos_storage,
    write_storage_at, ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS,
};
pub use storage::Storage;
pub use vector::{open_sub_storage_vector, SubStorageVector};
