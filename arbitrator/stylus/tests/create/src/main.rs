// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{contract::Deploy, evm, Bytes32};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let kind = input[0];
    let mut input = &input[1..];

    let endowment = Bytes32::from_slice(&input[..32]).unwrap();
    input = &input[32..];

    let mut salt = None;
    if kind == 2 {
        salt = Some(Bytes32::from_slice(&input[..32]).unwrap());
        input = &input[32..];
    }

    let code = input;
    let contract = Deploy::new().salt_option(salt).deploy(code, endowment)?;
    evm::log(&[contract.into()], &[]).unwrap();
    Ok(contract.to_vec())
}
