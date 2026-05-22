use arb_storage::StorageError;

/// Errors raised by the address-set storage container.
#[derive(thiserror::Error, Debug)]
pub enum AddressSetError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// `rectify_mapping` was called for an address that is not in the set.
    #[error("address is not a member of the set")]
    NotMember,

    /// `rectify_mapping` found the slot-to-index mapping already consistent;
    /// no repair is needed.
    #[error("address-set mapping is already consistent")]
    MappingAlreadyConsistent,
}
