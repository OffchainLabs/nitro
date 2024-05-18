// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::{Address, B256},
    call::RawCall,
    console,
    prelude::*,
};

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut input = input.as_slice();
    let count = input[0];
    input = &input[1..];

    // combined output of all calls
    let mut output = vec![];

    console!("Calling {count} contract(s)");
    for _ in 0..count {
        let length = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
        input = &input[4..];

        let next = &input[length..];
        let mut curr = &input[..length];

        let kind = curr[0];
        curr = &curr[1..];

        let mut value = None;
        if kind == 0 {
            value = Some(B256::try_from(&curr[..32]).unwrap());
            curr = &curr[32..];
        }

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

        let raw_call = match kind {
            0 => RawCall::new_with_value(value.unwrap_or_default().into()),
            1 => RawCall::new_delegate(),
            2 => RawCall::new_static(),
            x => panic!("unknown call kind {x}"),
        };
        let return_data = unsafe { raw_call.call(addr, data)? };

        if !return_data.is_empty() {
            console!("Contract {addr} returned {} bytes", return_data.len());
        }
        output.extend(return_data);
        input = next;
    }

    Ok(output)
}
