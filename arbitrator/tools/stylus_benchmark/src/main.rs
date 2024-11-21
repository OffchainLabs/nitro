// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use jit::machine::WasmEnv;
use jit::program::{create_evm_data_v2, create_stylus_config};
use wasmer::{CompilerConfig, Function, FunctionEnv, Store, Value, Memory, MemoryType };
use wasmer_compiler_cranelift::Cranelift;

fn main() -> eyre::Result<()> {
    let mut compiler = Cranelift::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();
    let mut store = Store::new(compiler);

    let mut env = WasmEnv::default();
    let mem = Memory::new(&mut store, MemoryType::new(10000, Some(60000), false)).unwrap();
    env.memory = Some(mem);

    let func_env = FunctionEnv::new(&mut store, env);

    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(&mut store, &func_env, $func)
        };
    }
    let f_create_stylus_config = func!(create_stylus_config);
    let f_create_evm_data = func!(create_evm_data_v2);

    let config_handler = f_create_stylus_config
        .call(
            &mut store,
            &[
                Value::I32(0),
                Value::I32(10000),
                Value::I32(1),
                Value::I32(0),
            ],
        )
        .unwrap();
    println!("config_handler={:?}", config_handler);

    let block_base_fee = Vec::from([0u8; 32]);
    let block_coinbase = Vec::from([0u8; 32]);
    let contract_address = Vec::from([0u8; 32]);
    let module_hash = Vec::from([0u8; 32]);
    let msg_sender = Vec::from([0u8; 32]);
    let msg_value = Vec::from([0u8; 32]);
    let tx_gas_price = Vec::from([0u8; 32]);
    let tx_origin = Vec::from([0u8; 32]);
    let data_handler = f_create_evm_data
        .call(
            &mut store,
            &[
                Value::I64(0),
                Value::I32(&block_base_fee as *const _ as i32),
                Value::I64(0),
                Value::I32(&block_coinbase as *const _ as i32),
                Value::I64(0),
                Value::I64(0),
                Value::I64(0),
                Value::I32(&contract_address as *const _ as i32),
                Value::I32(&module_hash as *const _ as i32),
                Value::I32(&msg_sender as *const _ as i32),
                Value::I32(&msg_value as *const _ as i32),
                Value::I32(&tx_gas_price as *const _ as i32),
                Value::I32(&tx_origin as *const _ as i32),
                Value::I32(0),
                Value::I32(0),
            ],
        )
        .unwrap();
    println!("data_handler={:?}", data_handler);

    Ok(())
}
