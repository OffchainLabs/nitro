// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::Result;
use prover::{programs::config::PolyglotConfig, Machine};

fn new_test_machine(path: &str, config: PolyglotConfig) -> Result<Machine> {
    let wat = std::fs::read(path)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    Machine::from_user_wasm(&wasm, &config)
}

/// TODO: actually test for gas usage once metering is added in a future PR
#[test]
fn test_gas() -> Result<()> {
    let mut config = PolyglotConfig::default();
    config.costs = super::expensive_add;

    let mut machine = new_test_machine("tests/add.wat", config)?;

    let value = machine.call_function("user", "add_one", vec![32_u32.into()])?;
    assert_eq!(value, vec![33_u32.into()]);
    Ok(())
}
