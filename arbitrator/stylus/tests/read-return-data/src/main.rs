// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{
    contract::{self, Call},
    debug, Bytes20,
};

macro_rules! error {
    ($($msg:tt)*) => {{
        debug::println($($msg)*);
        panic!("invalid data")
    }};
}

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let offset = usize::from_be_bytes(input[..4].try_into().unwrap());
    let size = usize::from_be_bytes(input[4..8].try_into().unwrap());
    let expected_size = usize::from_be_bytes(input[8..12].try_into().unwrap());

    debug::println(format!("checking subset: {offset} {size} {expected_size}"));

    // Call identity precompile to test return data
    let calldata: [u8; 4] = [0, 1, 2, 3];
    let precompile = Bytes20::from(0x4_u32);

    let safe_offset = offset.min(calldata.len());
    let safe_size = size.min(calldata.len() - safe_offset);

    let full = Call::new().call(precompile, &calldata)?;
    if full != calldata {
        error!("data: {calldata:?}, offset: {offset}, size: {size} → {full:?}");
    }

    let limited = Call::new()
        .limit_return_data(offset, size)
        .call(precompile, &calldata)?;
    if limited.len() != expected_size || limited != calldata[safe_offset..][..safe_size] {
        error!(
            "data: {calldata:?}, offset: {offset}, size: {size}, expected size: {expected_size} → {limited:?}"
        );
    }

    let direct = contract::read_return_data(offset, Some(size));
    if direct != limited {
        error!(
            "data: {calldata:?}, offset: {offset}, size: {size}, expected size: {expected_size} → {direct:?}"
        );
    }

    Ok(limited)
}
