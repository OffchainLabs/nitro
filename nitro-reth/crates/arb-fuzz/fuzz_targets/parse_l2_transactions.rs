#![no_main]

use alloy_primitives::{Address, B256, U256};
use libfuzzer_sys::fuzz_target;

fuzz_target!(|data: &[u8]| {
    if data.len() < 6 {
        return;
    }
    let kind = data[0];
    let body = &data[5..];

    let request_id = if data[1] & 1 == 0 {
        None
    } else {
        let mut buf = [0u8; 32];
        buf[..body.len().min(32)].copy_from_slice(&body[..body.len().min(32)]);
        Some(B256::from(buf))
    };
    let l1_base_fee = if data[2] & 1 == 0 {
        None
    } else {
        Some(U256::from(u64::from_le_bytes([
            data[3], data[4], 0, 0, 0, 0, 0, 0,
        ])))
    };

    let _ = arbos::parse_l2::parse_l2_transactions(
        kind,
        Address::ZERO,
        body,
        request_id,
        l1_base_fee,
        42_161,
    );
});
