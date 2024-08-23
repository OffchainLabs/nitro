// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::util;
use arbitrary::Unstructured;
use eyre::{eyre, Result, WrapErr};
use std::borrow::Cow;
use wasm_smith::{Config, InstructionKinds, Module};
use wasmer::wasmparser::{Validator, WasmFeatures};

#[derive(Debug)]
pub struct WasmConfig;

impl Config for WasmConfig {
    fn available_imports(&self) -> Option<Cow<'_, [u8]>> {
        let text = r#"
            (module
                (import "vm_hooks" "memory_grow" (func (param i32)))
                (import "pricer" "toggle_timer" (func))
            )
        "#;
        Some(wasmer::wat2wasm(text.as_bytes()).unwrap())
    }
    fn force_imports(&self) -> bool {
        true
    }
    fn min_imports(&self) -> usize {
        2
    }
    fn max_imports(&self) -> usize {
        2
    }
    fn min_exports(&self) -> usize {
        0
    }
    fn max_exports(&self) -> usize {
        0
    }
    fn canonicalize_nans(&self) -> bool {
        false
    }
    fn max_elements(&self) -> usize {
        0
    }
    fn max_element_segments(&self) -> usize {
        0
    }
    fn max_components(&self) -> usize {
        0
    }
    fn max_data_segments(&self) -> usize {
        0
    }
    fn max_tags(&self) -> usize {
        0
    }

    fn force_loop(&self) -> Option<(u32, usize)> {
        Some((200, 50))
    }

    // upstream bug ignores this for small slices
    fn min_funcs(&self) -> usize {
        2 // memory_grow and start
    }
    fn max_funcs(&self) -> usize {
        5
    }
    fn max_types(&self) -> usize {
        5
    }
    fn max_globals(&self) -> usize {
        5
    }

    fn max_instructions(&self) -> usize {
        50
    }
    fn memory_name(&self) -> Option<String> {
        Some("memory".into())
    }
    fn min_memories(&self) -> u32 {
        1
    }
    fn max_memory_pages(&self, _is_64: bool) -> u64 {
        128 // a little over 1 MB
    }
    fn memory64_enabled(&self) -> bool {
        false
    }
    fn memory_offset_choices(&self) -> (u32, u32, u32) {
        (95, 4, 1) // out-of-bounds 5% of the time
    }
    fn multi_value_enabled(&self) -> bool {
        false // research why Singlepass doesn't have this on by default before enabling
    }

    fn allow_start_export(&self) -> bool {
        true
    }
    fn require_start_export(&self) -> bool {
        true
    }

    fn threads_enabled(&self) -> bool {
        false
    }

    fn allowed_instructions(&self) -> InstructionKinds {
        use wasm_smith::InstructionKind::*;
        InstructionKinds::new(&[
            Numeric, Reference, Parametric, Variable, Table, Memory, Control,
        ])
        //InstructionKinds::new(&[Numeric, Variable])
        //InstructionKinds::new(&[Numeric, Variable, Control])
    }
}

pub fn random(len: usize) -> Result<Vec<u8>> {
    let noise = util::random_vec(len);
    let mut input = Unstructured::new(&noise);
    let module = Module::new(WasmConfig, &mut input)?;
    Ok(module.to_bytes())
}

pub fn validate(wasm: &[u8]) -> Result<()> {
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
        floats: false,
        function_references: false,
        gc: false,
        memory_control: false,
    };
    Validator::new_with_features(features).validate_all(wasm)?;
    Ok(())
}

/*
#[allow(dead_code)]
pub fn wat(wasm: &[u8]) -> Result<String> {
    let text = wasmprinter::print_bytes(wasm);
    text.map_err(|x| eyre::eyre!(x))
}
    */
