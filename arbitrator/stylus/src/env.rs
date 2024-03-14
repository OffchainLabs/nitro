// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{
    evm::{
        api::{DataReader, EvmApi},
        EvmData,
    },
    pricing,
};
use caller_env::GuestPtr;
use derivative::Derivative;
use eyre::{eyre, ErrReport};
use prover::programs::{config::PricingParams, meter::OutOfInkError, prelude::*};
use std::{
    fmt::Debug,
    io,
    marker::PhantomData,
    mem::MaybeUninit,
    ops::{Deref, DerefMut},
    ptr::NonNull,
};
use thiserror::Error;
use wasmer::{FunctionEnvMut, Memory, MemoryAccessError, MemoryView, Pages, StoreMut};
use wasmer_types::RawValue;
use wasmer_vm::VMGlobalDefinition;

pub type WasmEnvMut<'a, D, E> = FunctionEnvMut<'a, WasmEnv<D, E>>;

#[derive(Derivative)]
#[derivative(Debug)]
pub struct WasmEnv<D: DataReader, E: EvmApi<D>> {
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
    // Using the unused generic parameter D in a PhantomData field
    _data_reader_marker: PhantomData<D>,
}

impl<D: DataReader, E: EvmApi<D>> WasmEnv<D, E> {
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
            _data_reader_marker: PhantomData,
        }
    }

    pub fn start<'a>(
        env: &'a mut WasmEnvMut<'_, D, E>,
        ink: u64,
    ) -> Result<HostioInfo<'a, D, E>, Escape> {
        let mut info = Self::program(env)?;
        info.buy_ink(pricing::HOSTIO_INK + ink)?;
        Ok(info)
    }

    pub fn program<'a>(env: &'a mut WasmEnvMut<'_, D, E>) -> Result<HostioInfo<'a, D, E>, Escape> {
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

    pub fn meter_mut(&mut self) -> &mut MeterData {
        self.meter.as_mut().expect("not metered")
    }

    pub fn meter(&self) -> &MeterData {
        self.meter.as_ref().expect("not metered")
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

pub struct HostioInfo<'a, D: DataReader, E: EvmApi<D>> {
    pub env: &'a mut WasmEnv<D, E>,
    pub memory: Memory,
    pub store: StoreMut<'a>,
    pub start_ink: u64,
}

impl<'a, D: DataReader, E: EvmApi<D>> HostioInfo<'a, D, E> {
    pub fn config(&self) -> StylusConfig {
        self.config.expect("no config")
    }

    pub fn pricing(&self) -> PricingParams {
        self.config().pricing
    }

    pub fn view(&self) -> MemoryView {
        self.memory.view(&self.store)
    }

    pub fn memory_size(&self) -> Pages {
        self.memory.ty(&self.store).minimum
    }

    // TODO: use the unstable array_assum_init
    pub fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> Result<[u8; N], MemoryAccessError> {
        let mut data = [MaybeUninit::uninit(); N];
        self.view().read_uninit(ptr.into(), &mut data)?;
        Ok(data.map(|x| unsafe { x.assume_init() }))
    }
}

impl<'a, D: DataReader, E: EvmApi<D>> MeteredMachine for HostioInfo<'a, D, E> {
    fn ink_left(&self) -> MachineMeter {
        let vm = self.env.meter();
        match vm.status() {
            0_u32 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let vm = self.env.meter_mut();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }
}

impl<'a, D: DataReader, E: EvmApi<D>> GasMeteredMachine for HostioInfo<'a, D, E> {
    fn pricing(&self) -> PricingParams {
        self.config().pricing
    }
}

impl<'a, D: DataReader, E: EvmApi<D>> Deref for HostioInfo<'a, D, E> {
    type Target = WasmEnv<D, E>;

    fn deref(&self) -> &Self::Target {
        self.env
    }
}

impl<'a, D: DataReader, E: EvmApi<D>> DerefMut for HostioInfo<'a, D, E> {
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
    #[error("logic error: `{0}`")]
    Logical(ErrReport),
    #[error("out of ink")]
    OutOfInk,
    #[error("exit early: `{0}`")]
    Exit(u32),
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
