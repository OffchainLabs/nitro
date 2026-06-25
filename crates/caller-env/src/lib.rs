// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

use alloc::vec::Vec;

pub use guest_ptr::GuestPtr;
use rand_pcg::Pcg32;
pub use wasip1_stub::Errno;

mod guest_ptr;
pub mod wavmio;

#[cfg(feature = "static_caller")]
pub mod static_caller;

#[cfg(feature = "wasmer_traits")]
pub mod wasmer_traits;

/// Generates a wasmer host function that owns `FunctionEnvMut<T>`, extracts memory
/// via `HasMemory::memory()`, and delegates to the `super::` function of the same
/// name which takes `&mut impl MemAccess` as its first argument.
#[cfg(feature = "wasmer_traits")]
macro_rules! host_fn {
    ($(#[$attr:meta])* fn $name:ident($($arg:ident : $ty:ty),*) $(-> $ret:ty)?) => {
        $(#[$attr])*
        pub fn $name<T: $crate::wasmer_traits::HasMemory + Send + 'static>(
            mut ctx: wasmer::FunctionEnvMut<T>,
            $($arg: $ty,)*
        ) $(-> $ret)? {
            let (data, store) = ctx.data_and_store_mut();
            let memory = data.memory();
            super::$name(&mut $crate::wasmer_traits::WasmerMem::new(memory, store), $($arg,)*)
        }
    };
}

/// Like [`host_fn!`] but for WASI stubs that take both `&mut impl MemAccess` and
/// `&mut impl ExecEnv`. The env data type `T` must implement both `HasMemory` and `ExecEnv`.
#[cfg(feature = "wasmer_traits")]
macro_rules! host_fn_exec {
    (fn $name:ident($($arg:ident : $ty:ty),*)) => {
        pub fn $name<T: $crate::wasmer_traits::HasMemory + $crate::ExecEnv + Send + 'static>(
            mut ctx: wasmer::FunctionEnvMut<T>,
            $($arg: $ty,)*
        ) -> $crate::wasip1_stub::Errno {
            let (data, store) = ctx.data_and_store_mut();
            let memory = data.memory();
            super::$name(&mut $crate::wasmer_traits::WasmerMem::new(memory, store), data, $($arg,)*)
        }
    };
}

#[cfg(feature = "brotli")]
pub mod brotli;

pub mod arbcrypto;
pub mod wasip1_stub;

/// Access Guest memory.
pub trait MemAccess {
    fn read_u8(&self, ptr: GuestPtr) -> u8;

    fn read_u16(&self, ptr: GuestPtr) -> u16;

    fn read_u32(&self, ptr: GuestPtr) -> u32;

    fn read_u64(&self, ptr: GuestPtr) -> u64;

    fn write_u8(&mut self, ptr: GuestPtr, x: u8);

    fn write_u16(&mut self, ptr: GuestPtr, x: u16);

    fn write_u32(&mut self, ptr: GuestPtr, x: u32);

    fn write_u64(&mut self, ptr: GuestPtr, x: u64);

    fn read_slice(&self, ptr: GuestPtr, len: usize) -> Vec<u8>;

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> [u8; N];

    fn write_slice(&mut self, ptr: GuestPtr, data: &[u8]);
}

/// Update the Host environment.
pub trait ExecEnv {
    fn advance_time(&mut self, ns: u64);

    fn get_time(&self) -> u64;

    fn next_rand_u32(&mut self) -> u32;

    fn print_string(&mut self, message: &[u8]);
}

#[derive(Clone, PartialEq, Eq)]
pub struct GoRuntimeState {
    /// An increasing clock used when Go asks for time, measured in nanoseconds.
    pub time: u64,
    /// Deterministic source of random data.
    pub rng: Pcg32,
}

impl Default for GoRuntimeState {
    fn default() -> Self {
        Self {
            time: 0,
            rng: create_pcg(),
        }
    }
}

/// Initializes a deterministic, pseudo-random number generator with a fixed seed.
fn create_pcg() -> Pcg32 {
    const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
    const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;
    Pcg32::new(PCG_INIT_STATE, PCG_INIT_STREAM)
}
