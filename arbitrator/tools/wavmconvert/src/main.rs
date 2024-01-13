// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE


use prover::{machine::{Machine, Module}, programs::config::CompileConfig};
use std::{path::Path};
use wasmer::Pages;

fn main() {
    let module = Module::from_user_path(Path::new("../../prover/test-cases/memory.wat")).unwrap();
    module.print_wat();

    /*
    let mut compile_config = CompileConfig::version(0, true);
    compile_config.debug.count_ops = true;
    compile_config.bounds.heap_bound = Pages(128);
    compile_config.pricing.costs = |_, _| 0;

    let machine = Machine::from_user_path(Path::new("../../stylus/tests/memory.wat"), &compile_config).unwrap();
    machine.print_wat();
    */
}
