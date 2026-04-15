// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use std::fs::File;
use std::io::Write;

fn main() {
    println!("cargo:rerun-if-changed=../wasm-libraries/forward");
    let mut out = Vec::new();
    forward::forward_stub(&mut out).expect("Failed to write stub");
    let mut file = File::create("src/forward_stub.wat").unwrap();
    file.write_all(&out).unwrap();
}
