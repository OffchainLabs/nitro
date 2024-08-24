// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![no_main]

use libfuzzer_sys::fuzz_target;

fuzz_target!(|arg: (&[u8], u32, u32)| {
    let data = arg.0;
    let quality = arg.1;
    let window = arg.2;
    let _ = brotli::compress(
        data,
        1 + quality % 12,
        10 + window % 15,
        brotli::Dictionary::StylusProgram,
    );
});
