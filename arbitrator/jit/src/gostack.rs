// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::useless_transmute)]

use crate::{
    machine::{WasmEnv, WasmEnvMut},
    syscall::JsValue,
};

use ouroboros::self_referencing;
use rand_pcg::Pcg32;
use wasmer::{AsStoreRef, Memory, MemoryView, StoreRef, WasmPtr};

use std::collections::{BTreeSet, BinaryHeap};

#[self_referencing]
struct MemoryViewContainer {
    memory: Memory,
    #[borrows(memory)]
    #[covariant]
    view: MemoryView<'this>,
}

impl MemoryViewContainer {
    fn create(env: &WasmEnvMut<'_>) -> Self {
        // this func exists to properly constrain the closure's type
        fn closure<'a>(
            store: &'a StoreRef,
        ) -> impl (for<'b> FnOnce(&'b Memory) -> MemoryView<'b>) + 'a {
            move |memory: &Memory| memory.view(&store)
        }

        let store = env.as_store_ref();
        let memory = env.data().memory.clone().unwrap();
        let view_builder = closure(&store);
        MemoryViewContainerBuilder {
            memory,
            view_builder,
        }
        .build()
    }

    fn view(&self) -> &MemoryView {
        self.borrow_view()
    }
}

pub struct GoStack {
    start: u32,
    memory: MemoryViewContainer,
}

#[allow(dead_code)]
impl GoStack {
    pub fn new<'a, 'b: 'a>(start: u32, env: &'a mut WasmEnvMut<'b>) -> (Self, &'a mut WasmEnv) {
        let memory = MemoryViewContainer::create(env);
        let sp = Self { start, memory };
        (sp, env.data_mut())
    }

    pub fn simple(start: u32, env: &WasmEnvMut<'_>) -> Self {
        let memory = MemoryViewContainer::create(env);
        Self { start, memory }
    }

    fn view(&self) -> &MemoryView {
        self.memory.view()
    }

    /// Returns the memory size, in bytes.
    /// note: wasmer measures memory in 65536-byte pages.
    pub fn memory_size(&self) -> u64 {
        self.view().size().0 as u64 * 65536
    }

    pub fn relative_offset(&self, arg: u32) -> u32 {
        (arg + 1) * 8
    }

    fn offset(&self, arg: u32) -> u32 {
        self.start + self.relative_offset(arg)
    }

    pub fn read_u8(&self, arg: u32) -> u8 {
        self.read_u8_ptr(self.offset(arg))
    }

    pub fn read_u32(&self, arg: u32) -> u32 {
        self.read_u32_ptr(self.offset(arg))
    }

    pub fn read_u64(&self, arg: u32) -> u64 {
        self.read_u64_ptr(self.offset(arg))
    }

    pub fn read_u8_ptr(&self, ptr: u32) -> u8 {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_u32_ptr(&self, ptr: u32) -> u32 {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_u64_ptr(&self, ptr: u32) -> u64 {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn write_u8(&self, arg: u32, x: u8) {
        self.write_u8_ptr(self.offset(arg), x);
    }

    pub fn write_u32(&self, arg: u32, x: u32) {
        self.write_u32_ptr(self.offset(arg), x);
    }

    pub fn write_u64(&self, arg: u32, x: u64) {
        self.write_u64_ptr(self.offset(arg), x);
    }

    pub fn write_u8_ptr(&self, ptr: u32, x: u8) {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
    }

    pub fn write_u32_ptr(&self, ptr: u32, x: u32) {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
    }

    pub fn write_u64_ptr(&self, ptr: u32, x: u64) {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
    }

    pub fn read_slice(&self, ptr: u64, len: u64) -> Vec<u8> {
        u32::try_from(ptr).expect("Go pointer not a u32"); // kept for consistency
        let len = u32::try_from(len).expect("length isn't a u32") as usize;
        let mut data = vec![0; len];
        self.view().read(ptr, &mut data).expect("failed to read");
        data
    }

    pub fn write_slice(&self, ptr: u64, src: &[u8]) {
        u32::try_from(ptr).expect("Go pointer not a u32");
        self.view().write(ptr, src).unwrap();
    }

    pub fn read_value_slice(&self, mut ptr: u64, len: u64) -> Vec<JsValue> {
        let mut values = Vec::new();
        for _ in 0..len {
            let p = u32::try_from(ptr).expect("Go pointer not a u32");
            values.push(JsValue::new(self.read_u64_ptr(p)));
            ptr += 8;
        }
        values
    }
}

pub struct GoRuntimeState {
    /// An increasing clock used when Go asks for time, measured in nanoseconds
    pub time: u64,
    /// The amount of time advanced each check. Currently 10 milliseconds
    pub time_interval: u64,
    /// The state of Go's timeouts
    pub timeouts: TimeoutState,
    /// Deterministic source of random data
    pub rng: Pcg32,
}

impl Default for GoRuntimeState {
    fn default() -> Self {
        Self {
            time: 0,
            time_interval: 10_000_000,
            timeouts: TimeoutState::default(),
            rng: Pcg32::new(0xcafef00dd15ea5e5, 0xa02bdbf7bb3c0a7),
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
