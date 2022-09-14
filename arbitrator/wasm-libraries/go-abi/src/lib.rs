use std::convert::TryFrom;

extern "C" {
    pub fn wavm_caller_load8(ptr: usize) -> u8;
    pub fn wavm_caller_load32(ptr: usize) -> u32;
    pub fn wavm_caller_store8(ptr: usize, val: u8);
    pub fn wavm_caller_store32(ptr: usize, val: u32);

    pub fn wavm_guest_call__getsp() -> usize;
    pub fn wavm_guest_call__resume();
}

pub unsafe fn wavm_caller_load64(ptr: usize) -> u64 {
    let lower = wavm_caller_load32(ptr);
    let upper = wavm_caller_load32(ptr + 4);
    lower as u64 | ((upper as u64) << 32)
}

pub unsafe fn wavm_caller_store64(ptr: usize, val: u64) {
    wavm_caller_store32(ptr, val as u32);
    wavm_caller_store32(ptr + 4, (val >> 32) as u32);
}

#[derive(Clone, Copy)]
#[repr(transparent)]
pub struct GoStack(pub usize);

impl GoStack {
    fn offset(&self, arg: usize) -> usize {
        self.0 + (arg + 1) * 8
    }

    pub unsafe fn read_u8(self, arg: usize) -> u8 {
        wavm_caller_load8(self.offset(arg))
    }

    pub unsafe fn read_u32(self, arg: usize) -> u32 {
        wavm_caller_load32(self.offset(arg))
    }

    pub unsafe fn read_u64(self, arg: usize) -> u64 {
        wavm_caller_load64(self.offset(arg))
    }

    pub unsafe fn write_u8(self, arg: usize, x: u8) {
        wavm_caller_store8(self.offset(arg), x);
    }

    pub unsafe fn write_u32(self, arg: usize, x: u32) {
        wavm_caller_store32(self.offset(arg), x);
    }

    pub unsafe fn write_u64(self, arg: usize, x: u64) {
        wavm_caller_store64(self.offset(arg), x);
    }
}

pub unsafe fn read_slice(ptr: u64, mut len: u64) -> Vec<u8> {
    let mut data = Vec::with_capacity(len as usize);
    if len == 0 {
        return data;
    }
    let mut ptr = usize::try_from(ptr).expect("Go pointer didn't fit in usize");
    while len >= 4 {
        data.extend(wavm_caller_load32(ptr).to_le_bytes());
        ptr += 4;
        len -= 4;
    }
    for _ in 0..len {
        data.push(wavm_caller_load8(ptr));
        ptr += 1;
    }
    data
}

pub unsafe fn write_slice(mut src: &[u8], ptr: u64) {
    if src.len() == 0 {
        return;
    }
    let mut ptr = usize::try_from(ptr).expect("Go pointer didn't fit in usize");
    while src.len() >= 4 {
        let mut arr = [0u8; 4];
        arr.copy_from_slice(&src[..4]);
        wavm_caller_store32(ptr, u32::from_le_bytes(arr));
        ptr += 4;
        src = &src[4..];
    }
    for &byte in src {
        wavm_caller_store8(ptr, byte);
        ptr += 1;
    }
}
