// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{contract, debug, Bytes20, Bytes32};
use eyre::bail;

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    do_call(input).map_err(|_| vec![])
}

fn do_call(input: Vec<u8>) -> eyre::Result<Vec<u8>> {
    let addr = Bytes20::from_slice(&input[..20])?;
    let data = &input[20..];

    debug::println(format!("Calling {addr} with {}", hex::encode(data)));
    match contract::call(addr, data, Bytes32::default()) {
        Ok(data) => Ok(data),
        Err(_) => bail!("call failed"),
    }
}
