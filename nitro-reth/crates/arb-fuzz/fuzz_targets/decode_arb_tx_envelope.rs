#![no_main]

use arb_alloy_consensus::ArbTxEnvelope;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    let _ = ArbTxEnvelope::decode_typed(data);
});
