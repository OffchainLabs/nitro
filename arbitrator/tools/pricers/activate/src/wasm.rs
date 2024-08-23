// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::util;
use arbitrary::Unstructured;
use eyre::Result;
use std::borrow::Cow;
use wasm_smith::{Config, InstructionKinds, Module};

#[derive(Debug)]
pub struct WasmConfig {
    max_funcs: usize,
    ops: usize,
    globals: usize,
    invalid: bool,
}

impl Default for WasmConfig {
    fn default() -> Self {
        Self {
            max_funcs: 4095,
            ops: 65536,
            globals: 32765,
            invalid: false,
        }
    }
}

impl Config for WasmConfig {
    fn available_imports(&self) -> Option<Cow<'_, [u8]>> {
        let text = r#"
            (module
                (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
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
        1024
    }
    fn canonicalize_nans(&self) -> bool {
        false
    }
    fn max_elements(&self) -> usize {
        128
    }
    fn max_element_segments(&self) -> usize {
        128
    }
    fn max_components(&self) -> usize {
        self.invalid as usize
    }
    fn max_data_segments(&self) -> usize {
        128
    }
    fn valid_data_inits(&self) -> bool {
        true
    }

    fn max_tags(&self) -> usize {
        self.invalid as usize
    }

    fn force_loop(&self) -> Option<(u32, usize)> {
        //Some((200, 50))
        None
    }

    // upstream bug ignores this for small slices
    fn min_funcs(&self) -> usize {
        2 // memory_grow and start
    }
    fn max_funcs(&self) -> usize {
        4_095
    }
    fn max_types(&self) -> usize {
        4096
    }
    fn max_globals(&self) -> usize {
        32765
    }

    fn max_tables(&self) -> usize {
        1
    }
    fn max_table_elements(&self) -> u32 {
        8192
    }

    fn max_instructions(&self) -> usize {
        65536
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
    }
    fn bulk_memory_enabled(&self) -> bool {
        //true
        false
    }
}

pub fn random_uniform(len: usize) -> Result<Vec<u8>> {
    random(&util::random_vec(len))
}

pub fn random(noise: &[u8]) -> Result<Vec<u8>> {
    let mut input = Unstructured::new(&noise);
    let mut module = Module::new(WasmConfig::default(), &mut input)?;
    module.add_entrypoint();
    Ok(module.to_bytes())
}

#[allow(dead_code)]
pub fn wat(wasm: &[u8]) -> Result<String> {
    let text = wasmprinter::print_bytes(wasm);
    text.map_err(|x| eyre::eyre!(x))
}
