// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_main]

use brotli::Dictionary;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let dict = Dictionary::Empty;
    let split = data
        .get(0)
        .map(|x| *x as usize)
        .unwrap_or_default()
        .min(data.len());

    let (header, data) = data.split_at(split);
    let image = brotli::compress_into(&data, header.to_owned(), 0, 22, dict).unwrap();
    let prior = brotli::decompress(&image[split..], dict).unwrap();

    assert_eq!(&image[..split], header);
    assert_eq!(prior, data);
});
