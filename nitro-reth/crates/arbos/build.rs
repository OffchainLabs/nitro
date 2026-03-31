fn main() {
    let manifest_dir = std::path::PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").unwrap());
    let brotli_dir = manifest_dir.join("../../vendor/brotli");
    let include_dir = brotli_dir.join("c/include");

    cc::Build::new()
        .files([
            brotli_dir.join("c/common/constants.c"),
            brotli_dir.join("c/common/context.c"),
            brotli_dir.join("c/common/dictionary.c"),
            brotli_dir.join("c/common/platform.c"),
            brotli_dir.join("c/common/shared_dictionary.c"),
            brotli_dir.join("c/common/transform.c"),
            brotli_dir.join("c/enc/backward_references.c"),
            brotli_dir.join("c/enc/backward_references_hq.c"),
            brotli_dir.join("c/enc/bit_cost.c"),
            brotli_dir.join("c/enc/block_splitter.c"),
            brotli_dir.join("c/enc/brotli_bit_stream.c"),
            brotli_dir.join("c/enc/cluster.c"),
            brotli_dir.join("c/enc/command.c"),
            brotli_dir.join("c/enc/compound_dictionary.c"),
            brotli_dir.join("c/enc/compress_fragment.c"),
            brotli_dir.join("c/enc/compress_fragment_two_pass.c"),
            brotli_dir.join("c/enc/dictionary_hash.c"),
            brotli_dir.join("c/enc/encode.c"),
            brotli_dir.join("c/enc/encoder_dict.c"),
            brotli_dir.join("c/enc/entropy_encode.c"),
            brotli_dir.join("c/enc/fast_log.c"),
            brotli_dir.join("c/enc/histogram.c"),
            brotli_dir.join("c/enc/literal_cost.c"),
            brotli_dir.join("c/enc/memory.c"),
            brotli_dir.join("c/enc/metablock.c"),
            brotli_dir.join("c/enc/static_dict.c"),
            brotli_dir.join("c/enc/utf8_util.c"),
        ])
        .include(&include_dir)
        .warnings(false)
        .compile("brotli");
}
