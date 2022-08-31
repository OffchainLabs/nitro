// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::WasmEnvArc;

pub fn js_finalize_ref(env: &WasmEnvArc, sp: u32) {}

pub fn js_string_val(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_get(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_set(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_delete(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_index(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_set_index(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_call(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_invoke(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_new(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_length(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_prepare_string(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_load_string(env: &WasmEnvArc, sp: u32) {}

pub fn js_value_instance_of(env: &WasmEnvArc, sp: u32) {}

pub fn js_copy_bytes_to_go(env: &WasmEnvArc, sp: u32) {}

pub fn js_copy_bytes_to_js(env: &WasmEnvArc, sp: u32) {}
