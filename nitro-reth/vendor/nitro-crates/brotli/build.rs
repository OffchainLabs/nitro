fn main() {
    use std::env;
    use std::path::PathBuf;
    let manifest_dir = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap());
    let brotli_c = manifest_dir.join("../../brotli/c");
    let include_dir = brotli_c.join("include");

    cc::Build::new()
        .files(&[
            brotli_c.join("common/constants.c"),
            brotli_c.join("common/context.c"),
            brotli_c.join("common/dictionary.c"),
            brotli_c.join("common/platform.c"),
            brotli_c.join("common/shared_dictionary.c"),
            brotli_c.join("common/transform.c"),
            brotli_c.join("dec/bit_reader.c"),
            brotli_c.join("dec/decode.c"),
            brotli_c.join("dec/huffman.c"),
            brotli_c.join("dec/state.c"),
            brotli_c.join("enc/backward_references.c"),
            brotli_c.join("enc/backward_references_hq.c"),
            brotli_c.join("enc/bit_cost.c"),
            brotli_c.join("enc/block_splitter.c"),
            brotli_c.join("enc/brotli_bit_stream.c"),
            brotli_c.join("enc/cluster.c"),
            brotli_c.join("enc/command.c"),
            brotli_c.join("enc/compound_dictionary.c"),
            brotli_c.join("enc/compress_fragment.c"),
            brotli_c.join("enc/compress_fragment_two_pass.c"),
            brotli_c.join("enc/dictionary_hash.c"),
            brotli_c.join("enc/encode.c"),
            brotli_c.join("enc/encoder_dict.c"),
            brotli_c.join("enc/entropy_encode.c"),
            brotli_c.join("enc/fast_log.c"),
            brotli_c.join("enc/histogram.c"),
            brotli_c.join("enc/literal_cost.c"),
            brotli_c.join("enc/memory.c"),
            brotli_c.join("enc/metablock.c"),
            brotli_c.join("enc/static_dict.c"),
            brotli_c.join("enc/utf8_util.c"),
        ])
        .include(&include_dir)
        .define("BROTLI_BUILD_ENC_EXTRA_API", None)
        .define("BROTLI_HAVE_LOG2", "1")
        .warnings(false)
        .compile("brotli");

    println!("cargo:include={}", include_dir.display());
}
