// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::env;

fn main() {
    let target_arch = env::var("TARGET").unwrap();

    if target_arch.contains("wasm32") {
        println!("cargo:rustc-link-search=../../target/lib-wasm/");
    } else {
        println!("cargo:rustc-link-search=../target/lib/");
        println!("cargo:rustc-link-search=../../target/lib/");
    }
    println!("cargo:rustc-link-lib=static=brotlienc-static");
    println!("cargo:rustc-link-lib=static=brotlidec-static");
    println!("cargo:rustc-link-lib=static=brotlicommon-static");
}
