// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

/// Extracts the original function names from the wasm module's custom `name`
/// section.
///
/// Returns one entry per function index (functions without a name are
/// `None`), with indices rebased so that the first named function sits at 0.
/// The replay guest consumes this mapping (serialized as JSON) to register
/// profiler symbols for wasmer-compiled code.
pub fn extract_function_names(_wasm: &[u8]) -> anyhow::Result<Vec<Option<String>>> {
    todo!("NIT-5349: parse the wasm name section")
}
