//! Flat EIP-2200 storage-access gas for ArbOS state: a single read/write price
//! per slot, with no EVM cold/warm distinction or refunds.

/// Gas for reading one storage slot.
pub const STORAGE_READ_GAS: u64 = 800;
/// Gas for writing a non-zero value to one storage slot.
pub const STORAGE_WRITE_GAS: u64 = 20_000;
/// Gas for clearing one storage slot to zero.
pub const STORAGE_WRITE_ZERO_GAS: u64 = 5_000;

/// Gas for one storage write, priced lower when the slot is cleared to zero.
pub const fn write_cost(value_is_zero: bool) -> u64 {
    if value_is_zero {
        STORAGE_WRITE_ZERO_GAS
    } else {
        STORAGE_WRITE_GAS
    }
}
