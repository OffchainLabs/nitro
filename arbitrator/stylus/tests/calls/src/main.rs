// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{contract, debug, Bytes20, Bytes32};
use eyre::bail;

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut input = input.as_slice();
    let count = input[0];
    input = &input[1..];

    debug::println(format!("Calling {count} contract(s)"));
    for i in 0..count {
        let length = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
        input = &input[4..];

        debug::println(format!("Length {length} of {}", input.len()));
        do_call(&input[..length]).map_err(|_| vec![i])?;
        input = &input[length..];
    }
    Ok(vec![])
}

fn do_call(input: &[u8]) -> eyre::Result<Vec<u8>> {
    let addr = Bytes20::from_slice(&input[..20])?;
    let data = &input[20..];

    debug::println(format!("Calling {addr} with {} bytes", data.len()));
    match contract::call(addr, data, Bytes32::default()) {
        Ok(data) => Ok(data),
        Err(_) => bail!("call failed"),
    }
}
