use arb_storage::StorageError;

/// Errors raised by the address-table storage container.
#[derive(Clone, thiserror::Error, Debug)]
pub enum AddressTableError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// `decompress` received bytes that are not a valid RLP-encoded address
    /// or compact index.
    #[error("failed to RLP-decode address-table input")]
    InvalidEncoding,

    /// `decompress` received an index that does not point at any registered
    /// address.
    #[error("address-table index {0} is out of range")]
    IndexOutOfRange(u64),
}
