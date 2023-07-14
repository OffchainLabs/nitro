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
    let call_type = usize::from_be_bytes(input[..4].try_into().unwrap());
    let offset = usize::from_be_bytes(input[4..8].try_into().unwrap());
    let size = usize::from_be_bytes(input[8..12].try_into().unwrap());
    let expected_size = usize::from_be_bytes(input[12..16].try_into().unwrap());
    let count = usize::from_be_bytes(input[16..20].try_into().unwrap());
    let call_data = input[20..].to_vec();

    // Call identity precompile to test return data
    let precompile = Bytes20::from(0x4_u32);

    let safe_offset = offset.min(call_data.len());

    if call_type == 2 {
        Call::new().limit_return_data(offset, size).call(precompile, &call_data)?;
    }

    for _ in 0..count {
        let data = match call_type {
            0 => Call::new().call(precompile, &call_data)?,
            1 => Call::new().limit_return_data(offset, size).call(precompile, &call_data)?,
            2 => {
                contract::read_return_data(offset, Some(size))
            },
            _ => error!{format!{"unknown call_type {call_type}"}},
        };

        let expected_data = call_data[safe_offset..][..expected_size].to_vec();
        if data != expected_data {
            error!(format!("call_type: {call_type}, calldata: {call_data:?}, offset: {offset}, size: {size} → {data:?} {expected_data:?}"));
        }
    }

    Ok(vec![])
}
