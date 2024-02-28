use crate::{CallerEnv, create_pcg, wasip1_stub::Uptr};
use rand_pcg::Pcg32;
use rand::RngCore;

static mut TIME: u64 = 0;
static mut RNG: Option<Pcg32> = None;

#[derive(Default)]
pub struct StaticCallerEnv{}

pub static mut STATIC_CALLER: StaticCallerEnv = StaticCallerEnv{};


#[allow(dead_code)]
extern "C" {
    fn wavm_caller_load8(ptr: Uptr) -> u8;
    fn wavm_caller_load32(ptr: Uptr) -> u32;
    fn wavm_caller_store8(ptr: Uptr, val: u8);
    fn wavm_caller_store32(ptr: Uptr, val: u32);
    fn wavm_halt_and_set_finished() -> !;
}

impl CallerEnv<'static> for StaticCallerEnv {
    fn read_u8(&self, ptr: u32) -> u8 {
        unsafe {
            wavm_caller_load8(ptr)
        }
    }

    fn read_u16(&self, ptr: u32) -> u16 {
        let lsb = self.read_u8(ptr);
        let msb = self.read_u8(ptr+1);
        (msb as u16) << 8 | (lsb as u16)
    }

    fn read_u32(&self, ptr: u32) -> u32 {
        let lsb = self.read_u16(ptr);
        let msb = self.read_u16(ptr+2);
        (msb as u32) << 16 | (lsb as u32)
    }

    fn read_u64(&self, ptr: u32) -> u64 {
        let lsb = self.read_u32(ptr);
        let msb = self.read_u32(ptr+4);
        (msb as u64) << 32 | (lsb as u64)
    }

    fn write_u8(&mut self, ptr: u32, x: u8 ){
        unsafe {
            wavm_caller_store8(ptr, x);
        }
    }

    fn write_u16(&mut self, ptr: u32, x: u16) {
        self.write_u8(ptr, (x & 0xff) as u8);
        self.write_u8(ptr + 1, ((x >> 8) & 0xff) as u8);
    }

    fn write_u32(&mut self, ptr: u32, x: u32) {
        self.write_u16(ptr, (x & 0xffff) as u16);
        self.write_u16(ptr + 2, ((x >> 16) & 0xffff) as u16);
    }

    fn write_u64(&mut self, ptr: u32, x: u64) {
        self.write_u32(ptr, (x & 0xffffffff) as u32);
        self.write_u32(ptr + 4, ((x >> 16) & 0xffffffff) as u32);
    }

    fn print_string(&mut self, _ptr: u32, _len: u32) {} // TODO?

    fn get_time(&self) -> u64 {
        unsafe {
            TIME
        }
    }

    fn advance_time(&mut self, delta: u64) {
        unsafe {
            TIME += delta
        }
    }

    fn next_rand_u32(&mut self) -> u32 {
        unsafe {
            RNG.get_or_insert_with(|| create_pcg())
        }
        .next_u32()
    }
}
