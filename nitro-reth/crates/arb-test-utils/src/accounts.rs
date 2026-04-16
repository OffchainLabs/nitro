//! Canonical test account addresses.
//!
//! Mirror the accounts used in `testing/synthetic_suite.py` so tests
//! across Rust unit suites and the Python dual-exec suite share names.

use alloy_primitives::{address, Address};

pub fn alice() -> Address {
    address!("00000000000000000000000000000000000A11CE")
}

pub fn bob() -> Address {
    address!("00000000000000000000000000000000000B0B00")
}

pub fn charlie() -> Address {
    address!("00000000000000000000000000000000C4A841E0")
}

pub fn dave() -> Address {
    address!("0000000000000000000000000000000000DA7E00")
}

pub fn eve() -> Address {
    address!("0000000000000000000000000000000000E7E000")
}

/// Reserved for intentional-failure tests. Nonce drift won't cascade
/// into other accounts.
pub fn frank() -> Address {
    address!("00000000000000000000000000000000F4A11CE0")
}

/// Returns one of the canonical test accounts by index `0..6`.
pub fn test_account(idx: usize) -> Address {
    match idx {
        0 => alice(),
        1 => bob(),
        2 => charlie(),
        3 => dave(),
        4 => eve(),
        5 => frank(),
        _ => panic!("test_account: index {idx} out of range (0..6)"),
    }
}
