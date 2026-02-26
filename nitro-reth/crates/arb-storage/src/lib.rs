mod slot;
mod state_ops;
mod storage;
mod backed_types;

pub use slot::storage_key_map;
pub use state_ops::{
    ensure_arbos_account_in_bundle, read_arbos_storage, write_arbos_storage, ARBOS_STATE_ADDRESS,
};
pub use storage::Storage;
pub use backed_types::{
    StorageBackedAddress, StorageBackedAddressOrNil, StorageBackedBigInt, StorageBackedBigUint,
    StorageBackedInt64, StorageBackedUint64,
};
