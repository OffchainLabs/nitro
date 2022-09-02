// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{Escape, WasmEnvArc};

use wasmer::{imports, Function, Instance, Module, RuntimeError, Store, Value};

use std::time::Instant;

mod arbcompress;
mod gostack;
mod runtime;
mod syscall;
mod test;
mod wavmio;

fn main() {
    let wasm = std::fs::read("../../target/machines/latest/replay.wasm").unwrap();
    //let wasm = std::fs::read("./programs/time/time.wasm").unwrap();
    let env = WasmEnvArc::default();

    let store = Store::default();
    let module = match Module::new(&store, &wasm) {
        Ok(module) => module,
        Err(err) => panic!("{}", err),
    };

    macro_rules! native {
        ($func:expr) => {
            Function::new_native(&store, $func)
        };
    }
    macro_rules! func {
        ($func:expr) => {
            Function::new_native_with_env(&store, env.clone(), $func)
        };
    }

    let imports = imports! {
        "go" => {
            "debug" => native!(runtime::go_debug),

            "runtime.resetMemoryDataView" => native!(runtime::reset_memory_data_view),
            "runtime.wasmExit" => func!(runtime::wasm_exit),
            "runtime.wasmWrite" => func!(runtime::wasm_write),
            "runtime.nanotime1" => func!(runtime::nanotime1),
            "runtime.walltime" => func!(runtime::walltime),
            "runtime.walltime1" => func!(runtime::walltime1),
            "runtime.scheduleTimeoutEvent" => func!(runtime::schedule_timeout_event),
            "runtime.clearTimeoutEvent" => func!(runtime::clear_timeout_event),
            "runtime.getRandomData" => func!(runtime::get_random_data),

            "syscall/js.finalizeRef" => func!(syscall::js_finalize_ref),
            "syscall/js.stringVal" => func!(syscall::js_string_val),
            "syscall/js.valueGet" => func!(syscall::js_value_get),
            "syscall/js.valueSet" => func!(syscall::js_value_set),
            "syscall/js.valueDelete" => func!(syscall::js_value_delete),
            "syscall/js.valueIndex" => func!(syscall::js_value_index),
            "syscall/js.valueSetIndex" => func!(syscall::js_value_set_index),
            "syscall/js.valueCall" => func!(syscall::js_value_call),
            "syscall/js.valueInvoke" => func!(syscall::js_value_invoke),
            "syscall/js.valueNew" => func!(syscall::js_value_new),
            "syscall/js.valueLength" => func!(syscall::js_value_length),
            "syscall/js.valuePrepareString" => func!(syscall::js_value_prepare_string),
            "syscall/js.valueLoadString" => func!(syscall::js_value_load_string),
            "syscall/js.valueInstanceOf" => func!(syscall::js_value_instance_of),
            "syscall/js.copyBytesToGo" => func!(syscall::js_copy_bytes_to_go),
            "syscall/js.copyBytesToJS" => func!(syscall::js_copy_bytes_to_js),

            "github.com/offchainlabs/nitro/wavmio.getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32),
            "github.com/offchainlabs/nitro/wavmio.setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32),
            "github.com/offchainlabs/nitro/wavmio.getGlobalStateU64" => func!(wavmio::get_global_state_u64),
            "github.com/offchainlabs/nitro/wavmio.setGlobalStateU64" => func!(wavmio::set_global_state_u64),
            "github.com/offchainlabs/nitro/wavmio.readInboxMessage" => func!(wavmio::read_inbox_message),
            "github.com/offchainlabs/nitro/wavmio.readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
            "github.com/offchainlabs/nitro/wavmio.resolvePreImage" => func!(wavmio::resolve_preimage),

            "github.com/offchainlabs/nitro/arbcompress.brotliCompress" => func!(arbcompress::brotli_compress),
            "github.com/offchainlabs/nitro/arbcompress.brotliDecompress" => func!(arbcompress::brotli_decompress),
        },
    };
    let instance = match Instance::new(&module, &imports) {
        Ok(instance) => instance,
        Err(err) => panic!("Failed to create instance: {}", err),
    };

    let memory = match instance.exports.get_memory("mem") {
        Ok(memory) => memory.clone(),
        Err(err) => panic!("Failed to get memory: {}", err),
    };

    env.lock().memory = Some(memory);

    let now = Instant::now();
    let main = instance.exports.get_function("run").unwrap();
    let resume = instance.exports.get_function("resume").unwrap();

    let mut escape;

    fn check_outcome(outcome: Result<Box<[Value]>, RuntimeError>) -> Option<Escape> {
        let outcome = match outcome {
            Ok(outcome) => {
                println!("Go returned values {:?}", outcome);
                return None;
            }
            Err(outcome) => outcome,
        };
        Some(match outcome.downcast() {
            Ok(escape) => escape,
            Err(outcome) => Escape::Failure(format!("unknown runtime error: {}", outcome)),
        })
    }

    let outcome = main.call(&[Value::I32(0), Value::I32(0)]);
    escape = check_outcome(outcome);

    if escape.is_none() {
        while let Some(event) = env.lock().js_future_events.pop_front() {
            if let Some(issue) = &env.lock().js_pending_event {
                println!("Go runtime overwriting pending event {:?}", issue);
            }
            env.lock().js_pending_event = Some(event);
            escape = check_outcome(resume.call(&[]));
            if escape.is_some() {
                break;
            }
        }
    }

    let elapsed = now.elapsed().as_millis();
    match escape {
        Some(Escape::Exit(0)) => println!("Completed in {elapsed}ms"),
        Some(Escape::Exit(x)) => println!("Failed in {elapsed}ms with exit code {x}"),
        Some(Escape::Failure(err)) => println!("Jit failed with {err} in {elapsed}ms"),
        Some(Escape::HostIO(err)) => println!("Hostio failed with {err} in {elapsed}ms"),
        _ => println!("Execution ended prematurely"),
    }
}
