// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::Bytes32;

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let read = input[0] == 0;
    let slot = Bytes32::from_slice(&input[1..33]).map_err(|_| vec![0x00])?;

    Ok(if read {
        let data = arbitrum::load_bytes32(slot);
        data.0.into()
    } else {
        let data = Bytes32::from_slice(&input[33..]).map_err(|_| vec![0x01])?;
        arbitrum::store_bytes32(slot, data);
        vec![]
    })
}
