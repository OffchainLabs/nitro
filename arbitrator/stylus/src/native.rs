// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    env::{MeterData, WasmEnv},
    host, GoAPI, RustVec,
};
use arbutil::{operator::OperatorCode, Color};
use eyre::{bail, eyre, ErrReport, Result};
use prover::programs::{
    counter::{Counter, CountingMachine, OP_OFFSETS},
    depth::STYLUS_STACK_LEFT,
    meter::{STYLUS_GAS_LEFT, STYLUS_GAS_STATUS},
    prelude::*,
    start::STYLUS_START,
};
use std::{
    collections::BTreeMap,
    fmt::Debug,
    ops::{Deref, DerefMut},
};
use wasmer::{
    imports, AsStoreMut, Function, FunctionEnv, Global, Instance, Module, Store, TypedFunction,
    Value,
};

pub struct NativeInstance {
    pub instance: Instance,
    pub store: Store,
    pub env: FunctionEnv<WasmEnv>,
}

impl NativeInstance {
    pub fn new(instance: Instance, store: Store, env: FunctionEnv<WasmEnv>) -> Self {
        Self {
            instance,
            store,
            env,
        }
    }

    pub fn new_sans_env(instance: Instance, mut store: Store) -> Self {
        let env = FunctionEnv::new(&mut store, WasmEnv::default());
        Self::new(instance, store, env)
    }

    pub fn env(&self) -> &WasmEnv {
        self.env.as_ref(&self.store)
    }

    pub fn env_mut(&mut self) -> &mut WasmEnv {
        self.env.as_mut(&mut self.store)
    }

    pub fn config(&self) -> StylusConfig {
        self.env().config.clone()
    }

    /// Creates a `NativeInstance` from a serialized module
    /// Safety: module bytes must represent a module
    pub unsafe fn deserialize(module: &[u8], config: StylusConfig) -> Result<Self> {
        let env = WasmEnv::new(config);
        let store = env.config.store();
        let module = Module::deserialize(&store, module)?;
        Self::from_module(module, store, env)
    }

    pub fn from_path(path: &str, config: &StylusConfig) -> Result<Self> {
        let env = WasmEnv::new(config.clone());
        let store = env.config.store();
        let wat_or_wasm = std::fs::read(path)?;
        let module = Module::new(&store, wat_or_wasm)?;
        Self::from_module(module, store, env)
    }

    fn from_module(module: Module, mut store: Store, env: WasmEnv) -> Result<Self> {
        let debug_funcs = env.config.debug.debug_funcs;
        let func_env = FunctionEnv::new(&mut store, env);
        let mut imports = imports! {
            "forward" => {
                "read_args" => Function::new_typed_with_env(&mut store, &func_env, host::read_args),
                "return_data" => Function::new_typed_with_env(&mut store, &func_env, host::return_data),
                "account_load_bytes32" => Function::new_typed_with_env(&mut store, &func_env, host::account_load_bytes32),
                "account_store_bytes32" => Function::new_typed_with_env(&mut store, &func_env, host::account_store_bytes32),
            },
        };
        if debug_funcs {
            imports.define(
                "forward",
                "debug_println",
                Function::new_typed_with_env(&mut store, &func_env, host::debug_println),
            );
        }
        let instance = Instance::new(&mut store, &module, &imports)?;
        let exports = &instance.exports;
        let memory = exports.get_memory("memory")?.clone();

        let expect_global = |name| -> Global { instance.exports.get_global(name).unwrap().clone() };
        let gas_left = expect_global(STYLUS_GAS_LEFT);
        let gas_status = expect_global(STYLUS_GAS_STATUS);

        let env = func_env.as_mut(&mut store);
        env.memory = Some(memory);
        env.meter = Some(MeterData {
            gas_left,
            gas_status,
            pricing: env.config.pricing,
        });
        Ok(Self::new(instance, store, func_env))
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

    pub fn set_go_api(&mut self, api: GoAPI) {
        let env = self.env.as_mut(&mut self.store);

        let get = api.get_bytes32;
        let set = api.set_bytes32;
        let id = api.id;

        let get_bytes32 = Box::new(move |key| unsafe {
            let mut cost = 0;
            let value = get(id, key, &mut cost as *mut _);
            (value, cost)
        });
        let set_bytes32 = Box::new(move |key, value| unsafe {
            let mut error = RustVec::new(vec![]);
            let mut cost = 0;
            let status = set(id, key, value, &mut cost as *mut _, &mut error as *mut _);
            let error = error.into_vec(); // done here to always drop
            match status {
                0 => Ok(cost),
                _ => Err(ErrReport::msg(String::from_utf8_lossy(&error).to_string())),
            }
        });
        let call_contract =
            Box::new(move |_contract, _input, _gas, _value| unimplemented!("contract call"));

        env.set_evm_api(get_bytes32, set_bytes32, call_contract)
    }
}

impl Deref for NativeInstance {
    type Target = Instance;

    fn deref(&self) -> &Self::Target {
        &self.instance
    }
}

impl DerefMut for NativeInstance {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.instance
    }
}

impl MeteredMachine for NativeInstance {
    fn gas_left(&mut self) -> MachineMeter {
        let status = self.get_global(STYLUS_GAS_STATUS).unwrap();
        let mut gas = || self.get_global(STYLUS_GAS_LEFT).unwrap();

        match status {
            0 => MachineMeter::Ready(gas()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, gas: u64) {
        self.set_global(STYLUS_GAS_LEFT, gas).unwrap();
        self.set_global(STYLUS_GAS_STATUS, 0).unwrap();
    }
}

impl CountingMachine for NativeInstance {
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

impl DepthCheckedMachine for NativeInstance {
    fn stack_left(&mut self) -> u32 {
        self.get_global(STYLUS_STACK_LEFT).unwrap()
    }

    fn set_stack(&mut self, size: u32) {
        self.set_global(STYLUS_STACK_LEFT, size).unwrap()
    }
}

impl StartlessMachine for NativeInstance {
    fn get_start(&self) -> Result<TypedFunction<(), ()>> {
        let store = &self.store;
        let exports = &self.instance.exports;
        exports
            .get_typed_function(store, STYLUS_START)
            .map_err(ErrReport::new)
    }
}

pub fn module(wasm: &[u8], config: StylusConfig) -> Result<Vec<u8>> {
    let mut store = config.store();
    let module = Module::new(&store, wasm)?;
    macro_rules! stub {
        ($($types:tt)+) => {
            Function::new_typed(&mut store, $($types)+ panic!("incomplete import"))
        };
    }
    let mut imports = imports! {
        "forward" => {
            "read_args" => stub!(|_: u32|),
            "return_data" => stub!(|_: u32, _: u32|),
            "account_load_bytes32" => stub!(|_: u32, _: u32|),
            "account_store_bytes32" => stub!(|_: u32, _: u32|),
        },
    };
    if config.debug.debug_funcs {
        imports.define("forward", "debug_println", stub!(|_: u32, _: u32|));
    }
    Instance::new(&mut store, &module, &imports)?;

    let module = module.serialize()?;
    Ok(module.to_vec())
}
