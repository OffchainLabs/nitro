// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes20, Bytes32};

pub fn gas_price() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::tx_gas_price(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn ink_price() -> u64 {
    unsafe { hostio::CACHED_INK_PRICE.get() }
}

#[allow(clippy::inconsistent_digit_grouping)]
pub fn gas_to_ink(gas: u64) -> u64 {
    let ink_price = unsafe { hostio::CACHED_INK_PRICE.get() };
    gas.saturating_mul(100_00) / ink_price
}

#[allow(clippy::inconsistent_digit_grouping)]
pub fn ink_to_gas(ink: u64) -> u64 {
    let ink_price = unsafe { hostio::CACHED_INK_PRICE.get() };
    ink.saturating_mul(ink_price) / 100_00
}

pub fn origin() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { hostio::tx_origin(data.as_mut_ptr()) };
    Bytes20(data)
}
