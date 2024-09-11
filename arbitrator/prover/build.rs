use core::str;
use std::{env, fmt::Write, fs, path::Path};

/// order matters!
pub const HOSTIOS: [[&str; 3]; 42] = [
    ["read_args", "i32", ""],
    ["write_result", "i32 i32", ""],
    ["exit_early", "i32", ""],
    ["storage_load_bytes32", "i32 i32", ""],
    ["storage_cache_bytes32", "i32 i32", ""],
    ["storage_flush_cache", "i32", ""],
    ["transient_load_bytes32", "i32 i32", ""],
    ["transient_store_bytes32", "i32 i32", ""],
    ["call_contract", "i32 i32 i32 i32 i64 i32", "i32"],
    ["delegate_call_contract", "i32 i32 i32 i64 i32", "i32"],
    ["static_call_contract", "i32 i32 i32 i64 i32", "i32"],
    ["create1", "i32 i32 i32 i32 i32", ""],
    ["create2", "i32 i32 i32 i32 i32 i32", ""],
    ["read_return_data", "i32 i32 i32", "i32"],
    ["return_data_size", "", "i32"],
    ["emit_log", "i32 i32 i32", ""],
    ["account_balance", "i32 i32", ""],
    ["account_code", "i32 i32 i32 i32", "i32"],
    ["account_code_size", "i32", "i32"],
    ["account_codehash", "i32 i32", ""],
    ["evm_gas_left", "", "i64"],
    ["evm_ink_left", "", "i64"],
    ["block_basefee", "i32", ""],
    ["chainid", "", "i64"],
    ["block_coinbase", "i32", ""],
    ["block_gas_limit", "", "i64"],
    ["block_number", "", "i64"],
    ["block_timestamp", "", "i64"],
    ["contract_address", "i32", ""],
    ["math_div", "i32 i32", ""],
    ["math_mod", "i32 i32", ""],
    ["math_pow", "i32 i32", ""],
    ["math_add_mod", "i32 i32 i32", ""],
    ["math_mul_mod", "i32 i32 i32", ""],
    ["msg_reentrant", "", "i32"],
    ["msg_sender", "i32", ""],
    ["msg_value", "i32", ""],
    ["native_keccak256", "i32 i32 i32", ""],
    ["tx_gas_price", "i32", ""],
    ["tx_ink_price", "", "i32"],
    ["tx_origin", "i32", ""],
    ["pay_for_memory_grow", "i32", ""],
];

pub fn gen_forwarder(out_path: &Path) {
    let mut wat = String::new();
    macro_rules! wln {
        ($($text:tt)*) => {
            writeln!(wat, $($text)*).unwrap();
        };
    }
    let s = "    ";

    wln!("(module");

    macro_rules! group {
        ($list:expr, $kind:expr) => {
            (!$list.is_empty())
                .then(|| format!(" ({} {})", $kind, $list))
                .unwrap_or_default()
        };
    }

    wln!("{s};; symbols to re-export");
    for [name, ins, outs] in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!(r#"{s}(import "user_host" "{name}" (func $_{name}{params}{result}))"#);
    }
    wln!();

    wln!("{s};; reserved offsets for future user_host imports");
    for i in HOSTIOS.len()..512 {
        wln!("{s}(func $reserved_{i} unreachable)");
    }
    wln!();

    wln!(
        "{s};; allows user_host to request a trap\n\
        {s}(global $trap (mut i32) (i32.const 0))\n\
        {s}(func $check\n\
        {s}{s}global.get $trap                    ;; see if set\n\
        {s}{s}(global.set $trap (i32.const 0))    ;; reset the flag\n\
        {s}{s}(if (then (unreachable)))\n\
        {s})\n\
        {s}(func (export \"forward__set_trap\")\n\
        {s}{s}(global.set $trap (i32.const 1))\n\
        {s})\n"
    );

    wln!("{s};; user linkage");
    for [name, ins, outs] in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!("{s}(func (export \"vm_hooks__{name}\"){params}{result}");

        let gets = (1 + ins.len()) / 4;
        for i in 0..gets {
            wln!("{s}{s}local.get {i}");
        }

        wln!(
            "{s}{s}call $_{name}\n\
             {s}{s}call $check\n\
             {s})"
        );
    }

    wln!(")");

    let wasm = wasmer::wat2wasm(wat.as_bytes()).unwrap();

    fs::write(out_path, wasm.as_ref()).unwrap();
}

pub fn gen_forwarder_stub(out_path: &Path) {
    let mut wat = String::new();

    macro_rules! wln {
        ($($text:tt)*) => {
            writeln!(wat, $($text)*).unwrap();
        };
    }
    let s = "    ";

    wln!(
        ";; Copyright 2022-2023, Offchain Labs, Inc.\n\
         ;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE\n\
         ;; This file is auto-generated.\n\
         \n\
         (module"
    );

    macro_rules! group {
        ($list:expr, $kind:expr) => {
            (!$list.is_empty())
                .then(|| format!(" ({} {})", $kind, $list))
                .unwrap_or_default()
        };
    }

    wln!("{s};; stubs for the symbols we re-export");
    for [name, ins, outs] in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!("{s}(func ${name}{params}{result} unreachable)");
    }
    wln!();

    wln!("{s};; reserved offsets for future user_host imports");
    for i in HOSTIOS.len()..512 {
        wln!("{s}(func $reserved_{i} unreachable)");
    }
    wln!();

    wln!(
        "{s};; allows user_host to request a trap\n\
        {s}(global $trap (mut i32) (i32.const 0))\n\
        {s}(func $check unreachable)\n\
        {s}(func (export \"forward__set_trap\") unreachable)"
    );

    wln!("{s};; user linkage");
    for [name, ins, outs] in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!("{s}(func (export \"vm_hooks__{name}\"){params}{result} unreachable)");
    }

    wln!(")");

    let wasm = wasmer::wat2wasm(wat.as_bytes()).unwrap();

    fs::write(out_path, wasm.as_ref()).unwrap();
}

fn main() {
    let out_dir = env::var("OUT_DIR").unwrap();
    let forwarder_path = Path::new(&out_dir).join("forwarder.wasm");
    let forwarder_stub_path = Path::new(&out_dir).join("forwarder_stub.wasm");
    gen_forwarder(&forwarder_path);
    gen_forwarder_stub(&forwarder_stub_path);
}
