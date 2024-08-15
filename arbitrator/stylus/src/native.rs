// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    cache::InitCache,
    env::{MeterData, WasmEnv},
    host,
};
use arbutil::{
    evm::{
        api::{DataReader, EvmApi},
        EvmData,
    },
    operator::OperatorCode,
    Bytes32, Color,
};
use eyre::{bail, eyre, ErrReport, Result};
use prover::{
    machine::Module as ProverModule,
    programs::{
        config::PricingParams,
        counter::{Counter, CountingMachine, OP_OFFSETS},
        depth::STYLUS_STACK_LEFT,
        meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS},
        prelude::*,
        start::StartMover,
        StylusData,
    },
};
use std::{
    collections::BTreeMap,
    fmt::Debug,
    ops::{Deref, DerefMut},
};
use wasmer::{
    AsStoreMut, Function, FunctionEnv, Imports, Instance, Memory, Module, Pages, Store, Target,
    TypedFunction, Value, WasmTypeList,
};
use wasmer_vm::VMExtern;

use crate::target_cache::target_native;

#[derive(Debug)]
pub struct NativeInstance<D: DataReader, E: EvmApi<D>> {
    pub instance: Instance,
    pub store: Store,
    pub env: FunctionEnv<WasmEnv<D, E>>,
}

impl<D: DataReader, E: EvmApi<D>> NativeInstance<D, E> {
    pub fn new(instance: Instance, store: Store, env: FunctionEnv<WasmEnv<D, E>>) -> Self {
        let mut native = Self {
            instance,
            store,
            env,
        };
        if let Some(config) = native.env().config {
            native.set_stack(config.max_depth);
        }
        native
    }

    pub fn env(&self) -> &WasmEnv<D, E> {
        self.env.as_ref(&self.store)
    }

    pub fn env_mut(&mut self) -> &mut WasmEnv<D, E> {
        self.env.as_mut(&mut self.store)
    }

    pub fn config(&self) -> StylusConfig {
        self.env().config.expect("no config")
    }

    pub fn memory(&self) -> Memory {
        self.env().memory.as_ref().unwrap().clone()
    }

    pub fn memory_size(&self) -> Pages {
        self.memory().ty(&self.store).minimum
    }

    pub fn read_slice(&self, mem: &str, ptr: usize, len: usize) -> Result<Vec<u8>> {
        let memory = self.exports.get_memory(mem)?;
        let memory = memory.view(&self.store);
        let mut data = vec![0; len];
        memory.read(ptr as u64, &mut data)?;
        Ok(data)
    }

    /// Creates a `NativeInstance` from a serialized module.
    ///
    /// # Safety
    ///
    /// `module` must represent a valid module.
    pub unsafe fn deserialize(
        module: &[u8],
        compile: CompileConfig,
        evm: E,
        evm_data: EvmData,
    ) -> Result<Self> {
        let env = WasmEnv::new(compile, None, evm, evm_data);
        let store = env.compile.store(target_native());
        let module = unsafe { Module::deserialize_unchecked(&store, module)? };
        Self::from_module(module, store, env)
    }

    /// Creates a `NativeInstance` from a serialized module, or from a cached one if known.
    ///
    /// # Safety
    ///
    /// `module` must represent a valid module.
    pub unsafe fn deserialize_cached(
        module: &[u8],
        asm_size_estimate: u32,
        version: u16,
        evm: E,
        evm_data: EvmData,
        mut long_term_tag: u32,
        debug: bool,
    ) -> Result<Self> {
        let compile = CompileConfig::version(version, debug);
        let env = WasmEnv::new(compile, None, evm, evm_data);
        let module_hash = env.evm_data.module_hash;

        if let Some((module, store)) = InitCache::get(module_hash, version, debug) {
            return Self::from_module(module, store, env);
        }
        if !env.evm_data.cached {
            long_term_tag = 0;
        }
        let (module, store) =
            InitCache::insert(module_hash, module, asm_size_estimate, version, long_term_tag, debug)?;
        Self::from_module(module, store, env)
    }

    pub fn from_path(
        path: &str,
        evm_api: E,
        evm_data: EvmData,
        compile: &CompileConfig,
        config: StylusConfig,
        target: Target,
    ) -> Result<Self> {
        let env = WasmEnv::new(compile.clone(), Some(config), evm_api, evm_data);
        let store = env.compile.store(target);
        let wat_or_wasm = std::fs::read(path)?;
        let module = Module::new(&store, wat_or_wasm)?;
        Self::from_module(module, store, env)
    }

    fn from_module(module: Module, mut store: Store, env: WasmEnv<D, E>) -> Result<Self> {
        let debug_funcs = env.compile.debug.debug_funcs;
        let func_env = FunctionEnv::new(&mut store, env);
        let mut imports = Imports::new();
        macro_rules! func {
            ($rust_mod:path, $func:ident) => {{
                use $rust_mod as rust_mod;
                Function::new_typed_with_env(&mut store, &func_env, rust_mod::$func)
            }};
        }
        macro_rules! define_imports {
            ($($wasm_mod:literal => $rust_mod:path { $( $import:ident ),* $(,)? },)* $(,)?) => {
                $(
                    $(
                        define_imports!(@@imports $wasm_mod, func!($rust_mod, $import), $import, "arbitrator_forward__");
                    )*
                )*
            };
            (@@imports $wasm_mod:literal, $func:expr, $import:ident, $($p:expr),*) => {
                define_imports!(@imports $wasm_mod, $func, $import, $($p),*, "");
            };
            (@imports $wasm_mod:literal, $func:expr, $import:ident, $($p:expr),*) => {
                $(
                    imports.define($wasm_mod, concat!($p, stringify!($import)), $func);
                )*
            };
        }
        define_imports!(
            "vm_hooks" => host {
                read_args, write_result, exit_early,
                storage_load_bytes32, storage_cache_bytes32, storage_flush_cache, transient_load_bytes32, transient_store_bytes32,
                call_contract, delegate_call_contract, static_call_contract, create1, create2, read_return_data, return_data_size,
                emit_log,
                account_balance, account_code, account_codehash, account_code_size,
                evm_gas_left, evm_ink_left,
                block_basefee, chainid, block_coinbase, block_gas_limit, block_number, block_timestamp,
                contract_address,
                math_div, math_mod, math_pow, math_add_mod, math_mul_mod,
                msg_reentrant, msg_sender, msg_value,
                tx_gas_price, tx_ink_price, tx_origin,
                pay_for_memory_grow,
                native_keccak256,
            },
        );
        if debug_funcs {
            define_imports!(
                "console" => host::console {
                    log_txt,
                    log_i32, log_i64, log_f32, log_f64,
                    tee_i32, tee_i64, tee_f32, tee_f64,
                },
                "debug" => host::debug {
                    null_host,
                },
            );
        }
        let instance = Instance::new(&mut store, &module, &imports)?;
        let exports = &instance.exports;
        let memory = exports.get_memory("memory")?.clone();

        let env = func_env.as_mut(&mut store);
        env.memory = Some(memory);

        let mut native = Self::new(instance, store, func_env);
        native.set_meter_data();
        Ok(native)
    }

    pub fn set_meter_data(&mut self) {
        let store = &mut self.store;
        let exports = &self.instance.exports;

        let mut expect_global = |name| {
            let VMExtern::Global(sh) = exports.get_extern(name).unwrap().to_vm_extern() else {
                panic!("name not found global");
            };
            sh.get(store.objects_mut()).vmglobal()
        };
        let ink_left = expect_global(STYLUS_INK_LEFT);
        let ink_status = expect_global(STYLUS_INK_STATUS);

        self.env_mut().meter = Some(MeterData {
            ink_left,
            ink_status,
        });
    }

    pub fn get_global<T>(&mut self, name: &str) -> Result<T>
    where
        T: TryFrom<Value>,
        T::Error: Debug,
    {
        let store = &mut self.store.as_store_mut();
        let Ok(global) = self.instance.exports.get_global(name) else {
            bail!("global {} does not exist", name.red())
        };
        let ty = global.get(store);

        ty.try_into()
            .map_err(|_| eyre!("global {} has the wrong type", name.red()))
    }

    pub fn set_global<T>(&mut self, name: &str, value: T) -> Result<()>
    where
        T: Into<Value>,
    {
        let store = &mut self.store.as_store_mut();
        let Ok(global) = self.instance.exports.get_global(name) else {
            bail!("global {} does not exist", name.red())
        };
        global.set(store, value.into()).map_err(ErrReport::msg)
    }

    pub fn call_func<R>(&mut self, func: TypedFunction<(), R>, ink: u64) -> Result<R>
    where
        R: WasmTypeList,
    {
        self.set_ink(ink);
        Ok(func.call(&mut self.store)?)
    }
}

impl<D: DataReader, E: EvmApi<D>> Deref for NativeInstance<D, E> {
    type Target = Instance;

    fn deref(&self) -> &Self::Target {
        &self.instance
    }
}

impl<D: DataReader, E: EvmApi<D>> DerefMut for NativeInstance<D, E> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.instance
    }
}

impl<D: DataReader, E: EvmApi<D>> MeteredMachine for NativeInstance<D, E> {
    fn ink_left(&self) -> MachineMeter {
        let vm = self.env().meter();
        match vm.status() {
            0 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let vm = self.env_mut().meter_mut();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }
}

impl<D: DataReader, E: EvmApi<D>> GasMeteredMachine for NativeInstance<D, E> {
    fn pricing(&self) -> PricingParams {
        self.env().config.unwrap().pricing
    }
}

impl<D: DataReader, E: EvmApi<D>> CountingMachine for NativeInstance<D, E> {
    fn operator_counts(&mut self) -> Result<BTreeMap<OperatorCode, u64>> {
        let mut counts = BTreeMap::new();

        for (&op, &offset) in OP_OFFSETS.lock().iter() {
            let count: u64 = self.get_global(&Counter::global_name(offset))?;
            if count != 0 {
                counts.insert(op, count);
            }
        }
        Ok(counts)
    }
}

impl<D: DataReader, E: EvmApi<D>> DepthCheckedMachine for NativeInstance<D, E> {
    fn stack_left(&mut self) -> u32 {
        self.get_global(STYLUS_STACK_LEFT).unwrap()
    }

    fn set_stack(&mut self, size: u32) {
        self.set_global(STYLUS_STACK_LEFT, size).unwrap()
    }
}

impl<D: DataReader, E: EvmApi<D>> StartlessMachine for NativeInstance<D, E> {
    fn get_start(&self) -> Result<TypedFunction<(), ()>> {
        let store = &self.store;
        let exports = &self.instance.exports;
        exports
            .get_typed_function(store, StartMover::NAME)
            .map_err(ErrReport::new)
    }
}

pub fn module(wasm: &[u8], compile: CompileConfig, target: Target) -> Result<Vec<u8>> {
    let store = compile.store(target);
    let module = Module::new(&store, wasm)?;

    let module = module.serialize()?;
    Ok(module.to_vec())
}

pub fn activate(
    wasm: &[u8],
    codehash: &Bytes32,
    version: u16,
    page_limit: u16,
    debug: bool,
    gas: &mut u64,
) -> Result<(ProverModule, StylusData)> {
    let (module, stylus_data) =
        ProverModule::activate(wasm, codehash, version, page_limit, debug, gas)?;

    Ok((module, stylus_data))
}

pub fn compile(wasm: &[u8], version: u16, debug: bool, target: Target) -> Result<Vec<u8>> {
    let compile = CompileConfig::version(version, debug);
    self::module(wasm, compile, target)
}
