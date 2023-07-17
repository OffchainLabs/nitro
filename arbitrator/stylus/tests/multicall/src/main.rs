// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{contract::Call, alloy_primitives::{Address, B256}};

stylus_sdk::entrypoint!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut input = input.as_slice();
    let count = input[0];
    input = &input[1..];

    // combined output of all calls
    let mut output = vec![];

    println(format!("Calling {count} contract(s)"));
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
        println(match value {
            Some(value) if !value.is_zero() => format!(
                "Calling {addr} with {} bytes and value {} {kind}",
                data.len(),
                hex::encode(value)
            ),
            _ => format!("Calling {addr} with {} bytes {kind}", curr.len()),
        });

        let return_data = match kind {
            0 => Call::new().value(value.unwrap_or_default()),
            1 => Call::new_delegate(),
            2 => Call::new_static(),
            x => panic!("unknown call kind {x}"),
        }.call(addr, data)?;
        if !return_data.is_empty() {
            println(format!(
                "Contract {addr} returned {} bytes",
                return_data.len()
            ));
        }
        output.extend(return_data);
        input = next;
    }

    Ok(output)
}

fn println(_text: impl AsRef<str>) {
    // arbitrum::debug::println(text)
}
