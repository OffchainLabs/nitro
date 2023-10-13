// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::useless_transmute)]

use crate::{
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
    syscall::JsValue,
    wavmio::{Bytes20, Bytes32},
};
use arbutil::Color;
use ouroboros::self_referencing;
use rand_pcg::Pcg32;
use std::{
    collections::{BTreeSet, BinaryHeap},
    fmt::Debug,
};
use wasmer::{AsStoreRef, Memory, MemoryView, StoreMut, StoreRef, WasmPtr};

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
    sp: u32,
    top: u32,
    memory: MemoryViewContainer,
}

#[allow(dead_code)]
impl GoStack {
    pub fn new<'a, 'b: 'a>(start: u32, env: &'a mut WasmEnvMut<'b>) -> (Self, &'a mut WasmEnv) {
        let sp = Self::simple(start, env);
        (sp, env.data_mut())
    }

    pub fn new_with_store<'a, 'b: 'a>(
        start: u32,
        env: &'a mut WasmEnvMut<'b>,
    ) -> (Self, &'a mut WasmEnv, StoreMut<'a>) {
        let sp = Self::simple(start, env);
        let (env, store) = env.data_and_store_mut();
        (sp, env, store)
    }

    pub fn simple(sp: u32, env: &WasmEnvMut<'_>) -> Self {
        let top = sp + 8;
        let memory = MemoryViewContainer::create(env);
        Self { sp, top, memory }
    }

    fn view(&self) -> &MemoryView {
        self.memory.view()
    }

    /// Returns the memory size, in bytes.
    /// note: wasmer measures memory in 65536-byte pages.
    pub fn memory_size(&self) -> u64 {
        self.view().size().0 as u64 * 65536
    }

    fn advance(&mut self, bytes: usize) -> u32 {
        let before = self.top;
        self.top += bytes as u32;
        before
    }

    pub fn read_u8(&mut self) -> u8 {
        let ptr = self.advance(1);
        self.read_u8_raw(ptr)
    }

    pub fn read_u16(&mut self) -> u16 {
        let ptr = self.advance(2);
        self.read_u16_raw(ptr)
    }

    pub fn read_u32(&mut self) -> u32 {
        let ptr = self.advance(4);
        self.read_u32_raw(ptr)
    }

    pub fn read_u64(&mut self) -> u64 {
        let ptr = self.advance(8);
        self.read_u64_raw(ptr)
    }

    pub fn read_u8_raw(&self, ptr: u32) -> u8 {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_u16_raw(&self, ptr: u32) -> u16 {
        let ptr: WasmPtr<u16> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_u32_raw(&self, ptr: u32) -> u32 {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_u64_raw(&self, ptr: u32) -> u64 {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(self.view()).read().unwrap()
    }

    pub fn read_ptr<T>(&mut self) -> *const T {
        self.read_u64() as *const T
    }

    pub fn read_ptr_mut<T>(&mut self) -> *mut T {
        self.read_u64() as *mut T
    }

    pub unsafe fn read_ref<'a, 'b, T>(&'a mut self) -> &'b T {
        &*self.read_ptr()
    }

    /// TODO: replace `unbox` with a safe id-based API
    pub fn unbox<T>(&mut self) -> T {
        let ptr: *mut T = self.read_ptr_mut();
        unsafe { *Box::from_raw(ptr) }
    }

    /// TODO: replace `unbox_option` with a safe id-based API
    pub fn unbox_option<T>(&mut self) -> Option<T> {
        let ptr: *mut T = self.read_ptr_mut();
        (!ptr.is_null()).then(|| unsafe { *Box::from_raw(ptr) })
    }

    pub fn write_u8(&mut self, x: u8) -> &mut Self {
        let ptr = self.advance(1);
        self.write_u8_raw(ptr, x)
    }

    pub fn write_u16(&mut self, x: u16) -> &mut Self {
        let ptr = self.advance(2);
        self.write_u16_raw(ptr, x)
    }

    pub fn write_u32(&mut self, x: u32) -> &mut Self {
        let ptr = self.advance(4);
        self.write_u32_raw(ptr, x)
    }

    pub fn write_u64(&mut self, x: u64) -> &mut Self {
        let ptr = self.advance(8);
        self.write_u64_raw(ptr, x)
    }

    pub fn write_u8_raw(&mut self, ptr: u32, x: u8) -> &mut Self {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
        self
    }

    pub fn write_u16_raw(&mut self, ptr: u32, x: u16) -> &mut Self {
        let ptr: WasmPtr<u16> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
        self
    }

    pub fn write_u32_raw(&mut self, ptr: u32, x: u32) -> &mut Self {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
        self
    }

    pub fn write_u64_raw(&mut self, ptr: u32, x: u64) -> &mut Self {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(self.view()).write(x).unwrap();
        self
    }

    pub fn write_ptr<T>(&mut self, ptr: *const T) -> &mut Self {
        self.write_u64(ptr as u64)
    }

    pub fn write_nullptr(&mut self) -> &mut Self {
        self.write_ptr(std::ptr::null::<u8>())
    }

    pub fn skip_u8(&mut self) -> &mut Self {
        self.advance(1);
        self
    }

    pub fn skip_u16(&mut self) -> &mut Self {
        self.advance(2);
        self
    }

    pub fn skip_u32(&mut self) -> &mut Self {
        self.advance(4);
        self
    }

    pub fn skip_u64(&mut self) -> &mut Self {
        self.advance(8);
        self
    }

    pub fn skip_space(&mut self) -> &mut Self {
        let space = 8 - (self.top - self.sp) % 8;
        self.advance(space as usize);
        self
    }

    pub fn read_slice(&self, ptr: u64, len: u64) -> Vec<u8> {
        u32::try_from(ptr).expect("Go pointer not a u32"); // kept for consistency
        let len = u32::try_from(len).expect("length isn't a u32") as usize;
        let mut data = vec![0; len];
        self.view().read(ptr, &mut data).expect("failed to read");
        data
    }

    pub fn write_slice<T: TryInto<u32>>(&self, ptr: T, src: &[u8]) {
        let ptr: u32 = ptr.try_into().map_err(|_| "Go pointer not a u32").unwrap();
        self.view().write(ptr.into(), src).unwrap();
    }

    pub fn read_value_slice(&self, mut ptr: u64, len: u64) -> Vec<JsValue> {
        let mut values = Vec::new();
        for _ in 0..len {
            let p = u32::try_from(ptr).expect("Go pointer not a u32");
            values.push(JsValue::new(self.read_u64_raw(p)));
            ptr += 8;
        }
        values
    }

    pub fn read_bool32(&mut self) -> bool {
        self.read_u32() != 0
    }

    pub fn read_go_ptr(&mut self) -> u32 {
        self.read_u64().try_into().expect("go pointer doesn't fit")
    }

    pub fn read_bytes20(&mut self) -> Bytes20 {
        let ptr = self.read_go_ptr().into();
        self.read_slice(ptr, 20).try_into().unwrap()
    }

    pub fn read_bytes32(&mut self) -> Bytes32 {
        let ptr = self.read_go_ptr().into();
        self.read_slice(ptr, 32).try_into().unwrap()
    }

    pub fn read_go_slice(&mut self) -> (u64, u64) {
        let ptr = self.read_u64();
        let len = self.read_u64();
        self.skip_u64(); // skip the slice's capacity
        (ptr, len)
    }

    pub fn read_go_slice_owned(&mut self) -> Vec<u8> {
        let (ptr, len) = self.read_go_slice();
        self.read_slice(ptr, len)
    }

    pub fn read_js_string(&mut self) -> Vec<u8> {
        let ptr = self.read_u64();
        let len = self.read_u64();
        self.read_slice(ptr, len)
    }

    /// Resumes the Go runtime, updating the stack pointer.
    ///
    /// # Safety
    ///
    /// Caller must cut lifetimes before this call.
    pub unsafe fn resume(&mut self, env: &mut WasmEnv, store: &mut StoreMut) -> MaybeEscape {
        let Some(resume) = &env.exports.resume else {
            return Escape::failure(format!("wasmer failed to bind {}", "resume".red()));
        };
        let Some(get_stack_pointer) = &env.exports.get_stack_pointer else {
            return Escape::failure(format!("wasmer failed to bind {}", "getsp".red()));
        };

        // save our progress from the stack pointer
        let saved = self.top - self.sp;

        // recursively call into wasmer (reentrant)
        resume.call(store)?;

        // recover the stack pointer
        let pointer = get_stack_pointer.call(store)? as u32;
        self.sp = pointer;
        self.top = pointer + saved;
        Ok(())
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

#[test]
#[allow(clippy::identity_op, clippy::field_reassign_with_default)]
fn test_sp() -> eyre::Result<()> {
    use prover::programs::prelude::CompileConfig;
    use wasmer::{FunctionEnv, MemoryType};

    let mut store = CompileConfig::default().store();
    let mut env = WasmEnv::default();
    env.memory = Some(Memory::new(&mut store, MemoryType::new(0, None, false))?);
    let env = FunctionEnv::new(&mut store, env);

    let mut sp = GoStack::simple(0, &env.into_mut(&mut store));
    assert_eq!(sp.advance(3), 8 + 0);
    assert_eq!(sp.advance(2), 8 + 3);
    assert_eq!(sp.skip_space().top, 8 + 8);
    assert_eq!(sp.skip_space().top, 8 + 16);
    assert_eq!(sp.skip_u32().skip_space().top, 8 + 24);
    Ok(())
}
