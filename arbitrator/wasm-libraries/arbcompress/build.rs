// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

fn main() {
    // Tell Cargo that if the given file changes, to rerun this build script.
    println!("cargo:rustc-link-search=../../target/lib-wasm/");
    println!("cargo:rustc-link-search=../target/lib/");
    println!("cargo:rustc-link-lib=static=brotlienc-static");
    println!("cargo:rustc-link-lib=static=brotlidec-static");
    println!("cargo:rustc-link-lib=static=brotlicommon-static");
}
