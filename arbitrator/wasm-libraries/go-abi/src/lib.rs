// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;

extern "C" {
    pub fn wavm_guest_call__getsp() -> usize;
    pub fn wavm_guest_call__resume();
}

#[derive(Clone, Copy)]
#[repr(transparent)]
pub struct GoStack(pub usize);

impl GoStack {
    fn offset(&self, arg: usize) -> usize {
        self.0 + (arg + 1) * 8
    }

    pub unsafe fn read_u8(self, arg: usize) -> u8 {
        wavm::caller_load8(self.offset(arg))
    }

    pub unsafe fn read_u32(self, arg: usize) -> u32 {
        wavm::caller_load32(self.offset(arg))
    }

    pub unsafe fn read_u64(self, arg: usize) -> u64 {
        wavm::caller_load64(self.offset(arg))
    }

    pub unsafe fn write_u8(self, arg: usize, x: u8) {
        wavm::caller_store8(self.offset(arg), x);
    }

    pub unsafe fn write_u32(self, arg: usize, x: u32) {
        wavm::caller_store32(self.offset(arg), x);
    }

    pub unsafe fn write_u64(self, arg: usize, x: u64) {
        wavm::caller_store64(self.offset(arg), x);
    }
}
