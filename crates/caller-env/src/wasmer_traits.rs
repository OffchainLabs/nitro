// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use alloc::vec::Vec;

use arbutil::{Bytes20, Bytes32};
use wasmer::{FromToNativeWasmType, Memory, MemoryView, StoreMut, WasmPtr};

use crate::{Errno, GuestPtr, MemAccess};

unsafe impl FromToNativeWasmType for GuestPtr {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self::new(u32::from_native(native))
    }

    fn to_native(self) -> i32 {
        u32::from(self).to_native()
    }
}

unsafe impl FromToNativeWasmType for Errno {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self(u16::from_native(native))
    }

    fn to_native(self) -> i32 {
        self.0.to_native()
    }
}

impl<T> From<GuestPtr> for WasmPtr<T> {
    fn from(value: GuestPtr) -> Self {
        WasmPtr::new(value.into())
    }
}

pub struct WasmerMem<'s> {
    memory: Memory,
    store: StoreMut<'s>,
}

impl<'s> WasmerMem<'s> {
    pub fn new(memory: Memory, store: StoreMut<'s>) -> Self {
        Self { memory, store }
    }

    fn view(&self) -> MemoryView<'_> {
        self.memory.view(&self.store)
    }

    pub fn read_bytes20(&self, ptr: GuestPtr) -> Bytes20 {
        self.read_fixed(ptr).into()
    }

    pub fn read_bytes32(&self, ptr: GuestPtr) -> Bytes32 {
        self.read_fixed(ptr).into()
    }

    pub fn write_bytes32(&mut self, ptr: GuestPtr, val: Bytes32) {
        self.write_slice(ptr, val.as_slice())
    }
}

impl MemAccess for WasmerMem<'_> {
    fn read_u8(&self, ptr: GuestPtr) -> u8 {
        self.read_fixed::<1>(ptr)[0]
    }
    fn read_u16(&self, ptr: GuestPtr) -> u16 {
        u16::from_le_bytes(self.read_fixed(ptr))
    }
    fn read_u32(&self, ptr: GuestPtr) -> u32 {
        u32::from_le_bytes(self.read_fixed(ptr))
    }
    fn read_u64(&self, ptr: GuestPtr) -> u64 {
        u64::from_le_bytes(self.read_fixed(ptr))
    }

    fn write_u8(&mut self, ptr: GuestPtr, x: u8) {
        self.write_slice(ptr, &[x])
    }
    fn write_u16(&mut self, ptr: GuestPtr, x: u16) {
        self.write_slice(ptr, &x.to_le_bytes())
    }
    fn write_u32(&mut self, ptr: GuestPtr, x: u32) {
        self.write_slice(ptr, &x.to_le_bytes())
    }
    fn write_u64(&mut self, ptr: GuestPtr, x: u64) {
        self.write_slice(ptr, &x.to_le_bytes())
    }

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8> {
        let mut data = vec![0u8; len];
        self.view().read(ptr.into(), &mut data).expect("read slice");
        data
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        let mut buf = [0u8; N];
        self.view()
            .read(ptr.into(), &mut buf)
            .expect("read fixed bytes");
        buf
    }

    fn write_slice(&mut self, ptr: GuestPtr, data: &[u8]) {
        self.view().write(ptr.into(), data).expect("write slice");
    }
}

pub trait HasMemory {
    fn memory(&self) -> Memory;
}
