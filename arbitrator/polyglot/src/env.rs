// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use ouroboros::self_referencing;
use prover::programs::{
    config::PolyglotConfig,
    meter::{MachineMeter, MeteredMachine},
};
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
    /// Mechanism for accessing polyglot-specific global state
    pub state: Option<SystemState>,
    /// The instance's config
    pub config: PolyglotConfig,
}

#[derive(Clone, Debug)]
pub struct SystemState {
    /// The amount of wasm gas left
    pub gas_left: Global,
    /// Whether the instance has run out of gas
    pub gas_status: Global,
    /// The price of wasm gas, measured in bips of an evm gas
    pub wasm_gas_price: u64,
    /// The amount of wasm gas one pays to do a polyhost call
    pub hostio_cost: u64,
}

impl WasmEnv {
    pub fn new(config: PolyglotConfig, args: Vec<u8>) -> Self {
        let mut env = Self::default();
        env.config = config;
        env.args = args;
        env
    }

    pub fn memory(env: &mut WasmEnvMut<'_>) -> MemoryViewContainer {
        MemoryViewContainer::create(env)
    }

    pub fn data<'a, 'b: 'a>(env: &'a mut WasmEnvMut<'b>) -> (&'a mut WasmEnv, MemoryViewContainer) {
        let memory = MemoryViewContainer::create(env);
        (env.data_mut(), memory)
    }

    pub fn begin<'a, 'b>(
        env: &'a mut WasmEnvMut<'b>,
    ) -> Result<(SystemState, StoreMut<'a>), Escape> {
        let mut state = env.data().state.clone().unwrap();
        let mut store = env.as_store_mut();
        state.buy_gas(&mut store, state.hostio_cost)?;
        Ok((state, store))
    }
}

impl SystemState {
    pub fn buy_gas(&mut self, store: &mut StoreMut, gas: u64) -> MaybeEscape {
        let MachineMeter::Ready(gas_left) = self.gas_left(store) else {
            return Escape::out_of_gas();
        };
        if gas_left < gas {
            return Escape::out_of_gas();
        }
        self.set_gas(store, gas_left - gas);
        Ok(())
    }

    pub fn buy_evm_gas(&mut self, store: &mut StoreMut, evm: u64) -> MaybeEscape {
        let wasm_gas = evm.saturating_mul(self.wasm_gas_price) / 100_00;
        self.buy_gas(store, wasm_gas)
    }
}

impl MeteredMachine for SystemState {
    fn gas_left(&self, store: &mut StoreMut) -> MachineMeter {
        let status: u32 = self.gas_status.get(store).try_into().unwrap();
        let gas = self.gas_left.get(store).try_into().unwrap();

        match status {
            0 => MachineMeter::Ready(gas),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, store: &mut StoreMut, gas: u64) {
        self.gas_left.set(store, gas.into()).expect("no global");
        self.gas_status.set(store, 0.into()).expect("no global");
    }
}

pub type MaybeEscape = Result<(), Escape>;

#[derive(Error, Debug)]
pub enum Escape {
    #[error("failed to access memory: `{0}`")]
    Memory(MemoryAccessError),
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
