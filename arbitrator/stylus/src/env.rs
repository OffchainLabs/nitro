// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::ErrReport;
use ouroboros::self_referencing;
use prover::programs::{
    config::{PricingParams, StylusConfig},
    meter::{MachineMeter, MeteredMachine},
};
use std::ops::{Deref, DerefMut};
use thiserror::Error;
use wasmer::{
    AsStoreMut, AsStoreRef, FunctionEnvMut, Global, Memory, MemoryAccessError, MemoryView,
    StoreMut, StoreRef,
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
    /// Mechanism for accessing stylus-specific global state
    pub state: Option<SystemStateData>,
    /// The instance's config
    pub config: StylusConfig,
}

#[derive(Clone, Debug)]
pub struct SystemStateData {
    /// The amount of wasm gas left
    pub gas_left: Global,
    /// Whether the instance has run out of gas
    pub gas_status: Global,
    /// The pricing parameters associated with this program's environment
    pub pricing: PricingParams,
}

impl WasmEnv {
    pub fn new(config: StylusConfig) -> Self {
        Self {
            config,
            ..Default::default()
        }
    }

    pub fn memory(env: &mut WasmEnvMut<'_>) -> MemoryViewContainer {
        MemoryViewContainer::create(env)
    }

    pub fn data<'a, 'b: 'a>(env: &'a mut WasmEnvMut<'b>) -> (&'a mut WasmEnv, MemoryViewContainer) {
        let memory = MemoryViewContainer::create(env);
        (env.data_mut(), memory)
    }

    pub fn begin<'a, 'b>(env: &'a mut WasmEnvMut<'b>) -> Result<SystemState<'a>, Escape> {
        let state = env.data().state.clone().unwrap();
        let store = env.as_store_mut();
        let mut state = SystemState::new(state, store);
        state.buy_gas(state.pricing.hostio_cost)?;
        Ok(state)
    }
}

pub struct SystemState<'a> {
    state: SystemStateData,
    store: StoreMut<'a>,
}

impl<'a> Deref for SystemState<'a> {
    type Target = SystemStateData;

    fn deref(&self) -> &Self::Target {
        &self.state
    }
}

impl<'a> DerefMut for SystemState<'a> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.state
    }
}

impl<'a> SystemState<'a> {
    fn new(state: SystemStateData, store: StoreMut<'a>) -> Self {
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
}

impl<'a> MeteredMachine for SystemState<'a> {
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
    pub fn out_of_gas() -> MaybeEscape {
        Err(Self::OutOfGas)
    }
}

impl From<MemoryAccessError> for Escape {
    fn from(err: MemoryAccessError) -> Self {
        Self::Memory(err)
    }
}

impl From<ErrReport> for Escape {
    fn from(err: ErrReport) -> Self {
        Self::Internal(err)
    }
}
