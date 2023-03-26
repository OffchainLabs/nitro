// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{contract, debug, Bytes20};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut input = input.as_slice();
    let count = input[0];
    input = &input[1..];

    // combined output of all calls
    let mut output = vec![];

    debug::println(format!("Calling {count} contract(s)"));
    for _ in 0..count {
        let length = u32::from_be_bytes(input[..4].try_into().unwrap()) as usize;
        input = &input[4..];

        let addr = Bytes20::from_slice(&input[..20]).unwrap();
        let data = &input[20..length];
        debug::println(format!("Calling {addr} with {} bytes", data.len()));

        let return_data = contract::call(addr, data, None, None)?;
        if !return_data.is_empty() {
            debug::println(format!("Contract {addr} returned {} bytes", return_data.len()));
        }
        output.extend(return_data);
        input = &input[length..];
    }

    Ok(output)
}
