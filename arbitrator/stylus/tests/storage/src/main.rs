// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{debug, load_bytes32, store_bytes32, Bytes32};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let read = input[0] == 0;
    let slot = Bytes32::from_slice(&input[1..33]).unwrap();

    Ok(if read {
        debug::println(format!("read  {slot}"));
        let data = load_bytes32(slot);
        debug::println(format!("value {data}"));
        data.0.into()
    } else {
        debug::println(format!("write {slot}"));
        let data = Bytes32::from_slice(&input[33..]).unwrap();
        store_bytes32(slot, data);
        debug::println(format!("value {data}"));
        vec![]
    })
}
