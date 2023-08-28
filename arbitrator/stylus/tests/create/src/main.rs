// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{alloy_primitives::B256, deploy::RawDeploy, evm, prelude::*};

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let kind = input[0];
    let mut input = &input[1..];

    let endowment = B256::try_from(&input[..32]).unwrap();
    input = &input[32..];

    let mut salt = None;
    if kind == 2 {
        salt = Some(B256::try_from(&input[..32]).unwrap());
        input = &input[32..];
    }

    let code = input;
    let contract = unsafe { RawDeploy::new().salt_option(salt).deploy(code, endowment)? };
    evm::raw_log(&[contract.into_word()], &[]).unwrap();
    Ok(contract.to_vec())
}
