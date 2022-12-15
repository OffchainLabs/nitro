// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::Result;
use prover::programs::{
    config::PolyglotConfig,
    meter::{MachineMeter, MeteredMachine},
};
use wasmer::{imports, wasmparser::Operator, Instance, Module, Store};

fn expensive_add(op: &Operator) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
}

fn new_test_instance(path: &str, config: PolyglotConfig) -> Result<(Instance, Store)> {
    let wat = std::fs::read(path)?;
    let mut store = config.store();

    let module = Module::new(&store, &wat)?;
    let imports = imports! {};
    let instance = Instance::new(&mut store, &module, &imports)?;
    Ok((instance, store))
}

#[test]
fn test_gas() -> Result<()> {
    let mut config = PolyglotConfig::default();
    config.costs = expensive_add;

    let (mut instance, mut store) = new_test_instance("tests/add.wat", config)?;
    let exports = &instance.exports;
    let add_one = exports.get_typed_function::<i32, i32>(&store, "add_one")?;
    let store = &mut store;

    assert_eq!(instance.gas_left(store), MachineMeter::Ready(0));

    macro_rules! exhaust {
        ($gas:expr) => {
            instance.set_gas(store, $gas);
            assert_eq!(instance.gas_left(store), MachineMeter::Ready($gas));
            assert!(add_one.call(store, 32).is_err());
            assert_eq!(instance.gas_left(store), MachineMeter::Exhausted);
        };
    }

    exhaust!(0);
    exhaust!(50);
    exhaust!(99);

    let mut gas_left = 500;
    instance.set_gas(store, gas_left);
    while gas_left > 0 {
        assert_eq!(instance.gas_left(store), MachineMeter::Ready(gas_left));
        assert_eq!(add_one.call(store, 64)?, 65);
        gas_left -= 100;
    }
    assert!(add_one.call(store, 32).is_err());
    assert_eq!(instance.gas_left(store), MachineMeter::Exhausted);
    Ok(())
}
