// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{evm, Color};
use eyre::{eyre, ErrReport};
use prover::{
    programs::{
        config::{PricingParams, StylusConfig},
        meter::{MachineMeter, MeteredMachine},
        prelude::CompileConfig,
        run::UserOutcomeKind,
    },
    utils::{Bytes20, Bytes32},
};
use std::{
    fmt::Display,
    io,
    ops::{Deref, DerefMut},
};
use thiserror::Error;
use wasmer::{
    AsStoreRef, FunctionEnvMut, Global, Memory, MemoryAccessError, MemoryView, StoreMut, WasmPtr,
};

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
    /// Mechanism for reading EVM context data
    pub evm_data: Option<EvmData>,
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

/// Hash for given block: key → (value, cost)
pub type BlockHash = Box<dyn Fn(Bytes32) -> (Bytes32, u64) + Send>;

/// State load: key → (value, cost)
pub type GetBytes32 = Box<dyn Fn(Bytes32) -> (Bytes32, u64) + Send>;

/// State store: (key, value) → (cost, error)
pub type SetBytes32 = Box<dyn FnMut(Bytes32, Bytes32) -> eyre::Result<u64> + Send>;

/// Contract call: (contract, calldata, gas, value) → (return_data_len, gas_cost, status)
pub type ContractCall =
    Box<dyn Fn(Bytes20, Vec<u8>, u64, Bytes32) -> (u32, u64, UserOutcomeKind) + Send>;

/// Delegate call: (contract, calldata, gas) → (return_data_len, gas_cost, status)
pub type DelegateCall = Box<dyn Fn(Bytes20, Vec<u8>, u64) -> (u32, u64, UserOutcomeKind) + Send>;

/// Static call: (contract, calldata, gas) → (return_data_len, gas_cost, status)
pub type StaticCall = Box<dyn Fn(Bytes20, Vec<u8>, u64) -> (u32, u64, UserOutcomeKind) + Send>;

/// Last call's return data: () → return_data
pub type GetReturnData = Box<dyn Fn() -> Vec<u8> + Send>;

/// Emits a log event: (data, topics) -> error
pub type EmitLog = Box<dyn Fn(Vec<u8>, usize) -> eyre::Result<()> + Send>;

/// Creates a contract: (code, endowment, gas) -> (address/error, return_data_len, gas_cost)
pub type Create1 = Box<dyn Fn(Vec<u8>, Bytes32, u64) -> (eyre::Result<Bytes20>, u32, u64) + Send>;

/// Creates a contract: (code, endowment, salt, gas) -> (address/error, return_data_len, gas_cost)
pub type Create2 =
    Box<dyn Fn(Vec<u8>, Bytes32, Bytes32, u64) -> (eyre::Result<Bytes20>, u32, u64) + Send>;

pub struct EvmAPI {
    block_hash: BlockHash,
    get_bytes32: GetBytes32,
    set_bytes32: SetBytes32,
    contract_call: ContractCall,
    delegate_call: DelegateCall,
    static_call: StaticCall,
    create1: Create1,
    create2: Create2,
    get_return_data: GetReturnData,
    return_data_len: u32,
    emit_log: EmitLog,
}

#[repr(C)]
pub struct EvmData {
    pub block_basefee: Bytes32,
    pub block_chainid: Bytes32,
    pub block_coinbase: Bytes20,
    pub block_difficulty: Bytes32,
    pub block_gas_limit: u64,
    pub block_number: Bytes32,
    pub block_timestamp: Bytes32,
    pub msg_sender: Bytes20,
    pub msg_value: Bytes32,
    pub gas_price: Bytes32,
    pub origin: Bytes20,
}

impl WasmEnv {
    pub fn new(compile: CompileConfig, config: Option<StylusConfig>) -> Self {
        Self {
            compile,
            config,
            ..Default::default()
        }
    }

    pub fn set_evm_api(
        &mut self,
        block_hash: BlockHash,
        get_bytes32: GetBytes32,
        set_bytes32: SetBytes32,
        contract_call: ContractCall,
        delegate_call: DelegateCall,
        static_call: StaticCall,
        create1: Create1,
        create2: Create2,
        get_return_data: GetReturnData,
        emit_log: EmitLog,
    ) {
        self.evm = Some(EvmAPI {
            block_hash,
            get_bytes32,
            set_bytes32,
            contract_call,
            delegate_call,
            static_call,
            create1,
            create2,
            get_return_data,
            emit_log,
            return_data_len: 0,
        })
    }

    pub fn evm(&mut self) -> &mut EvmAPI {
        self.evm.as_mut().expect("no evm api")
    }

    pub fn evm_ref(&self) -> &EvmAPI {
        self.evm.as_ref().expect("no evm api")
    }

    pub fn evm_data(&self) -> &EvmData {
        self.evm_data.as_ref().expect("no evm data")
    }

    pub fn return_data_len(&self) -> u32 {
        self.evm_ref().return_data_len
    }

    pub fn set_return_data_len(&mut self, len: u32) {
        self.evm().return_data_len = len;
    }

    pub fn start<'a>(env: &'a mut WasmEnvMut<'_>) -> Result<HostioInfo<'a>, Escape> {
        let mut info = Self::start_free(env);
        let cost = info.config().pricing.hostio_ink;
        info.buy_ink(cost)?;
        Ok(info)
    }

    pub fn start_free<'a>(env: &'a mut WasmEnvMut<'_>) -> HostioInfo<'a> {
        let (env, store) = env.data_and_store_mut();
        let memory = env.memory.clone().unwrap();
        HostioInfo { env, memory, store }
    }

    pub fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }
}

pub struct HostioInfo<'a> {
    pub env: &'a mut WasmEnv,
    pub memory: Memory,
    pub store: StoreMut<'a>,
}

impl<'a> HostioInfo<'a> {
    pub fn config(&self) -> StylusConfig {
        self.config.expect("no config")
    }

    pub fn pricing(&self) -> PricingParams {
        self.config().pricing
    }

    pub fn gas_left(&mut self) -> u64 {
        let ink = self.ink_left().into();
        self.pricing().ink_to_gas(ink)
    }

    pub fn buy_ink(&mut self, ink: u64) -> MaybeEscape {
        let MachineMeter::Ready(ink_left) = self.ink_left() else {
            return Escape::out_of_ink();
        };
        if ink_left < ink {
            return Escape::out_of_ink();
        }
        self.set_ink(ink_left - ink);
        Ok(())
    }

    pub fn buy_gas(&mut self, gas: u64) -> MaybeEscape {
        let ink = self.pricing().gas_to_ink(gas);
        self.buy_ink(ink)
    }

    /// Checks if the user has enough gas, but doesn't burn any
    pub fn require_gas(&mut self, gas: u64) -> MaybeEscape {
        let ink = self.pricing().gas_to_ink(gas);
        let MachineMeter::Ready(ink_left) = self.ink_left() else {
            return Escape::out_of_ink();
        };
        match ink_left < ink {
            true => Escape::out_of_ink(),
            false => Ok(()),
        }
    }

    pub fn pay_for_evm_copy(&mut self, bytes: u64) -> MaybeEscape {
        let evm_words = |count: u64| count.saturating_mul(31) / 32;
        let gas = evm_words(bytes).saturating_mul(evm::COPY_WORD_GAS);
        self.buy_gas(gas)
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

    pub fn write_bytes32(&self, ptr: u32, src: Bytes32) -> eyre::Result<()> {
        self.write_slice(ptr, &src.0)?;
        Ok(())
    }
}

impl<'a> MeteredMachine for HostioInfo<'a> {
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

    fn set_ink(&mut self, ink: u64) {
        let store = &mut self.store;
        let meter = self.env.meter.as_ref().unwrap();
        meter.ink_left.set(store, ink.into()).unwrap();
        meter.ink_status.set(store, 0.into()).unwrap();
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

impl EvmAPI {
    pub fn block_hash(&mut self, key: Bytes32) -> (Bytes32, u64) {
        (self.block_hash)(key)
    }

    pub fn load_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        (self.get_bytes32)(key)
    }

    pub fn store_bytes32(&mut self, key: Bytes32, value: Bytes32) -> eyre::Result<u64> {
        (self.set_bytes32)(key, value)
    }

    pub fn contract_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        (self.contract_call)(contract, input, gas, value)
    }

    pub fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        (self.delegate_call)(contract, input, gas)
    }

    pub fn static_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        (self.static_call)(contract, input, gas)
    }

    pub fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64) {
        (self.create1)(code, endowment, gas)
    }

    pub fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64) {
        (self.create2)(code, endowment, salt, gas)
    }

    pub fn load_return_data(&mut self) -> Vec<u8> {
        (self.get_return_data)()
    }

    pub fn emit_log(&mut self, data: Vec<u8>, topics: usize) -> eyre::Result<()> {
        (self.emit_log)(data, topics)
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
