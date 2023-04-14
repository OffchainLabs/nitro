// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::env::EvmData;
use crate::{
    env::{MeterData, WasmEnv},
    host, GoApi, GoApiStatus, RustVec,
};
use arbutil::{operator::OperatorCode, Color};
use eyre::{bail, eyre, ErrReport, Result};
use prover::{
    programs::{
        counter::{Counter, CountingMachine, OP_OFFSETS},
        depth::STYLUS_STACK_LEFT,
        meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS},
        prelude::*,
        start::STYLUS_START,
    },
    utils::Bytes20,
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

    pub fn read_slice(&self, mem: &str, ptr: usize, len: usize) -> Result<Vec<u8>> {
        let memory = self.exports.get_memory(mem)?;
        let memory = memory.view(&self.store);
        let mut data = vec![0; len];
        memory.read(ptr as u64, &mut data)?;
        Ok(data)
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
        macro_rules! func {
            ($func:expr) => {
                Function::new_typed_with_env(&mut store, &func_env, $func)
            };
        }
        let mut imports = imports! {
            "forward" => {
                "read_args" => func!(host::read_args),
                "return_data" => func!(host::return_data),
                "account_load_bytes32" => func!(host::account_load_bytes32),
                "account_store_bytes32" => func!(host::account_store_bytes32),
                "call_contract" => func!(host::call_contract),
                "delegate_call_contract" => func!(host::delegate_call_contract),
                "static_call_contract" => func!(host::static_call_contract),
                "create1" => func!(host::create1),
                "create2" => func!(host::create2),
                "read_return_data" => func!(host::read_return_data),
                "return_data_size" => func!(host::return_data_size),
                "emit_log" => func!(host::emit_log),
                "block_basefee" => func!(host::block_basefee),
                "block_chainid" => func!(host::block_chainid),
                "block_coinbase" => func!(host::block_coinbase),
                "block_difficulty" => func!(host::block_difficulty),
                "block_gas_limit" => func!(host::block_gas_limit),
                "block_number" => func!(host::block_number),
                "block_timestamp" => func!(host::block_timestamp),
                "msg_sender" => func!(host::msg_sender),
                "msg_value" => func!(host::msg_value),
                "tx_gas_price" => func!(host::tx_gas_price),
                "tx_origin" => func!(host::tx_origin),
            },
        };
        if debug_funcs {
            imports.define("console", "log_txt", func!(host::console_log_text));
            imports.define("console", "log_i32", func!(host::console_log::<u32>));
            imports.define("console", "log_i64", func!(host::console_log::<u64>));
            imports.define("console", "log_f32", func!(host::console_log::<f32>));
            imports.define("console", "log_f64", func!(host::console_log::<f64>));
            imports.define("console", "tee_i32", func!(host::console_tee::<u32>));
            imports.define("console", "tee_i64", func!(host::console_tee::<u64>));
            imports.define("console", "tee_f32", func!(host::console_tee::<f32>));
            imports.define("console", "tee_f64", func!(host::console_tee::<f64>));
        }
        let instance = Instance::new(&mut store, &module, &imports)?;
        let exports = &instance.exports;
        let memory = exports.get_memory("memory")?.clone();

        let expect_global = |name| -> Global { instance.exports.get_global(name).unwrap().clone() };
        let ink_left = expect_global(STYLUS_INK_LEFT);
        let ink_status = expect_global(STYLUS_INK_STATUS);

        let env = func_env.as_mut(&mut store);
        env.memory = Some(memory);
        env.meter = Some(MeterData {
            ink_left,
            ink_status,
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

    pub fn set_go_api(&mut self, api: GoApi) {
        let env = self.env.as_mut(&mut self.store);
        use GoApiStatus::*;

        macro_rules! ptr {
            ($expr:expr) => {
                &mut $expr as *mut _
            };
        }
        macro_rules! error {
            ($data:expr) => {
                ErrReport::msg(String::from_utf8_lossy(&$data).to_string())
            };
        }

        let get_bytes32 = api.get_bytes32;
        let set_bytes32 = api.set_bytes32;
        let contract_call = api.contract_call;
        let delegate_call = api.delegate_call;
        let static_call = api.static_call;
        let create1 = api.create1;
        let create2 = api.create2;
        let get_return_data = api.get_return_data;
        let emit_log = api.emit_log;
        let id = api.id;

        let get_bytes32 = Box::new(move |key| unsafe {
            let mut cost = 0;
            let value = get_bytes32(id, key, ptr!(cost));
            (value, cost)
        });
        let set_bytes32 = Box::new(move |key, value| unsafe {
            let mut error = RustVec::new(vec![]);
            let mut cost = 0;
            let api_status = set_bytes32(id, key, value, ptr!(cost), ptr!(error));
            let error = error.into_vec(); // done here to always drop
            match api_status {
                Success => Ok(cost),
                Failure => Err(error!(error)),
            }
        });
        let contract_call = Box::new(move |contract, calldata, gas, value| unsafe {
            let mut call_gas = gas; // becomes the call's cost
            let mut return_data_len = 0;
            let api_status = contract_call(
                id,
                contract,
                ptr!(RustVec::new(calldata)),
                ptr!(call_gas),
                value,
                ptr!(return_data_len),
            );
            (return_data_len, call_gas, api_status.into())
        });
        let delegate_call = Box::new(move |contract, calldata, gas| unsafe {
            let mut call_gas = gas; // becomes the call's cost
            let mut return_data_len = 0;
            let api_status = delegate_call(
                id,
                contract,
                ptr!(RustVec::new(calldata)),
                ptr!(call_gas),
                ptr!(return_data_len),
            );
            (return_data_len, call_gas, api_status.into())
        });
        let static_call = Box::new(move |contract, calldata, gas| unsafe {
            let mut call_gas = gas; // becomes the call's cost
            let mut return_data_len = 0;
            let api_status = static_call(
                id,
                contract,
                ptr!(RustVec::new(calldata)),
                ptr!(call_gas),
                ptr!(return_data_len),
            );
            (return_data_len, call_gas, api_status.into())
        });
        let create1 = Box::new(move |code, endowment, gas| unsafe {
            let mut call_gas = gas; // becomes the call's cost
            let mut return_data_len = 0;
            let mut code = RustVec::new(code);
            let api_status = create1(
                id,
                ptr!(code),
                endowment,
                ptr!(call_gas),
                ptr!(return_data_len),
            );
            let output = code.into_vec();
            let result = match api_status {
                Success => Ok(Bytes20::try_from(output).unwrap()),
                Failure => Err(error!(output)),
            };
            (result, return_data_len, call_gas)
        });
        let create2 = Box::new(move |code, endowment, salt, gas| unsafe {
            let mut call_gas = gas; // becomes the call's cost
            let mut return_data_len = 0;
            let mut code = RustVec::new(code);
            let api_status = create2(
                id,
                ptr!(code),
                endowment,
                salt,
                ptr!(call_gas),
                ptr!(return_data_len),
            );
            let output = code.into_vec();
            let result = match api_status {
                Success => Ok(Bytes20::try_from(output).unwrap()),
                Failure => Err(error!(output)),
            };
            (result, return_data_len, call_gas)
        });
        let get_return_data = Box::new(move || unsafe {
            let mut data = RustVec::new(vec![]);
            get_return_data(id, ptr!(data));
            data.into_vec()
        });
        let emit_log = Box::new(move |data, topics| unsafe {
            let mut data = RustVec::new(data);
            let api_status = emit_log(id, ptr!(data), topics);
            let error = data.into_vec(); // done here to always drop
            match api_status {
                Success => Ok(()),
                Failure => Err(error!(error)),
            }
        });

        env.set_evm_api(
            get_bytes32,
            set_bytes32,
            contract_call,
            delegate_call,
            static_call,
            create1,
            create2,
            get_return_data,
            emit_log,
        )
    }

    pub fn set_evm_data(&mut self, evm_data: EvmData) {
        let env = self.env.as_mut(&mut self.store);
        env.evm_data = Some(evm_data);
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
    fn ink_left(&mut self) -> MachineMeter {
        let status = self.get_global(STYLUS_INK_STATUS).unwrap();
        let mut ink = || self.get_global(STYLUS_INK_LEFT).unwrap();

        match status {
            0 => MachineMeter::Ready(ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_ink(&mut self, ink: u64) {
        self.set_global(STYLUS_INK_LEFT, ink).unwrap();
        self.set_global(STYLUS_INK_STATUS, 0).unwrap();
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
        "forward" => {
            "read_args" => stub!(|_: u32|),
            "return_data" => stub!(|_: u32, _: u32|),
            "account_load_bytes32" => stub!(|_: u32, _: u32|),
            "account_store_bytes32" => stub!(|_: u32, _: u32|),
            "call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u32, _: u64, _: u32|),
            "delegate_call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u64, _: u32|),
            "static_call_contract" => stub!(u8 <- |_: u32, _: u32, _: u32, _: u64, _: u32|),
            "create1" => stub!(|_: u32, _: u32, _: u32, _: u32, _: u32|),
            "create2" => stub!(|_: u32, _: u32, _: u32, _: u32, _: u32, _: u32|),
            "read_return_data" => stub!(|_: u32|),
            "return_data_size" => stub!(u32 <- ||),
            "emit_log" => stub!(|_: u32, _: u32, _: u32|),
            "block_basefee" => stub!(|_: u32|),
            "block_chainid" => stub!(|_: u32|),
            "block_coinbase" => stub!(|_: u32|),
            "block_difficulty" => stub!(|_: u32|),
            "block_gas_limit" => stub!(u64 <- ||),
            "block_number" => stub!(|_: u32|),
            "block_timestamp" => stub!(|_: u32|),
            "msg_sender" => stub!(|_: u32|),
            "msg_value" => stub!(|_: u32|),
            "tx_gas_price" => stub!(|_: u32|),
            "tx_origin" => stub!(|_: u32|),
        },
    };
    if config.debug.debug_funcs {
        imports.define("console", "log_txt", stub!(|_: u32, _: u32|));
        imports.define("console", "log_i32", stub!(|_: u32|));
        imports.define("console", "log_i64", stub!(|_: u64|));
        imports.define("console", "log_f32", stub!(|_: f32|));
        imports.define("console", "log_f64", stub!(|_: f64|));
        imports.define("console", "tee_i32", stub!(u32 <- |_: u32|));
        imports.define("console", "tee_i64", stub!(u64 <- |_: u64|));
        imports.define("console", "tee_f32", stub!(f32 <- |_: f32|));
        imports.define("console", "tee_f64", stub!(f64 <- |_: f64|));
    }
    Instance::new(&mut store, &module, &imports)?;

    let module = module.serialize()?;
    Ok(module.to_vec())
}
