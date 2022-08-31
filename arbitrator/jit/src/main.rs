// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::WasmEnvArc;

use wasmer::{imports, Function, Instance, Module, Store, Value};

mod gostack;
mod host;
mod test;

fn main() {
    let wasm = std::fs::read("programs/print/print.wasm").unwrap();
    let env = WasmEnvArc::default();

    let store = Store::default();
    let module = match Module::new(&store, &wasm) {
        Ok(module) => module,
        Err(err) => panic!("{}", err),
    };

    macro_rules! native {
        ($func:ident) => {
            Function::new_native(&store, host::$func)
        };
    }
    macro_rules! register {
        ($func:ident) => {
            Function::new_native_with_env(&store, env.clone(), host::$func)
        };
    }

    let imports = imports! {
        "go" => {
            "debug" => native!(go_debug),
            "runtime.resetMemoryDataView" => native!(runtime_reset_memory_data_view),
            "runtime.wasmExit" => register!(runtime_wasm_exit),
            "runtime.wasmWrite" => register!(runtime_wasm_write),
            "runtime.nanotime1" => register!(runtime_nanotime1),
            "runtime.walltime" => register!(runtime_walltime),
            "runtime.scheduleTimeoutEvent" => register!(runtime_schedule_timeout_event),
            "runtime.clearTimeoutEvent" => register!(runtime_clear_timeout_event),
            "runtime.getRandomData" => register!(runtime_get_random_data),
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

    let add_one = instance.exports.get_function("run").unwrap();
    let result = add_one.call(&[Value::I32(0), Value::I32(0)]).unwrap();
    assert_eq!(result[0], Value::I32(0));
}
