use arbos::programs::types::EvmData;
use eyre::Result;
use std::ops::{Deref, DerefMut};
use wasmer::{
    imports, Function, FunctionEnv, Instance, Memory, Module, Store, TypedFunction, Value,
};

use crate::cache::InitCache;
use crate::config::{CompileConfig, PricingParams, StylusConfig};
use crate::env::{MeterData, WasmEnv};
use crate::evm_api::EvmApi;
use crate::host;
use crate::ink::Ink;
use crate::meter::{
    DepthCheckedMachine, GasMeteredMachine, MachineMeter, MeteredMachine, STYLUS_INK_LEFT,
    STYLUS_INK_STATUS, STYLUS_STACK_LEFT,
};

/// A native WASM instance ready for execution.
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

    /// Create from a serialized module with caching.
    ///
    /// # Safety
    ///
    /// `module` must represent a valid serialized module.
    pub unsafe fn deserialize_cached(
        module: &[u8],
        version: u16,
        evm: E,
        evm_data: EvmData,
        mut long_term_tag: u32,
        debug: bool,
    ) -> Result<Self> {
        let compile = CompileConfig::version(version, debug);
        let env = WasmEnv::new(compile, None, evm, evm_data);
        let module_hash = env.evm_data.module_hash;
        if !env.evm_data.cached {
            long_term_tag = 0;
        }
        if let Some((module, store)) = InitCache::get(module_hash, version, long_term_tag, debug) {
            return Self::from_module(module, store, env);
        }
        let (module, store) =
            InitCache::insert(module_hash, module, version, long_term_tag, debug)?;
        Self::from_module(module, store, env)
    }

    /// Create from WASM bytes (compiles the module).
    pub fn from_bytes(
        bytes: impl AsRef<[u8]>,
        evm_api: E,
        evm_data: EvmData,
        compile: &CompileConfig,
        config: StylusConfig,
    ) -> Result<Self> {
        let env = WasmEnv::new(compile.clone(), Some(config), evm_api, evm_data);
        let store = env.compile.store();
        let module = Module::new(&store, bytes)?;
        Self::from_module(module, store, env)
    }

    pub fn from_module(module: Module, mut store: Store, env: WasmEnv<E>) -> Result<Self> {
        let debug_funcs = env.compile.debug.debug_funcs;
        let func_env = FunctionEnv::new(&mut store, env);

        macro_rules! func {
            ($func:expr) => {
                Function::new_typed_with_env(&mut store, &func_env, $func)
            };
        }

        let mut import_object = imports! {
            "vm_hooks" => {
                "read_args" => func!(host::read_args::<E>),
                "write_result" => func!(host::write_result::<E>),
                "exit_early" => func!(host::exit_early::<E>),
                "storage_load_bytes32" => func!(host::storage_load_bytes32::<E>),
                "storage_cache_bytes32" => func!(host::storage_cache_bytes32::<E>),
                "storage_flush_cache" => func!(host::storage_flush_cache::<E>),
                "transient_load_bytes32" => func!(host::transient_load_bytes32::<E>),
                "transient_store_bytes32" => func!(host::transient_store_bytes32::<E>),
                "call_contract" => func!(host::call_contract::<E>),
                "delegate_call_contract" => func!(host::delegate_call_contract::<E>),
                "static_call_contract" => func!(host::static_call_contract::<E>),
                "create1" => func!(host::create1::<E>),
                "create2" => func!(host::create2::<E>),
                "read_return_data" => func!(host::read_return_data::<E>),
                "return_data_size" => func!(host::return_data_size::<E>),
                "emit_log" => func!(host::emit_log::<E>),
                "account_balance" => func!(host::account_balance::<E>),
                "account_code" => func!(host::account_code::<E>),
                "account_codehash" => func!(host::account_codehash::<E>),
                "account_code_size" => func!(host::account_code_size::<E>),
                "evm_gas_left" => func!(host::evm_gas_left::<E>),
                "evm_ink_left" => func!(host::evm_ink_left::<E>),
                "block_basefee" => func!(host::block_basefee::<E>),
                "chainid" => func!(host::chainid::<E>),
                "block_coinbase" => func!(host::block_coinbase::<E>),
                "block_gas_limit" => func!(host::block_gas_limit::<E>),
                "block_number" => func!(host::block_number::<E>),
                "block_timestamp" => func!(host::block_timestamp::<E>),
                "contract_address" => func!(host::contract_address::<E>),
                "math_div" => func!(host::math_div::<E>),
                "math_mod" => func!(host::math_mod::<E>),
                "math_pow" => func!(host::math_pow::<E>),
                "math_add_mod" => func!(host::math_add_mod::<E>),
                "math_mul_mod" => func!(host::math_mul_mod::<E>),
                "msg_reentrant" => func!(host::msg_reentrant::<E>),
                "msg_sender" => func!(host::msg_sender::<E>),
                "msg_value" => func!(host::msg_value::<E>),
                "tx_gas_price" => func!(host::tx_gas_price::<E>),
                "tx_ink_price" => func!(host::tx_ink_price::<E>),
                "tx_origin" => func!(host::tx_origin::<E>),
                "pay_for_memory_grow" => func!(host::pay_for_memory_grow::<E>),
                "native_keccak256" => func!(host::native_keccak256::<E>),
            },
        };

        if debug_funcs {
            import_object.define(
                "console",
                "log_txt",
                func!(host::console_log_text::<E>),
            );
            import_object.define("console", "log_i32", func!(host::console_log::<E, u32>));
            import_object.define("console", "log_i64", func!(host::console_log::<E, u64>));
            import_object.define("console", "log_f32", func!(host::console_log::<E, f32>));
            import_object.define("console", "log_f64", func!(host::console_log::<E, f64>));
            import_object.define("console", "tee_i32", func!(host::console_tee::<E, u32>));
            import_object.define("console", "tee_i64", func!(host::console_tee::<E, u64>));
            import_object.define("console", "tee_f32", func!(host::console_tee::<E, f32>));
            import_object.define("console", "tee_f64", func!(host::console_tee::<E, f64>));
            import_object.define("debug", "null_host", func!(host::null_host::<E>));
            import_object.define(
                "debug",
                "start_benchmark",
                func!(host::start_benchmark::<E>),
            );
            import_object.define("debug", "end_benchmark", func!(host::end_benchmark::<E>));
        }

        let instance = Instance::new(&mut store, &module, &import_object)?;
        let memory = instance.exports.get_memory("memory")?.clone();

        let env = func_env.as_mut(&mut store);
        env.memory = Some(memory);

        let mut native = Self::new(instance, store, func_env);
        native.set_meter_data();
        Ok(native)
    }

    pub fn set_meter_data(&mut self) {
        self.env_mut().meter = Some(MeterData::new());
    }

    /// Sync meter data from WASM globals.
    pub(crate) fn sync_meter_from_globals(&mut self) {
        let mut ink_val = 0u64;
        let mut status_val = 0u32;
        {
            let store = &mut self.store;
            let exports = &self.instance.exports;
            if let Ok(ink_left) = exports.get_global(STYLUS_INK_LEFT) {
                if let Value::I64(v) = ink_left.get(store) {
                    ink_val = v as u64;
                }
            }
            if let Ok(ink_status) = exports.get_global(STYLUS_INK_STATUS) {
                if let Value::I32(v) = ink_status.get(store) {
                    status_val = v as u32;
                }
            }
        }
        if let Some(meter) = self.env_mut().meter.as_mut() {
            meter.set_ink(Ink(ink_val));
            meter.set_status(status_val);
        }
    }

    /// Push meter data to WASM globals.
    pub(crate) fn sync_meter_to_globals(&mut self) {
        let meter_data = self.env().meter.as_ref().map(|m| (m.ink(), m.status()));
        if let Some((ink, status)) = meter_data {
            let store = &mut self.store;
            let exports = &self.instance.exports;
            if let Ok(g) = exports.get_global(STYLUS_INK_LEFT) {
                let _ = g.set(store, Value::I64(ink.0 as i64));
            }
            if let Ok(g) = exports.get_global(STYLUS_INK_STATUS) {
                let _ = g.set(store, Value::I32(status as i32));
            }
        }
    }

    pub fn get_global<T>(&mut self, name: &str) -> Result<T>
    where
        T: TryFrom<Value>,
        T::Error: std::fmt::Debug,
    {
        let store = &mut self.store;
        let global = self
            .instance
            .exports
            .get_global(name)
            .map_err(|_| eyre::eyre!("global {name} does not exist"))?;
        global
            .get(store)
            .try_into()
            .map_err(|_| eyre::eyre!("global {name} has wrong type"))
    }

    pub fn set_global<T>(&mut self, name: &str, value: T) -> Result<()>
    where
        T: Into<Value>,
    {
        let store = &mut self.store;
        let global = self
            .instance
            .exports
            .get_global(name)
            .map_err(|_| eyre::eyre!("global {name} does not exist"))?;
        global
            .set(store, value.into())
            .map_err(|e| eyre::eyre!("{e}"))
    }

    pub fn call_func<R>(&mut self, func: TypedFunction<(), R>, ink: Ink) -> Result<R>
    where
        R: wasmer::WasmTypeList,
    {
        self.set_ink(ink);
        self.sync_meter_to_globals();
        let result = func.call(&mut self.store)?;
        self.sync_meter_from_globals();
        Ok(result)
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

impl<E: EvmApi> GasMeteredMachine for NativeInstance<E> {
    fn pricing(&self) -> PricingParams {
        self.env().config.unwrap().pricing
    }
}

impl<E: EvmApi> DepthCheckedMachine for NativeInstance<E> {
    fn stack_left(&mut self) -> u32 {
        self.get_global(STYLUS_STACK_LEFT).unwrap_or(0)
    }

    fn set_stack(&mut self, size: u32) {
        let _ = self.set_global(STYLUS_STACK_LEFT, size);
    }
}

/// Compile WASM bytes into a serialized module.
pub fn compile_module(
    wasm: &[u8],
    version: u16,
    debug: bool,
) -> Result<Vec<u8>> {
    let compile = CompileConfig::version(version, debug);
    let store = compile.store();
    let module = Module::new(&store, wasm)?;
    let serialized = module.serialize()?;
    Ok(serialized.to_vec())
}
