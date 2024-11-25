// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::EvmData;
use jit::machine::WasmEnv;
use jit::program::{
    exec_program, get_last_msg, pop_with_wasm_env, send_response_with_wasm_env,
    set_response_with_wasm_env, start_program_with_wasm_env, JitConfig,
};
use prover::programs::{
    config::CompileConfig, config::CompileDebugParams, config::CompileMemoryParams,
    config::CompilePricingParams, config::PricingParams, prelude::StylusConfig,
};
use stylus::native::compile;
use wasmer::Target;
use std::str;

const EVM_API_METHOD_REQ_OFFSET: u32 = 0x10000000;

fn to_result(req_type: u32, req_data: &Vec<u8>) -> (&str, &str) {
    let msg = match str::from_utf8(req_data) {
        Ok(v) => v,
        Err(e) => panic!("Invalid UTF-8 sequence: {}", e),
    };

    match req_type {
        0 => return ("", ""), // userSuccess
        1 => return (msg, "ErrExecutionReverted"), // userRevert
        2 => return (msg, "ErrExecutionReverted"), // userFailure
        3 => return ("", "ErrOutOfGas"), // userOutOfInk
        4 => return ("", "ErrDepth"), // userOutOfStack
        _ => return ("", "ErrExecutionReverted") // userUnknown
    }
}

fn main() -> eyre::Result<()> {
    let wasm = match std::fs::read("./programs_to_benchmark/user.wasm") {
        Ok(wasm) => wasm,
        Err(err) => panic!("failed to read: {err}"),
    };

    let compiled_module = compile(&wasm, 0, false, Target::default())?;

    let exec = &mut WasmEnv::default();

    let calldata = Vec::from([0u8; 32]);
    let evm_data = EvmData::default();
    let config = JitConfig {
        stylus: StylusConfig {
            version: 0,
            max_depth: 10000,
            pricing: PricingParams { ink_price: 1 },
        },
        compile: CompileConfig {
            version: 0,
            pricing: CompilePricingParams::default(),
            bounds: CompileMemoryParams::default(),
            debug: CompileDebugParams::default(),
        },
    };

    let module = exec_program(
        exec,
        compiled_module.into(),
        calldata,
        config,
        evm_data,
        160000000,
    )
    .unwrap();
    println!("module: {:?}", module);

    let mut req_id = start_program_with_wasm_env(exec, module).unwrap();
    println!("req_id: {:?}", req_id);

    loop {
        let msg = get_last_msg(exec, req_id).unwrap();
        println!(
            "msg.req_type: {:?}, msg.req_data: {:?}",
            msg.req_type, msg.req_data
        );

        if msg.req_type < EVM_API_METHOD_REQ_OFFSET {
            let _ = pop_with_wasm_env(exec);

            let gas_left = u64::from_be_bytes(msg.req_data[..8].try_into().unwrap());
            let req_data = msg.req_data[8..].to_vec();
            let (msg, err) = to_result(msg.req_type, &req_data);
            println!("gas_left: {:?}, msg: {:?}, err: {:?}", gas_left, msg, err);

            break;
        }

        if msg.req_type != EVM_API_METHOD_REQ_OFFSET {
            panic!("unsupported call");
        }
        set_response_with_wasm_env(exec, req_id, 1, vec![0u8; 32], msg.req_data)?;
        req_id = send_response_with_wasm_env(exec, req_id).unwrap();
    }

    Ok(())
}
