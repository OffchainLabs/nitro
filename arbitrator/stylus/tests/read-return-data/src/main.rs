// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![no_main]

use stylus_sdk::{
    alloy_primitives::{b256, Address},
    call::RawCall,
    console, contract,
    prelude::*,
};

macro_rules! error {
    ($($msg:tt)*) => {{
        console!($($msg)*);
        panic!("invalid data")
    }};
}

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut call_data = input.as_slice();
    let mut read = || {
        let x = usize::from_be_bytes(call_data[..4].try_into().unwrap());
        call_data = &call_data[4..];
        x
    };

    let call_type = read();
    let offset = read();
    let size = read();
    let expected_size = read();
    let count = read();

    // Call identity precompile to test return data
    let precompile: Address = Address::from_word(b256!(
        "0000000000000000000000000000000000000000000000000000000000000004"
    ));

    let safe_offset = offset.min(call_data.len());

    if call_type == 2 {
        RawCall::new()
            .limit_return_data(offset, size)
            .call(precompile, call_data)?;
    }

    for _ in 0..count {
        let data = match call_type {
            0 => RawCall::new().call(precompile, call_data)?,
            1 => RawCall::new()
                .limit_return_data(offset, size)
                .call(precompile, call_data)?,
            2 => contract::read_return_data(offset, Some(size)),
            _ => error!("unknown call_type {call_type}"),
        };

        let expected_data = &call_data[safe_offset..][..expected_size];
        if data != expected_data {
            error!("call_type: {call_type}, calldata: {call_data:?}, offset: {offset}, size: {size} → {data:?} {expected_data:?}");
        }
    }

    Ok(vec![])
}
