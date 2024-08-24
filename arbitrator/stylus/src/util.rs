// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::crypto;
use eyre::Report;

/// This function panics while saving an offending wasm to disk.
pub fn panic_with_wasm(wasm: &[u8], error: Report) -> ! {
    // save at a deterministic path
    let hash = hex::encode(crypto::keccak(wasm));
    let mut path = std::env::temp_dir();
    path.push(format!("stylus-panic-{hash}.wasm"));

    // try to save to disk, otherwise dump to the console
    if let Err(io_error) = std::fs::write(&path, wasm) {
        let wasm = hex::encode(wasm);
        panic!("failed to write fatal wasm {error:?}: {io_error:?}\nwasm: {wasm}");
    }
    panic!("encountered fatal wasm: {error:?}");
}
