// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::tx;

arbitrum::arbitrum_main!(user_main);

fn user_main(_input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let gas_price = tx::gas_price();
    let origin = tx::origin();

    let mut output = vec![];
    output.extend(gas_price.to_be_bytes());
    output.extend(origin.0);
    Ok(output)
}
