// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{
    evm::{api::EvmApi, EvmData},
    pricing, Bytes20, Bytes32, Color,
};
use derivative::Derivative;
use eyre::{eyre, ErrReport};
use prover::programs::{config::PricingParams, meter::OutOfInkError, prelude::*};
use std::{
    fmt::{Debug, Display},
    io,
    mem::MaybeUninit,
    ops::{Deref, DerefMut},
    ptr::NonNull,
};
use thiserror::Error;
use wasmer::{
    AsStoreRef, FunctionEnvMut, Memory, MemoryAccessError, MemoryView, Pages, StoreMut, WasmPtr,
};
use wasmer_types::RawValue;
use wasmer_vm::VMGlobalDefinition;

pub type WasmEnvMut<'a, E> = FunctionEnvMut<'a, WasmEnv<E>>;

#[derive(Derivative)]
#[derivative(Debug)]
pub struct WasmEnv<E: EvmApi> {
    /// The instance's arguments
    #[derivative(Debug(format_with = "arbutil::format::hex_fmt"))]
    pub args: Vec<u8>,
    /// The instance's return data
    #[derivative(Debug(format_with = "arbutil::format::hex_fmt"))]
    pub outs: Vec<u8>,
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Mechanism for accessing metering-specific global state
    pub meter: Option<MeterData>,
    /// Mechanism for reading and writing permanent storage, and doing calls
    pub evm_api: E,
    /// Mechanism for reading EVM context data
    pub evm_data: EvmData,
    /// The compile time config
    pub compile: CompileConfig,
    /// The runtime config
    pub config: Option<StylusConfig>,
}

impl<E: EvmApi> WasmEnv<E> {
    pub fn new(
        compile: CompileConfig,
        config: Option<StylusConfig>,
        evm_api: E,
        evm_data: EvmData,
    ) -> Self {
        Self {
            compile,
            config,
            evm_api,
            evm_data,
            args: vec![],
            outs: vec![],
            memory: None,
            meter: None,
        }
    }

    pub fn start<'a>(
        env: &'a mut WasmEnvMut<'_, E>,
        ink: u64,
    ) -> Result<HostioInfo<'a, E>, Escape> {
        let mut info = Self::start_free(env)?;
        info.buy_ink(pricing::HOSTIO_INK + ink)?;
        Ok(info)
    }

    pub fn start_free<'a>(env: &'a mut WasmEnvMut<'_, E>) -> Result<HostioInfo<'a, E>, Escape> {
        let (env, store) = env.data_and_store_mut();
        let memory = env.memory.clone().unwrap();
        let mut info = HostioInfo {
            env,
            memory,
            store,
            start_ink: 0,
        };
        if info.env.evm_data.tracing {
            info.start_ink = info.ink_ready()?;
        }
        Ok(info)
    }

    pub fn meter(&mut self) -> &mut MeterData {
        self.meter.as_mut().expect("not metered")
    }

    pub fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }
}

#[derive(Clone, Copy, Debug)]
pub struct MeterData {
    /// The amount of ink left
    pub ink_left: NonNull<VMGlobalDefinition>,
    /// Whether the instance has run out of ink
    pub ink_status: NonNull<VMGlobalDefinition>,
}

impl MeterData {
    pub fn ink(&self) -> u64 {
        unsafe { self.ink_left.as_ref().val.u64 }
    }

    pub fn status(&self) -> u32 {
        unsafe { self.ink_status.as_ref().val.u32 }
    }

    pub fn set_ink(&mut self, ink: u64) {
        unsafe { self.ink_left.as_mut().val = RawValue { u64: ink } }
    }

    pub fn set_status(&mut self, status: u32) {
        unsafe { self.ink_status.as_mut().val = RawValue { u32: status } }
    }
}

/// The data we're pointing to is owned by the `NativeInstance`.
/// These are simple integers whose lifetime is that of the instance.
/// Stylus is also single-threaded.
unsafe impl Send for MeterData {}

pub struct HostioInfo<'a, E: EvmApi> {
    pub env: &'a mut WasmEnv<E>,
    pub memory: Memory,
    pub store: StoreMut<'a>,
    pub start_ink: u64,
}

impl<'a, E: EvmApi> HostioInfo<'a, E> {
    pub fn config(&self) -> StylusConfig {
        self.config.expect("no config")
    }

    pub fn pricing(&self) -> PricingParams {
        self.config().pricing
    }

    pub fn view(&self) -> MemoryView {
        self.memory.view(&self.store.as_store_ref())
    }

    pub fn memory_size(&self) -> Pages {
        self.memory.ty(&self.store).minimum
    }

    pub fn _write_u8(&mut self, ptr: u32, x: u8) -> Result<&mut Self, MemoryAccessError> {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x)?;
        Ok(self)
    }

    pub fn write_u32(&mut self, ptr: u32, x: u32) -> Result<&mut Self, MemoryAccessError> {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x)?;
        Ok(self)
    }

    pub fn write_u64(&mut self, ptr: u32, x: u64) -> Result<&mut Self, MemoryAccessError> {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x)?;
        Ok(self)
    }

    pub fn read_slice(&self, ptr: u32, len: u32) -> Result<Vec<u8>, MemoryAccessError> {
        let mut data = vec![0; len as usize];
        self.view().read(ptr.into(), &mut data)?;
        Ok(data)
    }

    // TODO: use the unstable array_assum_init
    pub fn read_fixed<const N: usize>(&self, ptr: u32) -> Result<[u8; N], MemoryAccessError> {
        let mut data = [MaybeUninit::uninit(); N];
        self.view().read_uninit(ptr.into(), &mut data)?;
        Ok(data.map(|x| unsafe { x.assume_init() }))
    }

    pub fn read_bytes20(&self, ptr: u32) -> eyre::Result<Bytes20> {
        let data = self.read_fixed(ptr)?;
        Ok(data.into())
    }

    pub fn read_bytes32(&self, ptr: u32) -> eyre::Result<Bytes32> {
        let data = self.read_fixed(ptr)?;
        Ok(data.into())
    }

    pub fn write_slice(&self, ptr: u32, src: &[u8]) -> Result<(), MemoryAccessError> {
        self.view().write(ptr.into(), src)
    }

    pub fn write_bytes20(&self, ptr: u32, src: Bytes20) -> eyre::Result<()> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }

    pub fn write_bytes32(&self, ptr: u32, src: Bytes32) -> eyre::Result<()> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }

    pub fn trace(&self, name: &str, args: &[u8], outs: &[u8], start_ink: u64, end_ink: u64) {
        self.evm_api
            .capture_hostio(name, args, outs, start_ink, end_ink);
    }
}

impl<'a, E: EvmApi> MeteredMachine for HostioInfo<'a, E> {
    fn ink_left(&mut self) -> MachineMeter {
        let vm = self.env.meter();
        match vm.status() {
            0_u32 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let vm = self.env.meter();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }
}

impl<'a, E: EvmApi> GasMeteredMachine for HostioInfo<'a, E> {
    fn pricing(&mut self) -> PricingParams {
        self.config().pricing
    }
}

impl<'a, E: EvmApi> Deref for HostioInfo<'a, E> {
    type Target = WasmEnv<E>;

    fn deref(&self) -> &Self::Target {
        self.env
    }
}

impl<'a, E: EvmApi> DerefMut for HostioInfo<'a, E> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.env
    }
}

pub type MaybeEscape = Result<(), Escape>;

#[derive(Error, Debug)]
pub enum Escape {
    #[error("failed to access memory: `{0}`")]
    Memory(MemoryAccessError),
    #[error("internal error: `{0}`")]
    Internal(ErrReport),
    #[error("Logic error: `{0}`")]
    Logical(ErrReport),
    #[error("out of ink")]
    OutOfInk,
}

impl Escape {
    pub fn _internal<T>(error: &'static str) -> Result<T, Escape> {
        Err(Self::Internal(eyre!(error)))
    }

    pub fn logical<T>(error: &'static str) -> Result<T, Escape> {
        Err(Self::Logical(eyre!(error)))
    }

    pub fn out_of_ink<T>() -> Result<T, Escape> {
        Err(Self::OutOfInk)
    }
}

impl From<OutOfInkError> for Escape {
    fn from(_: OutOfInkError) -> Self {
        Self::OutOfInk
    }
}

impl From<MemoryAccessError> for Escape {
    fn from(err: MemoryAccessError) -> Self {
        Self::Memory(err)
    }
}

impl From<io::Error> for Escape {
    fn from(err: io::Error) -> Self {
        Self::Internal(eyre!(err))
    }
}

impl From<ErrReport> for Escape {
    fn from(err: ErrReport) -> Self {
        Self::Internal(err)
    }
}
