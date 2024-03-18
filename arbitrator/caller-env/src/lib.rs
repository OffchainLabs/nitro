// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

use alloc::vec::Vec;
use rand_pcg::Pcg32;

pub use guest_ptr::GuestPtr;
pub use wasip1_stub::Errno;

#[cfg(feature = "static_caller")]
pub mod static_caller;

#[cfg(feature = "wasmer_traits")]
pub mod wasmer_traits;

pub mod brotli;
mod guest_ptr;
pub mod wasip1_stub;

/// Initializes a deterministic, psuedo-random number generator with a fixed seed.
pub fn create_pcg() -> Pcg32 {
    const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
    const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;
    Pcg32::new(PCG_INIT_STATE, PCG_INIT_STREAM)
}

/// Access Guest memory.
pub trait MemAccess {
    fn read_u8(&self, ptr: GuestPtr) -> u8;

    fn read_u16(&self, ptr: GuestPtr) -> u16;

    fn read_u32(&self, ptr: GuestPtr) -> u32;

    fn read_u64(&self, ptr: GuestPtr) -> u64;

    fn write_u8(&mut self, ptr: GuestPtr, x: u8);

    fn write_u16(&mut self, ptr: GuestPtr, x: u16);

    fn write_u32(&mut self, ptr: GuestPtr, x: u32);

    fn write_u64(&mut self, ptr: GuestPtr, x: u64);

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8>;

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N];

    fn write_slice(&mut self, ptr: GuestPtr, data: &[u8]);
}

/// Update the Host environment.
pub trait ExecEnv {
    fn advance_time(&mut self, ns: u64);

    fn get_time(&self) -> u64;

    fn next_rand_u32(&mut self) -> u32;

    fn print_string(&mut self, message: &[u8]);
}
