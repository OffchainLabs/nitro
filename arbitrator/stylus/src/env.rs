// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::{eyre, ErrReport};
use ouroboros::self_referencing;
use prover::{
    programs::{
        config::{PricingParams, StylusConfig},
        meter::{MachineMeter, MeteredMachine},
        run::UserOutcomeKind,
    },
    utils::{Bytes20, Bytes32},
};
use std::{
    io,
    ops::{Deref, DerefMut},
};
use thiserror::Error;
use wasmer::{
    AsStoreMut, AsStoreRef, FunctionEnvMut, Global, Memory, MemoryAccessError, MemoryView,
    StoreMut, StoreRef, WasmPtr,
};

#[self_referencing]
pub struct MemoryViewContainer {
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

    pub fn view(&self) -> &MemoryView {
        self.borrow_view()
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
}

pub type WasmEnvMut<'a> = FunctionEnvMut<'a, WasmEnv>;

#[derive(Default)]
pub struct WasmEnv {
    /// The instance's arguments
    pub args: Vec<u8>,
    /// The instance's return data
    pub outs: Vec<u8>,
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Mechanism for accessing metering-specific global state
    pub meter: Option<MeterData>,
    /// Mechanism for reading and writing permanent storage, and doing calls
    pub evm: Option<EvmAPI>,
    /// The instance's config
    pub config: StylusConfig,
}

#[derive(Clone, Debug)]
pub struct MeterData {
    /// The amount of wasm gas left
    pub gas_left: Global,
    /// Whether the instance has run out of gas
    pub gas_status: Global,
    /// The pricing parameters associated with this program's environment
    pub pricing: PricingParams,
}

/// State load: key → (value, cost)
pub type GetBytes32 = Box<dyn Fn(Bytes32) -> (Bytes32, u64) + Send>;

/// State store: (key, value) → (cost, error)
pub type SetBytes32 = Box<dyn FnMut(Bytes32, Bytes32) -> eyre::Result<u64> + Send>;

/// Contract call: (contract, calldata, evm_gas, value) → (return_data_len, evm_cost, status)
pub type CallContract =
    Box<dyn Fn(Bytes20, Vec<u8>, u64, Bytes32) -> (u32, u64, UserOutcomeKind) + Send>;

/// Last call's return data: () → (return_data)
pub type GetReturnData = Box<dyn Fn() -> Vec<u8> + Send>;

pub struct EvmAPI {
    get_bytes32: GetBytes32,
    set_bytes32: SetBytes32,
    call_contract: CallContract,
    get_return_data: GetReturnData,
    return_data_len: u32,
}

impl WasmEnv {
    pub fn new(config: StylusConfig) -> Self {
        Self {
            config,
            ..Default::default()
        }
    }

    pub fn set_evm_api(
        &mut self,
        get_bytes32: GetBytes32,
        set_bytes32: SetBytes32,
        call_contract: CallContract,
        get_return_data: GetReturnData,
    ) {
        self.evm = Some(EvmAPI {
            get_bytes32,
            set_bytes32,
            call_contract,
            get_return_data,
            return_data_len: 0,
        })
    }

    pub fn evm(&mut self) -> &mut EvmAPI {
        self.evm.as_mut().expect("no evm api")
    }

    pub fn evm_ref(&self) -> &EvmAPI {
        self.evm.as_ref().expect("no evm api")
    }

    pub fn memory(env: &mut WasmEnvMut<'_>) -> MemoryViewContainer {
        MemoryViewContainer::create(env)
    }

    pub fn return_data_len(&self) -> u32 {
        self.evm_ref().return_data_len
    }

    pub fn set_return_data_len(&mut self, len: u32) {
        self.evm().return_data_len = len;
    }

    pub fn data<'a, 'b: 'a>(env: &'a mut WasmEnvMut<'b>) -> (&'a mut Self, MemoryViewContainer) {
        let memory = MemoryViewContainer::create(env);
        (env.data_mut(), memory)
    }

    pub fn meter<'a, 'b>(env: &'a mut WasmEnvMut<'b>) -> MeterState<'a> {
        let state = env.data().meter.clone().unwrap();
        let store = env.as_store_mut();
        MeterState::new(state, store)
    }

    pub fn begin<'a, 'b>(env: &'a mut WasmEnvMut<'b>) -> Result<MeterState<'a>, Escape> {
        let mut state = Self::meter(env);
        state.buy_gas(state.pricing.hostio_cost)?;
        Ok(state)
    }

    pub fn start<'a, 'b>(env: &'a mut WasmEnvMut<'b>) -> Result<HostioInfo<'a>, Escape> {
        let (env, store) = env.data_and_store_mut();
        let memory = env.memory.clone().unwrap();
        let mut info = HostioInfo { env, memory, store };
        let cost = info.meter().pricing.hostio_cost;
        info.buy_gas(cost)?;
        Ok(info)
    }
}

pub struct HostioInfo<'a> {
    pub env: &'a mut WasmEnv,
    pub memory: Memory,
    pub store: StoreMut<'a>,
}

impl<'a> HostioInfo<'a> {
    pub fn meter(&mut self) -> &mut MeterData {
        self.meter.as_mut().unwrap()
    }

    pub fn buy_gas(&mut self, gas: u64) -> MaybeEscape {
        let MachineMeter::Ready(gas_left) = self.gas_left() else {
            return Escape::out_of_gas();
        };
        if gas_left < gas {
            return Escape::out_of_gas();
        }
        self.set_gas(gas_left - gas);
        Ok(())
    }

    pub fn buy_evm_gas(&mut self, evm: u64) -> MaybeEscape {
        if let Ok(wasm_gas) = self.meter().pricing.evm_to_wasm(evm) {
            self.buy_gas(wasm_gas)?;
        }
        Ok(())
    }

    /// Checks if the user has enough evm gas, but doesn't burn any
    pub fn require_evm_gas(&mut self, evm: u64) -> MaybeEscape {
        let Ok(wasm_gas) = self.meter().pricing.evm_to_wasm(evm) else {
            return Ok(())
        };
        let MachineMeter::Ready(gas_left) = self.gas_left() else {
            return Escape::out_of_gas();
        };
        match gas_left < wasm_gas {
            true => Escape::out_of_gas(),
            false => Ok(()),
        }
    }

    pub fn pay_for_evm_copy(&mut self, bytes: usize) -> MaybeEscape {
        let evm_words = |count: u64| count.saturating_mul(31) / 32;
        let evm_gas = evm_words(bytes as u64).saturating_mul(3); // 3 evm gas per word
        self.buy_evm_gas(evm_gas)
    }

    pub fn view(&self) -> MemoryView {
        self.memory.view(&self.store.as_store_ref())
    }

    pub fn write_u8(&mut self, ptr: u32, x: u8) -> &mut Self {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x).unwrap();
        self
    }

    pub fn write_u32(&mut self, ptr: u32, x: u32) -> &mut Self {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x).unwrap();
        self
    }

    pub fn write_u64(&mut self, ptr: u32, x: u64) -> &mut Self {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(&self.view()).write(x).unwrap();
        self
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
}

impl<'a> MeteredMachine for HostioInfo<'a> {
    fn gas_left(&mut self) -> MachineMeter {
        let store = &mut self.store;
        let meter = self.env.meter.as_ref().unwrap();
        let status = meter.gas_status.get(store);
        let status = status.try_into().expect("type mismatch");
        let gas = meter.gas_left.get(store);
        let gas = gas.try_into().expect("type mismatch");

        match status {
            0_u32 => MachineMeter::Ready(gas),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, gas: u64) {
        let store = &mut self.store;
        let meter = self.env.meter.as_ref().unwrap();
        meter.gas_left.set(store, gas.into()).unwrap();
        meter.gas_status.set(store, 0.into()).unwrap();
    }
}

impl<'a> Deref for HostioInfo<'a> {
    type Target = WasmEnv;

    fn deref(&self) -> &Self::Target {
        self.env
    }
}

impl<'a> DerefMut for HostioInfo<'a> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.env
    }
}

pub struct MeterState<'a> {
    state: MeterData,
    store: StoreMut<'a>,
}

impl<'a> Deref for MeterState<'a> {
    type Target = MeterData;

    fn deref(&self) -> &Self::Target {
        &self.state
    }
}

impl<'a> DerefMut for MeterState<'a> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.state
    }
}

impl<'a> MeterState<'a> {
    pub fn new(state: MeterData, store: StoreMut<'a>) -> Self {
        Self { state, store }
    }

    pub fn buy_gas(&mut self, gas: u64) -> MaybeEscape {
        let MachineMeter::Ready(gas_left) = self.gas_left() else {
            return Escape::out_of_gas();
        };
        if gas_left < gas {
            return Escape::out_of_gas();
        }
        self.set_gas(gas_left - gas);
        Ok(())
    }

    pub fn buy_evm_gas(&mut self, evm: u64) -> MaybeEscape {
        if let Ok(wasm_gas) = self.pricing.evm_to_wasm(evm) {
            self.buy_gas(wasm_gas)?;
        }
        Ok(())
    }

    /// Checks if the user has enough evm gas, but doesn't burn any
    pub fn require_evm_gas(&mut self, evm: u64) -> MaybeEscape {
        let Ok(wasm_gas) = self.pricing.evm_to_wasm(evm) else {
            return Ok(())
        };
        let MachineMeter::Ready(gas_left) = self.gas_left() else {
            return Escape::out_of_gas();
        };
        match gas_left < wasm_gas {
            true => Escape::out_of_gas(),
            false => Ok(()),
        }
    }

    pub fn pay_for_evm_copy(&mut self, bytes: usize) -> MaybeEscape {
        let evm_words = |count: u64| count.saturating_mul(31) / 32;
        let evm_gas = evm_words(bytes as u64).saturating_mul(3); // 3 evm gas per word
        self.buy_evm_gas(evm_gas)
    }
}

impl<'a> MeteredMachine for MeterState<'a> {
    fn gas_left(&mut self) -> MachineMeter {
        let store = &mut self.store;
        let state = &self.state;

        let status = state.gas_status.get(store);
        let status = status.try_into().expect("type mismatch");
        let gas = state.gas_left.get(store);
        let gas = gas.try_into().expect("type mismatch");

        match status {
            0_u32 => MachineMeter::Ready(gas),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, gas: u64) {
        let store = &mut self.store;
        let state = &self.state;
        state.gas_left.set(store, gas.into()).unwrap();
        state.gas_status.set(store, 0.into()).unwrap();
    }
}

impl EvmAPI {
    pub fn load_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        (self.get_bytes32)(key)
    }

    pub fn store_bytes32(&mut self, key: Bytes32, value: Bytes32) -> eyre::Result<u64> {
        (self.set_bytes32)(key, value)
    }

    pub fn call_contract(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        evm_gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        (self.call_contract)(contract, input, evm_gas, value)
    }

    pub fn load_return_data(&mut self) -> Vec<u8> {
        (self.get_return_data)()
    }
}

pub type MaybeEscape = Result<(), Escape>;

#[derive(Error, Debug)]
pub enum Escape {
    #[error("failed to access memory: `{0}`")]
    Memory(MemoryAccessError),
    #[error("internal error: `{0}`")]
    Internal(ErrReport),
    #[error("out of gas")]
    OutOfGas,
}

impl Escape {
    pub fn internal<T>(error: &'static str) -> Result<T, Escape> {
        Err(Self::Internal(eyre!(error)))
    }

    pub fn out_of_gas<T>() -> Result<T, Escape> {
        Err(Self::OutOfGas)
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
