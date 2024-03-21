// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::{env, path::Path};

fn main() {
    let target_arch = env::var("TARGET").unwrap();
    let manifest = env::var("CARGO_MANIFEST_DIR").unwrap();
    let manifest = Path::new(&manifest);

    if target_arch.contains("wasm32") {
        println!("cargo:rustc-link-search=../../target/lib-wasm/");
    } else {
        // search for brotli libs depending on where cargo is invoked
        let arbitrator = Some(Path::new("arbitrator").file_name());
        match arbitrator == manifest.parent().map(Path::file_name) {
            true => println!("cargo:rustc-link-search=../target/lib/"),
            false => println!("cargo:rustc-link-search=../../target/lib/"),
        }
    }
    println!("cargo:rustc-link-lib=static=brotlienc-static");
    println!("cargo:rustc-link-lib=static=brotlidec-static");
    println!("cargo:rustc-link-lib=static=brotlicommon-static");
}
