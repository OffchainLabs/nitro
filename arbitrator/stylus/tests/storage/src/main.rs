// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::B256,
    console,
    storage::{StorageCache, GlobalStorage},
    stylus_proc::entrypoint,
};

#[link(wasm_import_module = "vm_hooks")]
extern "C" {
    fn transient_load_bytes32(key: *const u8, dest: *mut u8);
    fn transient_store_bytes32(key: *const u8, value: *const u8);
}

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let slot = B256::try_from(&input[1..33]).unwrap();

    Ok(match input[0] {
        0 => {
            console!("read {slot}");
            let data = StorageCache::get_word(slot.into());
            console!("value {data}");
            data.0.into()
        }
        1 => {
            console!("write {slot}");
            let data = B256::try_from(&input[33..]).unwrap();
            unsafe { StorageCache::set_word(slot.into(), data) };
            console!(("value {data}"));
            vec![]
        }
        2 => unsafe {
            let mut data = [0; 32];
            transient_load_bytes32(slot.as_ptr(), data.as_mut_ptr());
            data.into()
        }
        _ => unsafe {
            let data = B256::try_from(&input[33..]).unwrap();
            transient_store_bytes32(slot.as_ptr(), data.as_ptr());
            vec![]
        }
    })
}
