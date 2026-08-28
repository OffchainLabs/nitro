//! Seed of the classic replay entrypoint (replay.wasm design doc, P3).
//!
//! For now it only proves the wasm build works end to end: brotli round-trips through the
//! `arbcompress` host imports, resolved by whichever runner instantiates the module.

use std::mem::MaybeUninit;

use nitro_brotli::{DEFAULT_WINDOW_SIZE, Dictionary};

fn main() {
    let payload = b"arb-replay brotli test payload ".repeat(16);

    let compressed = nitro_brotli::compress(&payload, 11, DEFAULT_WINDOW_SIZE, Dictionary::Empty)
        .expect("compress failed");

    let mut buf = vec![MaybeUninit::<u8>::uninit(); payload.len() + 64];
    let decompressed = nitro_brotli::decompress_fixed(&compressed, &mut buf, Dictionary::Empty)
        .expect("decompress failed");
    assert_eq!(decompressed, payload, "brotli round-trip mismatch");

    println!(
        "arb-replay brotli round-trip: ok ({} -> {} bytes)",
        payload.len(),
        compressed.len()
    );
}
