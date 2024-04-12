// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::{b256, B256},
    prelude::*,
    ArbResult,
};
extern crate alloc;

#[link(wasm_import_module = "vm_hooks")]
extern "C" {
    fn math_div(value: *mut u8, divisor: *const u8);
    fn math_mod(value: *mut u8, modulus: *const u8);
    fn math_pow(value: *mut u8, exponent: *const u8);
    fn math_add_mod(value: *mut u8, addend: *const u8, modulus: *const u8);
    fn math_mul_mod(value: *mut u8, multiplier: *const u8, modulus: *const u8);
}

#[entrypoint]
fn user_main(_: Vec<u8>) -> ArbResult {
    let mut value = b256!("eddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786");
    let unknown = b256!("c6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f");
    let ed25519 = b256!("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f");

    let part_1 = b256!("000000000000000000000000000000000000000000000000eddecf107b5740ce");
    let part_2 = b256!("000000000000000000000000000000000000000000000000fffffffefffffc2f");
    let part_3 = b256!("000000000000000000000000000000000000000000000000c6178c2de1078cd3");
    unsafe {
        math_mul_mod(value.as_mut_ptr(), unknown.as_ptr(), ed25519.as_ptr());
        math_add_mod(value.as_mut_ptr(), ed25519.as_ptr(), unknown.as_ptr());
        math_div(value.as_mut_ptr(), part_1.as_ptr());
        math_pow(value.as_mut_ptr(), part_2.as_ptr());
        math_mod(value.as_mut_ptr(), part_3.as_ptr());
        Ok(value.0.into())
    }
}
