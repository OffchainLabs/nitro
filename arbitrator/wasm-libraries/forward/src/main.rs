// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use eyre::Result;
use std::{fs::File, io::Write, path::PathBuf};
use structopt::StructOpt;

/// order matters!
const HOSTIOS: [[&str; 3]; 31] = [
    ["read_args", "i32", ""],
    ["write_result", "i32 i32", ""],
    ["storage_load_bytes32", "i32 i32", ""],
    ["storage_store_bytes32", "i32 i32", ""],
    ["call_contract", "i32 i32 i32 i32 i64 i32", "i32"],
    ["delegate_call_contract", "i32 i32 i32 i64 i32", "i32"],
    ["static_call_contract", "i32 i32 i32 i64 i32", "i32"],
    ["create1", "i32 i32 i32 i32 i32", ""],
    ["create2", "i32 i32 i32 i32 i32 i32", ""],
    ["read_return_data", "i32 i32 i32", "i32"],
    ["return_data_size", "", "i32"],
    ["emit_log", "i32 i32 i32", ""],
    ["account_balance", "i32 i32", ""],
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
    ["msg_reentrant", "", "i32"],
    ["msg_sender", "i32", ""],
    ["msg_value", "i32", ""],
    ["native_keccak256", "i32 i32 i32", ""],
    ["tx_gas_price", "i32", ""],
    ["tx_ink_price", "", "i32"],
    ["tx_origin", "i32", ""],
    ["memory_grow", "i32", ""],
];

#[derive(StructOpt)]
#[structopt(name = "arbitrator-prover")]
struct Opts {
    #[structopt(long)]
    path: PathBuf,
    #[structopt(long)]
    stub: bool,
}

fn main() -> Result<()> {
    let opts = Opts::from_args();
    let file = &mut File::options().create(true).write(true).open(opts.path)?;
    match opts.stub {
        true => forward_stub(file),
        false => forward(file),
    }
}

fn forward(file: &mut File) -> Result<()> {
    macro_rules! wln {
        ($($text:tt)*) => {
            writeln!(file, $($text)*)?;
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

    wln!("{s};; symbols to re-export");
    for [name, ins, outs] in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!(
            r#"{s}(import "user_host" "arbitrator_forward__{name}" (func ${name}{params}{result}))"#
        );
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
            "{s}{s}call ${name}\n\
             {s}{s}call $check\n\
             {s})"
        );
    }

    wln!(")");
    Ok(())
}

fn forward_stub(file: &mut File) -> Result<()> {
    macro_rules! wln {
        ($($text:tt)*) => {
            writeln!(file, $($text)*)?;
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
    Ok(())
}
