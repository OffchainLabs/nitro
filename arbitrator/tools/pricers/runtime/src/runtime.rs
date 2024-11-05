// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::host::{self, Escape, PricerEnv};
use arbutil::{Color, Bytes32};
//use arbutil::{operator::OperatorCode, Color};
use eyre::{bail, eyre, ErrReport, Result};
use prover::programs::{
    //    counter::Counter,
    depth::STYLUS_STACK_LEFT,
    meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS},
    prelude::*,
    start::{StartMover, StartlessMachine},
    MiddlewareWrapper,
    STYLUS_ENTRY_POINT
};
use std::{fmt::Debug, sync::Arc, time::Duration};
use stylus::env::MeterData;
use wasmer::{
    imports, AsStoreMut, CompilerConfig, sys::EngineBuilder, Function, FunctionEnv, Imports, Instance, Module, Store, Target, TypedFunction, Value
};
use wasmer_compiler_singlepass::Singlepass;
use wasmer_vm::VMExtern;

pub struct Runtime {
    pub instance: Instance,
    pub store: Store,
    pub env: FunctionEnv<PricerEnv>,
}

impl Runtime {
    pub fn new(wasm: &[u8], compile: CompileConfig, target: Target) -> Result<Runtime> {
        let mut store = compile.store(target);
        let (env, imports) = Self::new_env(&mut store);

        let module = Module::new(&store, wasm)?;
        let instance = Instance::new(&mut store, &module, &imports)?;
        let exports = &instance.exports;

        let mut expect_global = |name| {
            let VMExtern::Global(sh) = exports.get_extern(name).unwrap().to_vm_extern() else {
                panic!("name not found global");
            };
            sh.get(store.objects_mut()).vmglobal()
        };
        let ink_left = expect_global(STYLUS_INK_LEFT);
        let ink_status = expect_global(STYLUS_INK_STATUS);

        env.as_mut(&mut store).meter = Some(MeterData {
            ink_left,
            ink_status,
        });

        Ok(Self {
            instance,
            store,
            env,
        })
    }

    pub fn new_simple(wasm: &[u8], target: Target) -> Result<Self> {
        let mut compiler = Singlepass::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();

        let start = MiddlewareWrapper::new(StartMover::new(true));
        compiler.push_middleware(Arc::new(start));

        let mut store = Store::new(EngineBuilder::new(compiler).set_target(Some(target)));
        //let mut store = Store::new(compiler);

        let (env, imports) = Self::new_env(&mut store);
        let module = Module::new(&store, wasm)?;
        let instance = Instance::new(&mut store, &module, &imports)?;

        Ok(Self {
            instance,
            store,
            env,
        })
    }

    fn new_env(store: &mut Store) -> (FunctionEnv<PricerEnv>, Imports) {
        let env = FunctionEnv::new(store, PricerEnv::default());
        macro_rules! func {
            ($func:expr) => {
                Function::new_typed_with_env(store, &env, $func)
            };
        }
        let imports = imports! {
            "vm_hooks" => {
                "memory_grow" => func!(host::memory_grow),
            },
            "pricer" => {
                "toggle_timer" => func!(host::toggle_timer),
            }
        };
        (env, imports)
    }

    pub fn env(&self) -> &PricerEnv {
        self.env.as_ref(&self.store)
    }

    pub fn env_mut(&mut self) -> &mut PricerEnv {
        self.env.as_mut(&mut self.store)
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

    pub fn time(&self) -> (Duration, u64) {
        let time = self.env().elapsed.unwrap();
        let cycles = self.env().cycles_total.unwrap();
        (time, cycles)
    }

    pub fn run(&mut self, config: StylusConfig, ink: u64) -> Result<Escape> {
        self.set_ink(arbutil::evm::api::Ink(ink));
        self.set_stack(config.max_depth);

        let start = self.get_start()?;
        Ok(match start.call(&mut self.store) {
            Ok(_) => Escape::Incomplete,
            Err(outcome) => match outcome.downcast() {
                Ok(escape) => escape,
                Err(error) => bail!("error: {}", error),
            },
        })
    }
}

impl StartlessMachine for Runtime {
    fn get_start(&self) -> Result<TypedFunction<(), ()>> {
        let store = &self.store;
        let exports = &self.instance.exports;
        exports
            .get_typed_function(store, StartMover::NAME)
            .map_err(ErrReport::new)
    }
}

impl MeteredMachine for Runtime {
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

impl DepthCheckedMachine for Runtime {
    fn stack_left(&mut self) -> u32 {
        self.get_global(STYLUS_STACK_LEFT).unwrap()
    }

    fn set_stack(&mut self, size: u32) {
        self.set_global(STYLUS_STACK_LEFT, size).unwrap()
    }
}

/*impl CountingMachine for Runtime {
    fn operator_count(&mut self, op: OperatorCode) -> Result<usize> {
        let count: u64 = self.get_global(&Counter::global_name(op.seq()))?;
        Ok(count as usize)
    }
}
*/
