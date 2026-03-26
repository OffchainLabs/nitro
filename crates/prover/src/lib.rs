// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

pub mod binary;
mod host;
pub(crate) mod internal_func;
#[cfg(feature = "kzg")]
mod kzg;
pub mod machine;
/// cbindgen:ignore
pub mod memory;
pub(crate) mod memory_type;
pub mod merkle;
pub mod prepare;
mod print;
pub mod programs;
mod reinterpret;
pub mod utils;
pub mod value;
pub mod wavm;

#[cfg(test)]
mod test;

pub use machine::Machine;

use std::{marker::PhantomData, ptr};

#[repr(C)]
#[derive(Clone, Copy)]
pub struct CByteArray {
    pub ptr: *const u8,
    pub len: usize,
}

#[repr(C)]
pub struct RustSlice<'a> {
    pub ptr: *const u8,
    pub len: usize,
    pub phantom: PhantomData<&'a [u8]>,
}

impl<'a> RustSlice<'a> {
    pub fn new(slice: &'a [u8]) -> Self {
        if slice.is_empty() {
            return Self {
                ptr: ptr::null(),
                len: 0,
                phantom: PhantomData,
            };
        }
        Self {
            ptr: slice.as_ptr(),
            len: slice.len(),
            phantom: PhantomData,
        }
    }
}

#[repr(C)]
pub struct RustBytes {
    pub ptr: *mut u8,
    pub len: usize,
    pub cap: usize,
}

impl RustBytes {
    pub unsafe fn into_vec(self) -> Vec<u8> {
        Vec::from_raw_parts(self.ptr, self.len, self.cap)
    }

    pub unsafe fn write(&mut self, mut vec: Vec<u8>) {
        if vec.capacity() == 0 {
            *self = RustBytes {
                ptr: ptr::null_mut(),
                len: 0,
                cap: 0,
            };
            return;
        }
        self.ptr = vec.as_mut_ptr();
        self.len = vec.len();
        self.cap = vec.capacity();
        std::mem::forget(vec);
    }
}

#[repr(C)]
pub struct ResolvedPreimage {
    pub ptr: *mut u8,
    pub len: isize, // negative if not found
}
