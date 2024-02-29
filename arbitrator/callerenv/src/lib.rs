#![no_std]
use rand_pcg::Pcg32;

extern crate alloc;

use alloc::vec::Vec;

#[cfg(feature = "static_caller")]
pub mod static_caller;

pub mod brotli;
pub mod wasip1_stub;
pub type Uptr = u32;

const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;

pub fn create_pcg() -> Pcg32 {
    Pcg32::new(PCG_INIT_STATE, PCG_INIT_STREAM)
}

pub trait MemAccess {
    fn read_u8(&self, ptr: Uptr) -> u8;

    fn read_u16(&self, ptr: Uptr) -> u16;

    fn read_u32(&self, ptr: Uptr) -> u32;

    fn read_u64(&self, ptr: Uptr) -> u64;

    fn write_u8(&mut self, ptr: Uptr, x: u8);

    fn write_u16(&mut self, ptr: Uptr, x: u16);

    fn write_u32(&mut self, ptr: Uptr, x: u32);

    fn write_u64(&mut self, ptr: Uptr, x: u64);

    fn read_slice(&self, ptr: Uptr, len: usize) -> Vec<u8>;

    fn write_slice(&mut self, ptr: Uptr, data: &[u8]);
}

pub trait ExecEnv {
    fn print_string(&mut self, message: &[u8]);

    fn get_time(&self) -> u64;

    fn advance_time(&mut self, delta: u64);

    fn next_rand_u32(&mut self) -> u32;
}
