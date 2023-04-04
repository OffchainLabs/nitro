// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::tx;

arbitrum::arbitrum_main!(user_main);

fn user_main(_input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let origin = tx::origin();

    let mut output = vec![];
    output.extend(origin.0);
    Ok(output)
}
