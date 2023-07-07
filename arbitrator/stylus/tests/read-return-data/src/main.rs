// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{contract::{self, Call}, debug};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let offset = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
    let size = u32::from_be_bytes(input[4..8].try_into().unwrap()) as usize;
    let expected_size = u32::from_be_bytes(input[8..12].try_into().unwrap()) as usize;

    debug::println(format!("checking return data subset: {offset} {size}"));
    // Call identity precompile to test return data
    let call_data: [u8; 4] = [0, 1, 2, 3];
    let identity_precompile: u32 = 0x4;
    let mut safe_offset = offset;
    if safe_offset > call_data.len() {
        safe_offset = call_data.len();
    }
    let mut safe_size = size;
    if safe_size > call_data.len() - safe_offset {
        safe_size = call_data.len() - safe_offset;
    }

    let full_call_return_data = Call::new().
        call(identity_precompile.into(), &call_data)?;
    if full_call_return_data != call_data {
        debug::println(
            format!("data: {call_data:#?}, offset: {offset}, size: {size}, incorrect full call data: {full_call_return_data:#?}"),
        );
        panic!("invalid data");
    }

    let limit_call_return_data = Call::new().
        limit_return_data(offset, size).
        call(identity_precompile.into(), &call_data)?;
    if limit_call_return_data.len() != expected_size ||
        limit_call_return_data != call_data[safe_offset..safe_offset+safe_size] {
        debug::println(
            format!("data: {call_data:#?}, offset: {offset}, size: {size}, expected size: {expected_size}, incorrect limit call data: {limit_call_return_data:#?}"),
        );
        panic!("invalid data");
    }

    let partial_return_data = contract::partial_return_data(offset, size);
    if partial_return_data != limit_call_return_data {
        debug::println(
            format!("data: {call_data:#?}, offset: {offset}, size: {size}, expected size: {expected_size}, incorrect partial call data: {partial_return_data:#?}"),
        );
        panic!("invalid data");
    }

    Ok(limit_call_return_data)
}

