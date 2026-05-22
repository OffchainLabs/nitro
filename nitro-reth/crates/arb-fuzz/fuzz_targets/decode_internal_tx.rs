#![no_main]

use arbos::internal_tx::decode_start_block_data;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let _ = decode_start_block_data(data);
});
