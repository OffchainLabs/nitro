// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::WasmEnvArc;

pub fn get_global_state_bytes32(env: &WasmEnvArc, sp: u32) {}

pub fn set_global_state_bytes32(env: &WasmEnvArc, sp: u32) {}

pub fn get_global_state_u64(env: &WasmEnvArc, sp: u32) {}

pub fn set_global_state_u64(env: &WasmEnvArc, sp: u32) {}

pub fn read_inbox_message(env: &WasmEnvArc, sp: u32) {}

pub fn read_delayed_inbox_message(env: &WasmEnvArc, sp: u32) {}

pub fn resolve_preimage(env: &WasmEnvArc, sp: u32) {}
