// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

extern crate alloc;

use stylus_sdk::{
    storage::{StorageCache, GlobalStorage},
    alloy_primitives::{Address, B256},
    alloy_sol_types::sol,
    call::RawCall,
    console,
    evm,
    prelude::*,
};

use wee_alloc::WeeAlloc;

#[global_allocator]
static ALLOC: WeeAlloc = WeeAlloc::INIT;

sol!{
    event Called(address addr, uint8 count, bool success, bytes return_data);
    event Storage(bytes32 slot, bytes32 data, bool write);
}

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut input = input.as_slice();
    let count = input[0];
    input = &input[1..];

    // combined output of all calls
    let mut output = vec![];

    console!("Performing {count} action(s)");
    for _ in 0..count {
        let length = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
        input = &input[4..];

        let next = &input[length..];
        let mut curr = &input[..length];

        let kind = curr[0];
        curr = &curr[1..];

        if kind & 0xf0 == 0 {
            // caller
            let mut value = None;
            if kind & 0x3 == 0 {
                value = Some(B256::try_from(&curr[..32]).unwrap());
                curr = &curr[32..];
            };

            let addr = Address::try_from(&curr[..20]).unwrap();
            let data = &curr[20..];
            match value {
                Some(value) if !value.is_zero() => console!(
                    "Calling {addr} with {} bytes and value {} {kind}",
                    data.len(),
                    hex::encode(value)
                ),
                _ => console!("Calling {addr} with {} bytes {kind}", curr.len()),
            }

            let raw_call = match kind & 0x3 {
                0 => RawCall::new_with_value(value.unwrap_or_default().into()),
                1 => RawCall::new_delegate(),
                2 => RawCall::new_static(),
                x => panic!("unknown call kind {x}"),
            };
            let (success, return_data) = match unsafe { raw_call.call(addr, data) } {
                Ok(return_data) => (true, return_data),
                Err(revert_data) => {
                    if kind & 0x4 == 0 {
                        return Err(revert_data)
                    }
                    (false, vec![])
                },
            };
        
            if !return_data.is_empty() {
                console!("Contract {addr} returned {} bytes", return_data.len());
            }
            if kind & 0x8 != 0 {
                evm::log(Called { addr, count, success, return_data: return_data.clone() })
            }
            output.extend(return_data);
        } else if kind & 0xf0 == 0x10  {
            // storage
            let slot = B256::try_from(&curr[..32]).unwrap();
            curr = &curr[32..];
            let data;
            let write;
            if kind & 0x7 == 0 {
                console!("writing slot {}", curr.len());
                data = B256::try_from(&curr[..32]).unwrap();
                write = true;
                unsafe { StorageCache::set_word(slot.into(), data.into()) };
                StorageCache::flush();
            } else if kind & 0x7 == 1{
                console!("reading slot");
                write = false;
                data = StorageCache::get_word(slot.into());
                output.extend(data.clone());
            } else {
                panic!("unknown storage kind {kind}")
            }
            if kind & 0x8 != 0 {
                console!("slot: {}, data: {}, write {write}", slot, data);
                evm::log(Storage { slot: slot.into(), data: data.into(), write })
            }
        } else {
            panic!("unknown action {kind}")
        }
        input = next;
    }

    Ok(output)
}
