// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::machine::{WasmEnv, WasmEnvMut};
use arbutil::{Bytes20, Bytes32};
use caller_env::{ExecEnv, GuestPtr, MemAccess};
use rand::RngCore;
use rand_pcg::Pcg32;
use std::{
    collections::{BTreeSet, BinaryHeap},
    fmt::Debug,
};
use wasmer::{Memory, MemoryView, StoreMut, WasmPtr};

pub struct JitMemAccess<'s> {
    pub memory: Memory,
    pub store: StoreMut<'s>,
}

pub struct JitExecEnv<'s> {
    pub wenv: &'s mut WasmEnv,
}

pub fn jit_env<'s>(env: &'s mut WasmEnvMut) -> (JitMemAccess<'s>, JitExecEnv<'s>) {
    let memory = env.data().memory.clone().unwrap();
    let (wenv, store) = env.data_and_store_mut();
    (JitMemAccess { memory, store }, JitExecEnv { wenv })
}

#[allow(dead_code)]
impl<'s> JitMemAccess<'s> {
    /// Returns the memory size, in bytes.
    /// note: wasmer measures memory in 65536-byte pages.
    fn memory_size(&self) -> u64 {
        self.view().size().0 as u64 * 65536
    }

    fn view(&self) -> MemoryView {
        self.memory.view(&self.store)
    }

    pub fn write_bytes20(&mut self, ptr: GuestPtr, val: Bytes20) {
        self.write_slice(ptr, val.as_slice())
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

    pub fn read_string(&mut self, ptr: GuestPtr, len: u32) -> String {
        let bytes = self.read_slice(ptr, len as usize);
        match String::from_utf8(bytes) {
            Ok(s) => s,
            Err(e) => {
                let bytes = e.as_bytes();
                eprintln!("Go string {} is not valid utf8: {e:?}", hex::encode(bytes));
                String::from_utf8_lossy(bytes).into_owned()
            }
        }
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
        let mut data = Vec::with_capacity(len);
        unsafe {
            self.view().read(ptr.into(), &mut data).expect("bad read");
            data.set_len(len);
        }
        data
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N] {
        self.read_slice(ptr, N).try_into().unwrap()
    }

    fn write_slice(&mut self, ptr: GuestPtr, src: &[u8]) {
        self.view().write(ptr.into(), src).unwrap();
    }
}

impl ExecEnv for JitExecEnv<'_> {
    fn print_string(&mut self, bytes: &[u8]) {
        match String::from_utf8(bytes.to_vec()) {
            Ok(s) => eprintln!("JIT: WASM says: {s}"),
            Err(e) => {
                let bytes = e.as_bytes();
                eprintln!("Go string {} is not valid utf8: {e:?}", hex::encode(bytes));
            }
        }
    }

    fn get_time(&self) -> u64 {
        self.wenv.go_state.time
    }

    fn advance_time(&mut self, delta: u64) {
        self.wenv.go_state.time += delta
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.wenv.go_state.rng.next_u32()
    }
}

pub struct GoRuntimeState {
    /// An increasing clock used when Go asks for time, measured in nanoseconds
    pub time: u64,
    /// Deterministic source of random data
    pub rng: Pcg32,
}

impl Default for GoRuntimeState {
    fn default() -> Self {
        Self {
            time: 0,
            rng: caller_env::create_pcg(),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TimeoutInfo {
    pub time: u64,
    pub id: u32,
}

impl Ord for TimeoutInfo {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        other
            .time
            .cmp(&self.time)
            .then_with(|| other.id.cmp(&self.id))
    }
}

impl PartialOrd for TimeoutInfo {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

#[derive(Default, Debug)]
pub struct TimeoutState {
    /// Contains tuples of (time, id)
    pub times: BinaryHeap<TimeoutInfo>,
    pub pending_ids: BTreeSet<u32>,
    pub next_id: u32,
}
