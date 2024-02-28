
#![no_std]
use rand_pcg::Pcg32;

pub mod wasip1_stub;

const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;

pub fn create_pcg() -> Pcg32 {
    Pcg32::new(PCG_INIT_STATE, PCG_INIT_STREAM)
}

pub trait CallerEnv<'a> {
    fn read_u8(&self, ptr: u32) -> u8;

    fn read_u16(&self, ptr: u32) -> u16;

    fn read_u32(&self, ptr: u32) -> u32;

    fn read_u64(&self, ptr: u32) -> u64;

    fn write_u8(&mut self, ptr: u32, x: u8) -> &mut Self;

    fn write_u16(&mut self, ptr: u32, x: u16) -> &mut Self;

    fn write_u32(&mut self, ptr: u32, x: u32) -> &mut Self;

    fn write_u64(&mut self, ptr: u32, x: u64) -> &mut Self;

    fn print_string(&mut self, ptr: u32, len: u32);

    fn get_time(&self) -> u64;

    fn advance_time(&mut self, delta: u64);

    fn next_rand_u32(&mut self) -> u32;
}
