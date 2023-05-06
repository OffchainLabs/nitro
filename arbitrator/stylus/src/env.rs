// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{
    evm::{api::EvmApi, EvmData},
    Bytes20, Bytes32, Color,
};
use derivative::Derivative;
use eyre::{eyre, ErrReport};
use prover::programs::{config::PricingParams, meter::OutOfInkError, prelude::*};
use std::{
    fmt::{Debug, Display},
    io,
    ops::{Deref, DerefMut},
};
use thiserror::Error;
use wasmer::{
    AsStoreRef, FunctionEnvMut, Global, Memory, MemoryAccessError, MemoryView, StoreMut, WasmPtr,
};

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

#[derive(Clone, Debug)]
pub struct MeterData {
    /// The amount of ink left
    pub ink_left: Global,
    /// Whether the instance has run out of ink
    pub ink_status: Global,
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

    pub fn start<'a>(env: &'a mut WasmEnvMut<'_, E>) -> Result<HostioInfo<'a, E>, Escape> {
        let mut info = Self::start_free(env);
        let cost = info.config().pricing.hostio_ink;
        info.buy_ink(cost)?;
        Ok(info)
    }

    pub fn start_free<'a>(env: &'a mut WasmEnvMut<'_, E>) -> HostioInfo<'a, E> {
        let (env, store) = env.data_and_store_mut();
        let memory = env.memory.clone().unwrap();
        HostioInfo { env, memory, store }
    }

    pub fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }
}

pub struct HostioInfo<'a, E: EvmApi> {
    pub env: &'a mut WasmEnv<E>,
    pub memory: Memory,
    pub store: StoreMut<'a>,
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

    pub fn _write_u64(&mut self, ptr: u32, x: u64) -> Result<&mut Self, MemoryAccessError> {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x)?;
        Ok(self)
    }

    pub fn read_slice(&self, ptr: u32, len: u32) -> Result<Vec<u8>, MemoryAccessError> {
        let mut data = vec![0; len as usize];
        self.view().read(ptr.into(), &mut data)?;
        Ok(data)
    }

    pub fn read_bytes20(&self, ptr: u32) -> eyre::Result<Bytes20> {
        let data = self.read_slice(ptr, 20)?;
        Ok(data.try_into()?)
    }

    pub fn read_bytes32(&self, ptr: u32) -> eyre::Result<Bytes32> {
        let data = self.read_slice(ptr, 32)?;
        Ok(data.try_into()?)
    }

    pub fn write_slice(&self, ptr: u32, src: &[u8]) -> Result<(), MemoryAccessError> {
        self.view().write(ptr.into(), src)
    }

    pub fn write_bytes20(&self, ptr: u32, src: Bytes20) -> eyre::Result<()> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }

    pub fn _write_bytes32(&self, ptr: u32, src: Bytes32) -> eyre::Result<()> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }
}

impl<'a, E: EvmApi> MeteredMachine for HostioInfo<'a, E> {
    fn ink_left(&mut self) -> MachineMeter {
        let store = &mut self.store;
        let meter = self.env.meter.as_ref().unwrap();
        let status = meter.ink_status.get(store);
        let status = status.try_into().expect("type mismatch");
        let ink = meter.ink_left.get(store);
        let ink = ink.try_into().expect("type mismatch");

        match status {
            0_u32 => MachineMeter::Ready(ink),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, value: MachineMeter) {
        let store = &mut self.store;
        let meter = self.env.meter.as_ref().unwrap();
        let ink = value.ink();
        let status = value.status();
        meter.ink_left.set(store, ink.into()).unwrap();
        meter.ink_status.set(store, status.into()).unwrap();
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
