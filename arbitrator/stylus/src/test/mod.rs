// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::native::NativeInstance;
use eyre::Result;
use prover::{
    machine::GlobalState,
    programs::{counter::CountingMachine, prelude::*},
    utils::{Bytes20, Bytes32},
    Machine,
};
use rand::prelude::*;
use std::{collections::HashMap, path::Path, sync::Arc};
use wasmer::{imports, wasmparser::Operator, Function, Imports, Instance, Module, Store};

mod api;
mod misc;
mod native;
mod wavm;

fn expensive_add(op: &Operator) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
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

fn uniform_cost_config() -> StylusConfig {
    let mut config = StylusConfig::default();
    config.debug.count_ops = true;
    config.debug.debug_funcs = true;
    config.start_gas = 1_000_000;
    config.pricing.wasm_gas_price = 100_00;
    config.pricing.hostio_cost = 100;
    config.costs = |_| 1;
    config
}

fn new_test_instance(path: &str, config: StylusConfig) -> Result<NativeInstance> {
    let mut store = config.store();
    let imports = imports! {
        "test" => {
            "noop" => Function::new_typed(&mut store, || {}),
        },
    };
    new_test_instance_from_store(path, store, imports)
}

fn new_test_instance_from_store(
    path: &str,
    mut store: Store,
    imports: Imports,
) -> Result<NativeInstance> {
    let wat = std::fs::read(path)?;
    let module = Module::new(&store, wat)?;
    let instance = Instance::new(&mut store, &module, &imports)?;
    Ok(NativeInstance::new_sans_env(instance, store))
}

pub fn new_test_machine(path: &str, config: StylusConfig) -> Result<Machine> {
    let wat = std::fs::read(path)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let mut bin = prover::binary::parse(&wasm, Path::new("user"))?;
    let stylus_data = bin.instrument(&config)?;

    let wat = std::fs::read("tests/test.wat")?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let lib = prover::binary::parse(&wasm, Path::new("test"))?;

    Machine::from_binaries(
        &[lib],
        bin,
        false,
        false,
        true,
        GlobalState::default(),
        HashMap::default(),
        Arc::new(|_, _| panic!("tried to read preimage")),
        Some(stylus_data),
    )
}

pub fn check_instrumentation(mut native: NativeInstance, mut machine: Machine) -> Result<()> {
    assert_eq!(native.gas_left(), machine.gas_left());
    assert_eq!(native.stack_left(), machine.stack_left());

    let native_counts = native.operator_counts()?;
    let machine_counts = machine.operator_counts()?;
    assert_eq!(native_counts.get(&Operator::Unreachable.into()), None);
    assert_eq!(native_counts, machine_counts);
    Ok(())
}
