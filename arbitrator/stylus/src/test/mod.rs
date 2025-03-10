// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{env::WasmEnv, native::NativeInstance, run::RunProgram, test::api::TestEvmApi};
use arbutil::{
    evm::{
        api::{Ink, VecReader},
        user::UserOutcome,
    },
    Bytes20, Bytes32, Color,
};
use eyre::{bail, Result};
use prover::{
    machine::GlobalState,
    programs::{config::SigMap, prelude::*},
    Machine,
};
use rand::prelude::*;
use std::{collections::HashMap, path::Path, sync::Arc};
use wasmer::{
    imports, wasmparser::Operator, CompilerConfig, Function, FunctionEnv, Imports, Instance,
    Module, Store, Target,
};
use wasmer_compiler_singlepass::Singlepass;

mod api;
mod misc;
mod native;
mod sdk;
mod wavm;

#[cfg(feature = "timings")]
mod timings;

type TestInstance = NativeInstance<VecReader, TestEvmApi>;

impl TestInstance {
    fn new_test(path: &str, compile: CompileConfig) -> Result<Self> {
        let mut store = compile.store(Target::default());
        let imports = imports! {
            "test" => {
                "noop" => Function::new_typed(&mut store, || {}),
            },
        };
        let mut native = Self::new_from_store(path, store, imports)?;
        native.set_meter_data();
        native.set_ink(Ink(u64::MAX));
        native.set_stack(u32::MAX);
        Ok(native)
    }

    fn new_from_store(path: &str, mut store: Store, imports: Imports) -> Result<Self> {
        let wat = std::fs::read(path)?;
        let module = Module::new(&store, wat)?;
        let native = Instance::new(&mut store, &module, &imports)?;
        Ok(Self::new_sans_env(native, store))
    }

    fn new_vanilla(path: &str) -> Result<Self> {
        let mut compiler = Singlepass::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();

        let mut store = Store::new(compiler);
        let wat = std::fs::read(path)?;
        let module = Module::new(&store, wat)?;
        let instance = Instance::new(&mut store, &module, &Imports::new())?;
        Ok(Self::new_sans_env(instance, store))
    }

    fn new_sans_env(instance: Instance, mut store: Store) -> Self {
        let compile = CompileConfig::default();
        let (evm, evm_data) = TestEvmApi::new(compile.clone());
        let env = FunctionEnv::new(&mut store, WasmEnv::new(compile, None, evm, evm_data));
        Self::new(instance, store, env)
    }

    fn new_linked(
        path: impl AsRef<str>,
        compile: &CompileConfig,
        config: StylusConfig,
    ) -> Result<Self> {
        Self::new_with_evm(path.as_ref(), compile, config).map(|x| x.0)
    }

    fn new_with_evm(
        path: &str,
        compile: &CompileConfig,
        config: StylusConfig,
    ) -> Result<(Self, TestEvmApi)> {
        let (mut evm, evm_data) = TestEvmApi::new(compile.clone());
        let native = Self::from_path(
            path,
            evm.clone(),
            evm_data,
            compile,
            config,
            Target::default(),
        )?;
        let footprint = native.memory().ty(&native.store).minimum.0 as u16;
        evm.set_pages(footprint);
        Ok((native, evm))
    }
}

fn expensive_add(op: &Operator, _tys: &SigMap) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
}

pub fn random_ink(min: u64) -> Ink {
    Ink(rand::thread_rng().gen_range(min..=u64::MAX))
}

pub fn random_bytes20() -> Bytes20 {
    let mut data = [0; 20];
    rand::thread_rng().fill_bytes(&mut data);
    data.into()
}

fn random_bytes32() -> Bytes32 {
    let mut data = [0; 32];
    rand::thread_rng().fill_bytes(&mut data);
    data.into()
}

fn test_compile_config() -> CompileConfig {
    let mut compile_config = CompileConfig::version(0, true);
    compile_config.debug.count_ops = true;
    compile_config
}

fn uniform_cost_config() -> StylusConfig {
    let mut stylus_config = StylusConfig::default();
    stylus_config.pricing.ink_price = 10000;
    stylus_config
}

fn test_configs() -> (CompileConfig, StylusConfig, Ink) {
    (
        test_compile_config(),
        uniform_cost_config(),
        random_ink(1_000_000),
    )
}

fn new_test_machine(path: &str, compile: &CompileConfig) -> Result<Machine> {
    let wat = std::fs::read(path)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let mut bin = prover::binary::parse(&wasm, Path::new("user"))?;
    let stylus_data = bin.instrument(compile, &Bytes32::default())?;

    let wat = std::fs::read("tests/test.wat")?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let lib = prover::binary::parse(&wasm, Path::new("test"))?;

    let mut mach = Machine::from_binaries(
        &[lib],
        bin,
        false,
        true,
        compile.debug.debug_funcs,
        true,
        GlobalState::default(),
        HashMap::default(),
        Arc::new(|_, _, _| panic!("tried to read preimage")),
        Some(stylus_data),
    )?;
    mach.set_ink(Ink(u64::MAX));
    mach.set_stack(u32::MAX);
    Ok(mach)
}

fn run_native(native: &mut TestInstance, args: &[u8], ink: Ink) -> Result<Vec<u8>> {
    let config = native.env().config.expect("no config");
    match native.run_main(args, config, ink)? {
        UserOutcome::Success(output) => Ok(output),
        err => bail!("user program failure: {}", err.red()),
    }
}

fn run_machine(
    machine: &mut Machine,
    args: &[u8],
    config: StylusConfig,
    ink: Ink,
) -> Result<Vec<u8>> {
    match machine.run_main(args, config, ink)? {
        UserOutcome::Success(output) => Ok(output),
        err => bail!("user program failure: {}", err.red()),
    }
}

fn check_instrumentation(mut native: TestInstance, mut machine: Machine) -> Result<()> {
    assert_eq!(native.ink_left(), machine.ink_left());
    assert_eq!(native.stack_left(), machine.stack_left());

    let native_counts = native.operator_counts()?;
    let machine_counts = machine.operator_counts()?;
    assert_eq!(native_counts.get(&Operator::Unreachable.into()), None);
    assert_eq!(native_counts, machine_counts);
    Ok(())
}
