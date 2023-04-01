// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{evm, Bytes32};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let num_topics = input[0];
    let mut input = &input[1..];

    let mut topics = vec![];
    for _ in 0..num_topics {
        topics.push(Bytes32::from_slice(&input[..32]).unwrap());
        input = &input[32..];
    }
    evm::log(&topics, input).unwrap();
    Ok(vec![])
}
