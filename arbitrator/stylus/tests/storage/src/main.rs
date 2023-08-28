// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::B256,
    console,
    storage::{load_bytes32, store_bytes32},
    stylus_proc::entrypoint,
};

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let read = input[0] == 0;
    let slot = B256::try_from(&input[1..33]).unwrap();

    Ok(if read {
        console!("read {slot}");
        let data = unsafe { load_bytes32(slot.into()) };
        console!("value {data}");
        data.0.into()
    } else {
        console!("write {slot}");
        let data = B256::try_from(&input[33..]).unwrap();
        unsafe { store_bytes32(slot.into(), data) };
        console!(("value {data}"));
        vec![]
    })
}
