// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{create_pcg, ExecEnv, GuestPtr, MemAccess};
use alloc::vec::Vec;
use rand::RngCore;
use rand_pcg::Pcg32;

extern crate alloc;

static mut TIME: u64 = 0;
static mut RNG: Option<Pcg32> = None;

pub struct StaticMem;
pub struct StaticExecEnv;

pub static mut STATIC_MEM: StaticMem = StaticMem;
pub static mut STATIC_ENV: StaticExecEnv = StaticExecEnv;

extern "C" {
    fn wavm_caller_load8(ptr: GuestPtr) -> u8;
    fn wavm_caller_load32(ptr: GuestPtr) -> u32;
    fn wavm_caller_store8(ptr: GuestPtr, val: u8);
    fn wavm_caller_store32(ptr: GuestPtr, val: u32);
}

impl MemAccess for StaticMem {
    fn read_u8(&self, ptr: GuestPtr) -> u8 {
        unsafe { wavm_caller_load8(ptr) }
    }

    fn read_u16(&self, ptr: GuestPtr) -> u16 {
        let lsb = self.read_u8(ptr);
        let msb = self.read_u8(ptr + 1);
        (msb as u16) << 8 | (lsb as u16)
    }

    fn read_u32(&self, ptr: GuestPtr) -> u32 {
        unsafe { wavm_caller_load32(ptr) }
    }

    fn read_u64(&self, ptr: GuestPtr) -> u64 {
        let lsb = self.read_u32(ptr);
        let msb = self.read_u32(ptr + 4);
        (msb as u64) << 32 | (lsb as u64)
    }

    fn write_u8(&mut self, ptr: GuestPtr, x: u8) {
        unsafe { wavm_caller_store8(ptr, x) }
    }

    fn write_u16(&mut self, ptr: GuestPtr, x: u16) {
        self.write_u8(ptr, (x & 0xff) as u8);
        self.write_u8(ptr + 1, ((x >> 8) & 0xff) as u8);
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) {
        unsafe { wavm_caller_store32(ptr, x) }
    }

    fn write_u64(&mut self, ptr: GuestPtr, x: u64) {
        self.write_u32(ptr, (x & 0xffffffff) as u32);
        self.write_u32(ptr + 4, ((x >> 32) & 0xffffffff) as u32);
    }

    fn read_slice(&self, mut ptr: GuestPtr, mut len: usize) -> Vec<u8> {
        let mut data = Vec::with_capacity(len);
        if len == 0 {
            return data;
        }
        while len >= 4 {
            data.extend(self.read_u32(ptr).to_le_bytes());
            ptr += 4;
            len -= 4;
        }
        for _ in 0..len {
            data.push(self.read_u8(ptr));
            ptr += 1;
        }
        data
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        self.read_slice(ptr, N).try_into().unwrap()
    }

    fn write_slice(&mut self, mut ptr: GuestPtr, mut src: &[u8]) {
        while src.len() >= 4 {
            let mut arr = [0; 4];
            arr.copy_from_slice(&src[..4]);
            self.write_u32(ptr, u32::from_le_bytes(arr));
            ptr += 4;
            src = &src[4..];
        }
        for &byte in src {
            self.write_u8(ptr, byte);
            ptr += 1;
        }
    }
}

impl ExecEnv for StaticExecEnv {
    fn get_time(&self) -> u64 {
        unsafe { TIME }
    }

    fn advance_time(&mut self, delta: u64) {
        unsafe { TIME += delta }
    }

    fn next_rand_u32(&mut self) -> u32 {
        unsafe { RNG.get_or_insert_with(create_pcg) }.next_u32()
    }
}
