// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{debug, load_bytes32, store_bytes32, alloy_primitives::B256};

stylus_sdk::entrypoint!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let read = input[0] == 0;
    let slot = B256::try_from(&input[1..33]).unwrap();

    Ok(if read {
        debug::println(format!("read  {slot}"));
        let data = load_bytes32(slot);
        debug::println(format!("value {data}"));
        data.0.into()
    } else {
        debug::println(format!("write {slot}"));
        let data = B256::try_from(&input[33..]).unwrap();
        store_bytes32(slot, data);
        debug::println(format!("value {data}"));
        vec![]
    })
}
