
#![no_std]

pub mod wasip1_stub;

pub const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
pub const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;

pub trait CallerEnv<'a> {
    fn caller_read_u8(&self, ptr: u32) -> u8;

    fn caller_read_u16(&self, ptr: u32) -> u16;

    fn caller_read_u32(&self, ptr: u32) -> u32;

    fn caller_read_u64(&self, ptr: u32) -> u64;

    fn caller_write_u8(&mut self, ptr: u32, x: u8) -> &mut Self;

    fn caller_write_u16(&mut self, ptr: u32, x: u16) -> &mut Self;

    fn caller_write_u32(&mut self, ptr: u32, x: u32) -> &mut Self;

    fn caller_write_u64(&mut self, ptr: u32, x: u64) -> &mut Self;

    fn caller_print_string(&mut self, ptr: u32, len: u32);

    fn caller_get_time(&self) -> u64;

    fn caller_advance_time(&mut self, delta: u64);

    fn next_rand_u32(&mut self) -> u32;
}
