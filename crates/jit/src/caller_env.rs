// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::{WasmEnv, WasmEnvMut};
use arbutil::{Bytes20, Bytes32, PreimageType};
use caller_env::wavmio::{WavmEnv, WavmState};
use caller_env::{ExecEnv, GuestPtr, MemAccess};
use rand::RngCore;
use std::mem::{self, MaybeUninit};
use wasmer::{Memory, MemoryView, StoreMut, WasmPtr};

pub struct JitMemAccess<'s> {
    pub memory: Memory,
    pub store: StoreMut<'s>,
}

pub struct JitExecEnv<'s> {
    pub wenv: &'s mut WasmEnv,
}

/// Newtype for implementing WavmEnv (orphan rule: FunctionEnvMut is foreign).
pub(crate) struct JitWavm<'e>(pub WasmEnvMut<'e>);

/// Newtype wrapping &mut WasmEnv to implement WavmState (orphan rule).
pub(crate) struct JitState<'a>(pub &'a mut WasmEnv);

/// Extracts (JitMemAccess, JitState) from a WasmEnvMut in place.
pub(crate) fn jit_env<'a>(env: &'a mut WasmEnvMut) -> (JitMemAccess<'a>, JitState<'a>) {
    let memory = env.data().memory.clone().unwrap();
    let (wenv, store) = env.data_and_store_mut();
    (JitMemAccess { memory, store }, JitState(wenv))
}

impl WavmEnv for JitWavm<'_> {
    type Mem<'a> = JitMemAccess<'a> where Self: 'a;
    type State<'a> = JitState<'a> where Self: 'a;

    fn wavm_env(&mut self) -> (JitMemAccess<'_>, JitState<'_>) {
        jit_env(&mut self.0)
    }
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

impl ExecEnv for JitExecEnv<'_> {
    fn advance_time(&mut self, ns: u64) {
        self.wenv.go_state.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.wenv.go_state.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.wenv.go_state.rng.next_u32()
    }

    fn print_string(&mut self, bytes: &[u8]) {
        match String::from_utf8(bytes.to_vec()) {
            Ok(s) => eprintln!("JIT: WASM says: {s}"), // TODO: this adds too many newlines since go calls this in chunks
            Err(e) => {
                let bytes = e.as_bytes();
                eprintln!("Go string {} is not valid utf8: {e:?}", hex::encode(bytes));
            }
        }
    }
}

impl WavmState for JitState<'_> {
    fn get_u64_global(&self, idx: usize) -> Option<u64> {
        self.0.small_globals.get(idx).copied()
    }

    fn set_u64_global(&mut self, idx: usize, val: u64) -> bool {
        match self.0.small_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_bytes32_global(&self, idx: usize) -> Option<&[u8; 32]> {
        self.0.large_globals.get(idx).map(|b| &b.0)
    }

    fn set_bytes32_global(&mut self, idx: usize, val: [u8; 32]) -> bool {
        match self.0.large_globals.get_mut(idx) {
            Some(g) => {
                *g = val.into();
                true
            }
            None => false,
        }
    }

    fn get_sequencer_message(&self, num: u64) -> Option<&[u8]> {
        self.0.sequencer_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_delayed_message(&self, num: u64) -> Option<&[u8]> {
        self.0.delayed_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_preimage(&self, preimage_type: u8, hash: &[u8; 32]) -> Option<&[u8]> {
        let pt: PreimageType = preimage_type.try_into().ok()?;
        self.0
            .preimages
            .get(&pt)
            .and_then(|m| m.get(&Bytes32::from(*hash)))
            .map(|v| v.as_slice())
    }
}
