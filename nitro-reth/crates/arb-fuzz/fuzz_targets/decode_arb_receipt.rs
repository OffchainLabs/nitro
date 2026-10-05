#![no_main]

use alloy_eips::eip2718::Decodable2718;
use arb_primitives::receipt::ArbReceipt;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let mut slice = data;
    let _ = ArbReceipt::decode_2718(&mut slice);
});
