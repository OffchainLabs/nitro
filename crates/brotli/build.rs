// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

fn main() {
    // Without `link` nothing is compiled or linked: brotli is reached through the
    // `arbcompress` host imports, which only exist on wasm targets. Note: build scripts
    // run on the host, so the target must be read from the environment, not via cfg.
    if std::env::var_os("CARGO_FEATURE_LINK").is_none() {
        let target = std::env::var("TARGET").unwrap();
        assert!(
            target.contains("wasm32"),
            "the `link` feature is required for non-wasm targets"
        );
        return;
    }
    link();
}

#[cfg(not(feature = "cc_brotli"))]
fn link() {
    let target_arch = std::env::var("TARGET").unwrap();

    if target_arch.contains("wasm32") {
        println!("cargo:rustc-link-search=target/lib-wasm/");
    } else if target_arch.contains("riscv64") {
        println!("cargo:rustc-link-search=../../target/lib-sp1/lib");
    } else {
        println!("cargo:rustc-link-search=target/lib/");
        println!("cargo:rustc-link-search=../../target/lib/");
    }
    println!("cargo:rustc-link-lib=static=brotlienc-static");
    println!("cargo:rustc-link-lib=static=brotlidec-static");
    println!("cargo:rustc-link-lib=static=brotlicommon-static");
}

#[cfg(feature = "cc_brotli")]
fn link() {
    use std::{env, path::PathBuf};

    assert!(
        !env::var("TARGET").unwrap().contains("wasm32"),
        "cc_brotli cannot target wasm; wasm builds link prebuilt libs or use host imports"
    );

    let manifest_dir = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap());
    let include_dir = manifest_dir.join("../../brotli/c/include");
    cc::Build::new()
        .files(&[
            "../../brotli/c/common/constants.c",
            "../../brotli/c/common/context.c",
            "../../brotli/c/common/dictionary.c",
            "../../brotli/c/common/platform.c",
            "../../brotli/c/common/shared_dictionary.c",
            "../../brotli/c/common/transform.c",
            "../../brotli/c/dec/bit_reader.c",
            "../../brotli/c/dec/decode.c",
            "../../brotli/c/dec/huffman.c",
            "../../brotli/c/dec/state.c",
            "../../brotli/c/enc/backward_references.c",
            "../../brotli/c/enc/backward_references_hq.c",
            "../../brotli/c/enc/bit_cost.c",
            "../../brotli/c/enc/block_splitter.c",
            "../../brotli/c/enc/brotli_bit_stream.c",
            "../../brotli/c/enc/cluster.c",
            "../../brotli/c/enc/command.c",
            "../../brotli/c/enc/compound_dictionary.c",
            "../../brotli/c/enc/compress_fragment.c",
            "../../brotli/c/enc/compress_fragment_two_pass.c",
            "../../brotli/c/enc/dictionary_hash.c",
            "../../brotli/c/enc/encode.c",
            "../../brotli/c/enc/encoder_dict.c",
            "../../brotli/c/enc/entropy_encode.c",
            "../../brotli/c/enc/fast_log.c",
            "../../brotli/c/enc/histogram.c",
            "../../brotli/c/enc/literal_cost.c",
            "../../brotli/c/enc/memory.c",
            "../../brotli/c/enc/metablock.c",
            "../../brotli/c/enc/static_dict.c",
            "../../brotli/c/enc/utf8_util.c",
        ])
        .includes(["../../brotli/c/include"])
        .define("BROTLI_BUILD_ENC_EXTRA_API", None)
        .define("BROTLI_HAVE_LOG2", "1")
        .warnings(false)
        .compile("brotli");

    println!("cargo:include={}", include_dir.display());
    println!("cargo:rerun-if-changed=../../brotli/c");
}
