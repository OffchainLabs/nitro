// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;

extern "C" {
    pub fn wavm_guest_call__getsp() -> usize;
    pub fn wavm_guest_call__resume();
}

#[derive(Clone)]
pub struct GoStack {
    sp: usize,
    offset: usize,
}

impl GoStack {
    pub fn new(sp: usize) -> Self {
        let offset = sp + 8;
        Self { sp, offset }
    }

    /// returns the pointer at which a value may be accessed, moving the offset past the value
    fn advance(&mut self, bytes: usize) -> usize {
        let before = self.offset;
        self.offset += bytes;
        before
    }

    pub unsafe fn read_u8(&mut self) -> u8 {
        wavm::caller_load8(self.advance(1))
    }

    pub unsafe fn read_u32(&mut self) -> u32 {
        wavm::caller_load32(self.advance(4))
    }

    pub unsafe fn read_u64(&mut self) -> u64 {
        wavm::caller_load64(self.advance(8))
    }

    pub unsafe fn write_u8(&mut self, x: u8) {
        wavm::caller_store8(self.advance(1), x);
    }

    pub unsafe fn write_u32(&mut self, x: u32) {
        wavm::caller_store32(self.advance(4), x);
    }

    pub unsafe fn write_u64(&mut self, x: u64) {
        wavm::caller_store64(self.advance(8), x);
    }

    pub unsafe fn skip_u8(&mut self) -> &mut Self {
        self.advance(1);
        self
    }

    pub unsafe fn skip_u32(&mut self) -> &mut Self {
        self.advance(4);
        self
    }

    pub unsafe fn skip_u64(&mut self) -> &mut Self {
        self.advance(8);
        self
    }

    pub unsafe fn read_go_slice(&mut self) -> (u64, u64) {
        let ptr = self.read_u64();
        let len = self.read_u64();
        self.skip_u64(); // skip the slice's capacity
        (ptr, len)
    }

    pub unsafe fn read_js_string(&mut self) -> Vec<u8> {
        let ptr = self.read_u64();
        let len = self.read_u64();
        wavm::read_slice(ptr, len)
    }

    /// Resumes the go runtime, updating the stack pointer.
    /// Safety: caller must cut lifetimes before this call.
    pub unsafe fn resume(&mut self) {
        let saved = self.offset - (self.sp + 8);
        wavm_guest_call__resume();
        *self = Self::new(wavm_guest_call__getsp());
        self.advance(saved);
    }
}
