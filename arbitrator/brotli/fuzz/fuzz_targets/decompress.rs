// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_main]

use brotli::Dictionary;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let mut data = data;
    let dict = Dictionary::StylusProgram;

    let mut space = 0_u32;
    if data.len() >= 8 {
        space = u32::from_le_bytes(data[..4].try_into().unwrap());
        data = &data[4..];
    }

    let mut array = Vec::with_capacity(space as usize % 65536);
    let array = &mut array.spare_capacity_mut();

    let plain = brotli::decompress(data, dict);
    let fixed = brotli::decompress_fixed(data, array, dict);

    if let Ok(fixed) = fixed {
        assert_eq!(fixed.len(), plain.unwrap().len()); // fixed succeeding implies both do
    }
});
