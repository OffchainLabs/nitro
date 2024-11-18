use std::fs;
// use wabt::wat2wasm;
use wasmer::wasmparser::{Validator, WasmFeatures};
// use wasmer::{imports, Imports, Function, Instance, Module, Store};
use wasmer::{imports, Instance, Module, Store};

fn main() -> eyre::Result<()> {
    // let wasm = fs::read(format!("../../../target/machines/latest/user_host.wasm"))?;
    let wasm = fs::read(format!("../../../target/machines/latest/host_io.wasm"))?;

    let features = WasmFeatures {
        mutable_global: true,
        saturating_float_to_int: true,
        sign_extension: true,
        reference_types: false,
        multi_value: false,
        bulk_memory: true, // not all ops supported yet
        simd: false,
        relaxed_simd: false,
        threads: false,
        tail_call: false,
        multi_memory: false,
        exceptions: false,
        memory64: false,
        extended_const: false,
        component_model: false,
        component_model_nested_names: false,
        component_model_values: false,
        floats: true,
        function_references: false,
        gc: false,
        memory_control: false,
    };
    Validator::new_with_features(features).validate_all(&wasm)?;

    let mut store = Store::default();
    let module = Module::new(&store, wasm)?;

    // WHICH IMPORTS TO USE?
    let import_object = imports! {};
    let _instance = Instance::new(&mut store, &module, &import_object)?;

    println!("Hello, world!");
    Ok(())
}

// fn foo(n: i32) -> i32 {
//     n
// }
// fn main() -> eyre::Result<()> {
//     let wasm = fs::read(format!("./target/wasm32-wasi/debug/stylus_caller.wasm"))?;
//
//     let features = WasmFeatures {
//         mutable_global: true,
//         saturating_float_to_int: true,
//         sign_extension: true,
//         reference_types: false,
//         multi_value: false,
//         bulk_memory: true, // not all ops supported yet
//         simd: false,
//         relaxed_simd: false,
//         threads: false,
//         tail_call: false,
//         multi_memory: false,
//         exceptions: false,
//         memory64: false,
//         extended_const: false,
//         component_model: false,
//         component_model_nested_names: false,
//         component_model_values: false,
//         floats: true,
//         function_references: false,
//         gc: false,
//         memory_control: false,
//     };
//     Validator::new_with_features(features).validate_all(&wasm)?;
//
//     let mut store = Store::default();
//     let module = Module::new(&store, wasm)?;
//
//     let host_fn = Function::new_typed(&mut store, foo);
//     let import_object: Imports = imports! {
//         "hostio" => {
//             "wavm_link_module" => host_fn,
//         },
//     };
//
//     let instance = Instance::new(&mut store, &module, &import_object)?;
//
//     let main = instance.exports.get_function("main")?;
//     let result = main.call(&mut store, &[])?;
//
//     println!("Hello, world!, wasm: {:?}", result);
//     Ok(())
// }

// fn main() -> eyre::Result<()> {
//     let wat = fs::read(format!("./programs_to_benchmark/add_one.wat"))?;
//     let wasm = wat2wasm(&wat)?;
//
//     let features = WasmFeatures {
//         mutable_global: true,
//         saturating_float_to_int: true,
//         sign_extension: true,
//         reference_types: false,
//         multi_value: false,
//         bulk_memory: true, // not all ops supported yet
//         simd: false,
//         relaxed_simd: false,
//         threads: false,
//         tail_call: false,
//         multi_memory: false,
//         exceptions: false,
//         memory64: false,
//         extended_const: false,
//         component_model: false,
//         component_model_nested_names: false,
//         component_model_values: false,
//         floats: false,
//         function_references: false,
//         gc: false,
//         memory_control: false,
//     };
//     Validator::new_with_features(features).validate_all(&wasm)?;
//
//     let mut store = Store::default();
//     let module = Module::new(&store, &wat)?;
//     // The module doesn't import anything, so we create an empty import object.
//     let import_object = imports! {};
//     let instance = Instance::new(&mut store, &module, &import_object)?;
//
//     let add_one = instance.exports.get_function("add_one")?;
//     let result = add_one.call(&mut store, &[Value::I32(42)])?;
//     assert_eq!(result[0], Value::I32(43));
//
//     println!("Hello, world!, result: {:?}", result);
//     println!("{:?}", wasm);
//     Ok(())
// }
