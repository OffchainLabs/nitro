// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{Bytes20, contract};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let addr = Bytes20::from_slice(&input[0..20]).expect("incorrect slice size for Bytes20");
    let ink_bytes: [u8; 8] = input[20..28].try_into().expect("incorrect slice length for ink u64");
    let ink = u64::from_be_bytes(ink_bytes);
    contract::call(addr, &input[28..], None, Some(ink))
}
