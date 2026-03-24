// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use caller_env::{GuestPtr, MemAccess};
use wasmer::{Memory, StoreMut};

/// Adapter implementing MemAccess over wasmer MemoryView.
pub(crate) struct Sp1MemAccess<'s> {
    pub memory: Memory,
    pub store: StoreMut<'s>,
}

impl Sp1MemAccess<'_> {
    fn view(&self) -> wasmer::MemoryView<'_> {
        self.memory.view(&self.store)
    }
}

impl MemAccess for Sp1MemAccess<'_> {
    fn read_u8(&self, ptr: GuestPtr) -> u8 {
        let mut buf = [0u8; 1];
        self.view()
            .read(ptr.to_u64(), &mut buf)
            .expect("failed to read u8 from guest memory");
        buf[0]
    }

    fn read_u16(&self, ptr: GuestPtr) -> u16 {
        let mut buf = [0u8; 2];
        self.view()
            .read(ptr.to_u64(), &mut buf)
            .expect("failed to read u16 from guest memory");
        u16::from_le_bytes(buf)
    }

    fn read_u32(&self, ptr: GuestPtr) -> u32 {
        let mut buf = [0u8; 4];
        self.view()
            .read(ptr.to_u64(), &mut buf)
            .expect("failed to read u32 from guest memory");
        u32::from_le_bytes(buf)
    }

    fn read_u64(&self, ptr: GuestPtr) -> u64 {
        let mut buf = [0u8; 8];
        self.view()
            .read(ptr.to_u64(), &mut buf)
            .expect("failed to read u64 from guest memory");
        u64::from_le_bytes(buf)
    }

    fn write_u8(&mut self, ptr: GuestPtr, x: u8) {
        self.view()
            .write(ptr.to_u64(), &[x])
            .expect("failed to write u8 to guest memory");
    }

    fn write_u16(&mut self, ptr: GuestPtr, x: u16) {
        self.view()
            .write(ptr.to_u64(), &x.to_le_bytes())
            .expect("failed to write u16 to guest memory");
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) {
        self.view()
            .write(ptr.to_u64(), &x.to_le_bytes())
            .expect("failed to write u32 to guest memory");
    }

    fn write_u64(&mut self, ptr: GuestPtr, x: u64) {
        self.view()
            .write(ptr.to_u64(), &x.to_le_bytes())
            .expect("failed to write u64 to guest memory");
    }

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8> {
        let mut data = vec![0u8; len];
        self.view()
            .read(ptr.to_u64(), &mut data)
            .expect("failed to read slice from guest memory");
        data
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        let mut buf = [0u8; N];
        self.view()
            .read(ptr.to_u64(), &mut buf)
            .expect("failed to read fixed bytes from guest memory");
        buf
    }

    fn write_slice(&mut self, ptr: GuestPtr, data: &[u8]) {
        self.view()
            .write(ptr.to_u64(), data)
            .expect("failed to write slice to guest memory");
    }
}
