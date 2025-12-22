pub mod imports;
pub mod platform;
pub mod replay;
pub mod stylus;

use arbutil::{
    Bytes20, Bytes32,
    evm::api::{Gas, Ink},
};
use prover::programs::config::{CompileConfig, StylusConfig};
use std::io;
use std::mem::{self, MaybeUninit};
use std::ptr::NonNull;
use thiserror::Error;
use wasmer::{MemoryAccessError, MemoryView, WasmPtr};
use wasmer_types::RawValue;
use wasmer_vm::VMGlobalDefinition;

pub use crate::replay::run;

pub const STACK_SIZE: usize = 1024 * 1024;

// nitro uses 32-bit memory space
pub(crate) type Ptr = WasmPtr<u32>;

pub(crate) fn read_slice(ptr: Ptr, len: usize, memory: &MemoryView) -> Result<Vec<u8>, Escape> {
    let mut data: Vec<MaybeUninit<u8>> = Vec::with_capacity(len);
    // SAFETY: read_uninit fills all available space
    Ok(unsafe {
        data.set_len(len);
        memory.read_uninit(ptr.offset() as u64, &mut data)?;
        mem::transmute::<Vec<MaybeUninit<u8>>, Vec<u8>>(data)
    })
}

pub(crate) fn read_bytes20(ptr: Ptr, memory: &MemoryView) -> Result<Bytes20, Escape> {
    read_slice(ptr, 20, memory).map(|data| data.try_into().unwrap())
}

pub(crate) fn read_bytes32(ptr: Ptr, memory: &MemoryView) -> Result<Bytes32, Escape> {
    read_slice(ptr, 32, memory).map(|data| data.try_into().unwrap())
}

fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    use std::mem::MaybeUninit;
    use tiny_keccak::{Hasher, Keccak};

    let mut output = MaybeUninit::<[u8; 32]>::uninit();
    let mut hasher = Keccak::v256();
    hasher.update(preimage.as_ref());

    // SAFETY: finalize() writes 32 bytes
    unsafe {
        hasher.finalize(&mut *output.as_mut_ptr());
        output.assume_init()
    }
}

pub type MaybeEscape = Result<(), Escape>;

#[derive(Error, Debug)]
pub enum Escape {
    #[error("failed to access memory: `{0}`")]
    Memory(MemoryAccessError),
    #[error("internal error: `{0}`")]
    Internal(String),
    #[error("logic error: `{0}`")]
    Logical(String),
    #[error("out of ink")]
    OutOfInk,
    #[error("exit early: `{0}`")]
    Exit(u32),
}

impl Escape {
    pub fn logical<T, S: std::convert::AsRef<str>>(message: S) -> Result<T, Escape> {
        Err(Self::Logical(message.as_ref().to_string()))
    }
}

impl From<String> for Escape {
    fn from(err: String) -> Self {
        Self::Internal(err)
    }
}

impl From<eyre::ErrReport> for Escape {
    fn from(err: eyre::ErrReport) -> Self {
        Self::Internal(err.to_string())
    }
}

impl From<io::Error> for Escape {
    fn from(err: io::Error) -> Self {
        Self::Internal(format!("[io error]: {err:?}"))
    }
}

impl From<prover::programs::meter::OutOfInkError> for Escape {
    fn from(_: prover::programs::meter::OutOfInkError) -> Self {
        Self::OutOfInk
    }
}

impl From<MemoryAccessError> for Escape {
    fn from(err: MemoryAccessError) -> Self {
        Self::Memory(err)
    }
}

// Below are some data structures that do not belong to prover / arbutil,
// but we don't want to pull in full crate anyway. As a result, they are
// for now copied over.

pub struct JitConfig {
    pub stylus: StylusConfig,
    pub compile: CompileConfig,
}

pub struct CallInputs {
    pub contract: Bytes20,
    pub input: Vec<u8>,
    pub gas_left: Gas,
    pub gas_req: Gas,
    pub value: Option<Bytes32>,
}

#[derive(Clone, Copy, Debug)]
pub struct MeterData {
    /// The amount of ink left
    pub ink_left: NonNull<VMGlobalDefinition>,
    /// Whether the instance has run out of ink
    pub ink_status: NonNull<VMGlobalDefinition>,
}

impl MeterData {
    pub fn ink(&self) -> Ink {
        Ink(unsafe { self.ink_left.as_ref().val.u64 })
    }

    pub fn status(&self) -> u32 {
        unsafe { self.ink_status.as_ref().val.u32 }
    }

    pub fn set_ink(&mut self, ink: Ink) {
        unsafe { self.ink_left.as_mut().val = RawValue { u64: ink.0 } }
    }

    pub fn set_status(&mut self, status: u32) {
        unsafe { self.ink_status.as_mut().val = RawValue { u32: status } }
    }
}

/// The data we're pointing to is owned by the `NativeInstance`.
/// These are simple integers whose lifetime is that of the instance.
/// Stylus is also single-threaded.
unsafe impl Send for MeterData {}
