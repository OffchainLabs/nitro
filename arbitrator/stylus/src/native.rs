// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{MeterData, WasmEnv},
    host,
};
use arbutil::{
    evm::{api::EvmApi, EvmData},
    operator::OperatorCode,
    Bytes32, Color,
};
use eyre::{bail, eyre, Context, ErrReport, Result};
use prover::binary::WasmBinary;
use prover::programs::{
    config::{PricingParams, WasmPricingInfo},
    counter::{Counter, CountingMachine, OP_OFFSETS},
    depth::STYLUS_STACK_LEFT,
    meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS},
    prelude::*,
    start::STYLUS_START,
};
use std::{
    collections::BTreeMap,
    fmt::Debug,
    ops::{Deref, DerefMut},
};
use wasmer::{
    imports, AsStoreMut, Function, FunctionEnv, Global, Instance, Memory, Module, Pages, Store,
    TypedFunction, Value, WasmTypeList,
};

#[derive(Debug)]
pub struct NativeInstance<E: EvmApi> {
    pub instance: Instance,
    pub store: Store,
    pub env: FunctionEnv<WasmEnv<E>>,
}

impl<E: EvmApi> NativeInstance<E> {
    pub fn new(instance: Instance, store: Store, env: FunctionEnv<WasmEnv<E>>) -> Self {
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

    pub fn env(&self) -> &WasmEnv<E> {
        self.env.as_ref(&self.store)
    }

    pub fn env_mut(&mut self) -> &mut WasmEnv<E> {
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
        let store = env.compile.store();
        let module = Module::deserialize(&store, module)?;
        Self::from_module(module, store, env)
    }

    pub fn from_path(
        path: &str,
        evm_api: E,
        evm_data: EvmData,
        compile: &CompileConfig,
        config: StylusConfig,
    ) -> Result<Self> {
        let env = WasmEnv::new(compile.clone(), Some(config), evm_api, evm_data);
        let store = env.compile.store();
        let wat_or_wasm = std::fs::read(path)?;
        let module = Module::new(&store, wat_or_wasm)?;
        Self::from_module(module, store, env)
    }

    fn from_module(module: Module, mut store: Store, env: WasmEnv<E>) -> Result<Self> {
        let debug_funcs = env.compile.debug.debug_funcs;
        let func_env = FunctionEnv::new(&mut store, env);
        macro_rules! func {
            ($func:expr) => {
                Function::new_typed_with_env(&mut store, &func_env, $func)
            };
        }
        let mut imports = imports! {
            "vm_hooks" => {
                "read_args" => func!(host::read_args),
                "write_result" => func!(host::write_result),
                "storage_load_bytes32" => func!(host::storage_load_bytes32),
                "storage_store_bytes32" => func!(host::storage_store_bytes32),
                "call_contract" => func!(host::call_contract),
                "delegate_call_contract" => func!(host::delegate_call_contract),
                "static_call_contract" => func!(host::static_call_contract),
                "create1" => func!(host::create1),
                "create2" => func!(host::create2),
                "read_return_data" => func!(host::read_return_data),
                "return_data_size" => func!(host::return_data_size),
                "emit_log" => func!(host::emit_log),
                "account_balance" => func!(host::account_balance),
                "account_codehash" => func!(host::account_codehash),
                "evm_gas_left" => func!(host::evm_gas_left),
                "evm_ink_left" => func!(host::evm_ink_left),
                "block_basefee" => func!(host::block_basefee),
                "chainid" => func!(host::chainid),
                "block_coinbase" => func!(host::block_coinbase),
                "block_gas_limit" => func!(host::block_gas_limit),
                "block_number" => func!(host::block_number),
                "block_timestamp" => func!(host::block_timestamp),
                "contract_address" => func!(host::contract_address),
                "msg_reentrant" => func!(host::msg_reentrant),
                "msg_sender" => func!(host::msg_sender),
                "msg_value" => func!(host::msg_value),
                "tx_gas_price" => func!(host::tx_gas_price),
                "tx_ink_price" => func!(host::tx_ink_price),
                "tx_origin" => func!(host::tx_origin),
                "memory_grow" => func!(host::memory_grow),
                "native_keccak256" => func!(host::native_keccak256),
            },
        };
        if debug_funcs {
            imports.define("console", "log_txt", func!(host::console_log_text));
            imports.define("console", "log_i32", func!(host::console_log::<E, u32>));
            imports.define("console", "log_i64", func!(host::console_log::<E, u64>));
            imports.define("console", "log_f32", func!(host::console_log::<E, f32>));
            imports.define("console", "log_f64", func!(host::console_log::<E, f64>));
            imports.define("console", "tee_i32", func!(host::console_tee::<E, u32>));
            imports.define("console", "tee_i64", func!(host::console_tee::<E, u64>));
            imports.define("console", "tee_f32", func!(host::console_tee::<E, f32>));
            imports.define("console", "tee_f64", func!(host::console_tee::<E, f64>));
            imports.define("debug", "null_host", func!(host::null_host));
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

        let expect_global = |name| -> Global { exports.get_global(name).unwrap().clone() };
        let ink_left = unsafe { expect_global(STYLUS_INK_LEFT).vmglobal(store) };
        let ink_status = unsafe { expect_global(STYLUS_INK_STATUS).vmglobal(store) };

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

impl<E: EvmApi> Deref for NativeInstance<E> {
    type Target = Instance;

    fn deref(&self) -> &Self::Target {
        &self.instance
    }
}

impl<E: EvmApi> DerefMut for NativeInstance<E> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.instance
    }
}

impl<E: EvmApi> MeteredMachine for NativeInstance<E> {
    fn ink_left(&mut self) -> MachineMeter {
        let vm = self.env_mut().meter();
        match vm.status() {
            0 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let vm = self.env_mut().meter();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }
}

impl<E: EvmApi> GasMeteredMachine for NativeInstance<E> {
    fn pricing(&mut self) -> PricingParams {
        self.env().config.unwrap().pricing
    }
}

impl<E: EvmApi> CountingMachine for NativeInstance<E> {
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

impl<E: EvmApi> DepthCheckedMachine for NativeInstance<E> {
    fn stack_left(&mut self) -> u32 {
        self.get_global(STYLUS_STACK_LEFT).unwrap()
    }

    fn set_stack(&mut self, size: u32) {
        self.set_global(STYLUS_STACK_LEFT, size).unwrap()
    }
}

impl<E: EvmApi> StartlessMachine for NativeInstance<E> {
    fn get_start(&self) -> Result<TypedFunction<(), ()>> {
        let store = &self.store;
        let exports = &self.instance.exports;
        exports
            .get_typed_function(store, STYLUS_START)
            .map_err(ErrReport::new)
    }
}

pub fn module(wasm: &[u8], compile: CompileConfig) -> Result<Vec<u8>> {
    let mut store = compile.store();
    let module = Module::new(&store, wasm)?;
    macro_rules! stub {
        (u8 <- $($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ -> u8 { panic!("incomplete import") })
        };
        (u32 <- $($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ -> u32 { panic!("incomplete import") })
        };
        (u64 <- $($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ -> u64 { panic!("incomplete import") })
        };
        (f32 <- $($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ -> f32 { panic!("incomplete import") })
        };
        (f64 <- $($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ -> f64 { panic!("incomplete import") })
        };
        ($($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ panic!("incomplete import"))
        };
    }
    let mut imports = imports! {
        "vm_hooks" => {
            "read_args" => stub!(|_: u32|),
            "write_result" => stub!(|_: u32, _: u32|),
            "storage_load_bytes32" => stub!(|_: u32, _: u32|),
            "storage_store_bytes32" => stub!(|_: u32, _: u32|),
            "call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u32, _: u64, _: u32|),
            "delegate_call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u64, _: u32|),
            "static_call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u64, _: u32|),
            "create1" => stub!(|_: u32, _: u32, _: u32, _: u32, _: u32|),
            "create2" => stub!(|_: u32, _: u32, _: u32, _: u32, _: u32, _: u32|),
            "read_return_data" => stub!(u32 <- |_: u32, _: u32, _: u32|),
            "return_data_size" => stub!(u32 <- ||),
            "emit_log" => stub!(|_: u32, _: u32, _: u32|),
            "account_balance" => stub!(|_: u32, _: u32|),
            "account_codehash" => stub!(|_: u32, _: u32|),
            "evm_gas_left" => stub!(u64 <- ||),
            "evm_ink_left" => stub!(u64 <- ||),
            "block_basefee" => stub!(|_: u32|),
            "chainid" => stub!(u64 <- ||),
            "block_coinbase" => stub!(|_: u32|),
            "block_gas_limit" => stub!(u64 <- ||),
            "block_number" => stub!(u64 <- ||),
            "block_timestamp" => stub!(u64 <- ||),
            "contract_address" => stub!(|_: u32|),
            "msg_reentrant" => stub!(u32 <- ||),
            "msg_sender" => stub!(|_: u32|),
            "msg_value" => stub!(|_: u32|),
            "tx_gas_price" => stub!(|_: u32|),
            "tx_ink_price" => stub!(u32 <- ||),
            "tx_origin" => stub!(|_: u32|),
            "memory_grow" => stub!(|_: u16|),
            "native_keccak256" => stub!(|_: u32, _: u32, _: u32|),
        },
    };
    if compile.debug.debug_funcs {
        imports.define("console", "log_txt", stub!(|_: u32, _: u32|));
        imports.define("console", "log_i32", stub!(|_: u32|));
        imports.define("console", "log_i64", stub!(|_: u64|));
        imports.define("console", "log_f32", stub!(|_: f32|));
        imports.define("console", "log_f64", stub!(|_: f64|));
        imports.define("console", "tee_i32", stub!(u32 <- |_: u32|));
        imports.define("console", "tee_i64", stub!(u64 <- |_: u64|));
        imports.define("console", "tee_f32", stub!(f32 <- |_: f32|));
        imports.define("console", "tee_f64", stub!(f64 <- |_: f64|));
        imports.define("debug", "null_host", stub!(||));
    }
    Instance::new(&mut store, &module, &imports)?;

    let module = module.serialize()?;
    Ok(module.to_vec())
}

pub fn compile_user_wasm(
    wasm: &[u8],
    version: u16,
    page_limit: u16,
    debug_mode: bool,
) -> Result<(Vec<u8>, Bytes32, WasmPricingInfo)> {
    let compile = CompileConfig::version(version, debug_mode);
    let (bin, stylus_data, footprint) =
        WasmBinary::parse_user(wasm, page_limit, &compile).wrap_err("failed to parse program")?;

    let canonical_hash = prover::machine::Module::from_user_binary(
        &bin,
        compile.debug.debug_funcs,
        Some(stylus_data),
    )
    .wrap_err("failed to build module from program")?
    .hash();

    let info = WasmPricingInfo {
        size: wasm.len().try_into()?,
        footprint: footprint,
    };
    let module = module(wasm, compile).wrap_err("failed generating stylus module")?;

    Ok((module, canonical_hash, info))
}
