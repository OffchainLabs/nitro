// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::{Bytes20, Bytes32};
use caller_env::{GuestPtr, MemAccess};
use std::mem::{self, MaybeUninit};
use wasmer::{Memory, MemoryView, StoreMut, WasmPtr};

pub struct JitMemAccess<'s> {
    pub memory: Memory,
    pub store: StoreMut<'s>,
}

impl JitMemAccess<'_> {
    fn view(&self) -> MemoryView<'_> {
        self.memory.view(&self.store)
    }

    pub fn write_bytes32(&mut self, ptr: GuestPtr, val: Bytes32) {
        self.write_slice(ptr, val.as_slice())
    }

    pub fn read_bytes20(&mut self, ptr: GuestPtr) -> Bytes20 {
        self.read_fixed(ptr).into()
    }

    pub fn read_bytes32(&mut self, ptr: GuestPtr) -> Bytes32 {
        self.read_fixed(ptr).into()
    }
}

impl MemAccess for JitMemAccess<'_> {
    fn read_u8(&self, ptr: GuestPtr) -> u8 {
        let ptr: WasmPtr<u8> = ptr.into();
        ptr.deref(&self.view()).read().unwrap()
    }

    fn read_u16(&self, ptr: GuestPtr) -> u16 {
        let ptr: WasmPtr<u16> = ptr.into();
        ptr.deref(&self.view()).read().unwrap()
    }

    fn read_u32(&self, ptr: GuestPtr) -> u32 {
        let ptr: WasmPtr<u32> = ptr.into();
        ptr.deref(&self.view()).read().unwrap()
    }

    fn read_u64(&self, ptr: GuestPtr) -> u64 {
        let ptr: WasmPtr<u64> = ptr.into();
        ptr.deref(&self.view()).read().unwrap()
    }

    fn write_u8(&mut self, ptr: GuestPtr, x: u8) {
        let ptr: WasmPtr<u8> = ptr.into();
        ptr.deref(&self.view()).write(x).unwrap();
    }

    fn write_u16(&mut self, ptr: GuestPtr, x: u16) {
        let ptr: WasmPtr<u16> = ptr.into();
        ptr.deref(&self.view()).write(x).unwrap();
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) {
        let ptr: WasmPtr<u32> = ptr.into();
        ptr.deref(&self.view()).write(x).unwrap();
    }

    fn write_u64(&mut self, ptr: GuestPtr, x: u64) {
        let ptr: WasmPtr<u64> = ptr.into();
        ptr.deref(&self.view()).write(x).unwrap();
    }

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8> {
        let mut data: Vec<MaybeUninit<u8>> = Vec::with_capacity(len);
        // SAFETY: read_uninit fills all available space
        unsafe {
            data.set_len(len);
            self.view()
                .read_uninit(ptr.into(), &mut data)
                .expect("bad read");
            mem::transmute::<Vec<MaybeUninit<u8>>, Vec<u8>>(data)
        }
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        self.read_slice(ptr, N).try_into().unwrap()
    }

    fn write_slice(&mut self, ptr: GuestPtr, src: &[u8]) {
        self.view().write(ptr.into(), src).unwrap();
    }
}
