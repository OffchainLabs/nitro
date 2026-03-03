// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use caller_env::wavmio::{WavmEnv, WavmState};
use caller_env::{ExecEnv, GuestPtr, MemAccess};
use rand::RngCore;
use wasmer::{FunctionEnvMut, Memory, StoreMut};

use crate::replay::CustomEnvData;

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
        self.view().read(ptr.to_u64(), &mut buf).unwrap();
        buf[0]
    }

    fn read_u16(&self, ptr: GuestPtr) -> u16 {
        let mut buf = [0u8; 2];
        self.view().read(ptr.to_u64(), &mut buf).unwrap();
        u16::from_le_bytes(buf)
    }

    fn read_u32(&self, ptr: GuestPtr) -> u32 {
        let mut buf = [0u8; 4];
        self.view().read(ptr.to_u64(), &mut buf).unwrap();
        u32::from_le_bytes(buf)
    }

    fn read_u64(&self, ptr: GuestPtr) -> u64 {
        let mut buf = [0u8; 8];
        self.view().read(ptr.to_u64(), &mut buf).unwrap();
        u64::from_le_bytes(buf)
    }

    fn write_u8(&mut self, ptr: GuestPtr, x: u8) {
        self.view().write(ptr.to_u64(), &[x]).unwrap();
    }

    fn write_u16(&mut self, ptr: GuestPtr, x: u16) {
        self.view().write(ptr.to_u64(), &x.to_le_bytes()).unwrap();
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) {
        self.view().write(ptr.to_u64(), &x.to_le_bytes()).unwrap();
    }

    fn write_u64(&mut self, ptr: GuestPtr, x: u64) {
        self.view().write(ptr.to_u64(), &x.to_le_bytes()).unwrap();
    }

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8> {
        let mut data = vec![0u8; len];
        self.view().read(ptr.to_u64(), &mut data).unwrap();
        data
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        let mut buf = [0u8; N];
        self.view().read(ptr.to_u64(), &mut buf).unwrap();
        buf
    }

    fn write_slice(&mut self, ptr: GuestPtr, data: &[u8]) {
        self.view().write(ptr.to_u64(), data).unwrap();
    }
}

/// Newtype wrapper to implement WavmState and ExecEnv over CustomEnvData.
pub(crate) struct Sp1State<'a>(pub &'a mut CustomEnvData);

impl ExecEnv for Sp1State<'_> {
    fn advance_time(&mut self, ns: u64) {
        self.0.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.0.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.0.pcg.next_u32()
    }

    fn print_string(&mut self, bytes: &[u8]) {
        crate::platform::print_string(2, bytes);
    }
}

impl WavmState for Sp1State<'_> {
    fn get_u64_global(&self, idx: usize) -> Option<u64> {
        self.0.input().small_globals.get(idx).copied()
    }

    fn set_u64_global(&mut self, idx: usize, val: u64) -> bool {
        match self.0.input_mut().small_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_bytes32_global(&self, idx: usize) -> Option<&[u8; 32]> {
        self.0.input().large_globals.get(idx)
    }

    fn set_bytes32_global(&mut self, idx: usize, val: [u8; 32]) -> bool {
        match self.0.input_mut().large_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_sequencer_message(&self, num: u64) -> Option<&[u8]> {
        self.0.input().sequencer_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_delayed_message(&self, num: u64) -> Option<&[u8]> {
        self.0.input().delayed_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_preimage(&self, preimage_type: u8, hash: &[u8; 32]) -> Option<&[u8]> {
        self.0
            .input()
            .preimages
            .get(&preimage_type)
            .and_then(|m| m.get(hash))
            .map(|v| v.as_slice())
    }
}

/// Newtype for implementing WavmEnv (orphan rule: FunctionEnvMut is foreign).
pub(crate) struct Sp1Wavm<'e>(pub FunctionEnvMut<'e, CustomEnvData>);

impl WavmEnv for Sp1Wavm<'_> {
    type Mem<'a> = Sp1MemAccess<'a> where Self: 'a;
    type State<'a> = Sp1State<'a> where Self: 'a;

    fn wavm_env(&mut self) -> (Sp1MemAccess<'_>, Sp1State<'_>) {
        let memory = self.0.data().memory.clone().unwrap();
        let (data, store) = self.0.data_and_store_mut();
        (Sp1MemAccess { memory, store }, Sp1State(data))
    }
}

/// Converts a wasmer `Ptr` (WasmPtr<u32>) to a caller-env `GuestPtr`.
pub(crate) fn gp(p: crate::Ptr) -> GuestPtr {
    GuestPtr(p.offset())
}
