// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    machine::{Function, InboxIdentifier},
    value::{ArbValueType, FunctionType},
    wavm::{Instruction, Opcode},
};

pub fn get_host_impl(module: &str, name: &str) -> eyre::Result<Function> {
    let mut out = vec![];
    let ty;

    macro_rules! opcode {
        ($opcode:ident) => {
            out.push(Instruction::simple(Opcode::$opcode))
        };
        ($opcode:ident, $value:expr) => {
            out.push(Instruction::with_data(Opcode::$opcode, $value))
        };
    }

    match (module, name) {
        ("env", "wavm_caller_load8") => {
            ty = FunctionType::new(vec![ArbValueType::I32], vec![ArbValueType::I32]);
            opcode!(LocalGet, 0);
            opcode!(CallerModuleInternalCall, 0);
        }
        ("env", "wavm_caller_load32") => {
            ty = FunctionType::new(vec![ArbValueType::I32], vec![ArbValueType::I32]);
            opcode!(LocalGet, 0);
            opcode!(CallerModuleInternalCall, 1);
        }
        ("env", "wavm_caller_store8") => {
            ty = FunctionType::new(vec![ArbValueType::I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(CallerModuleInternalCall, 2);
        }
        ("env", "wavm_caller_store32") => {
            ty = FunctionType::new(vec![ArbValueType::I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(CallerModuleInternalCall, 3);
        }
        ("env", "wavm_get_globalstate_bytes32") => {
            ty = FunctionType::new(vec![ArbValueType::I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(GetGlobalStateBytes32);
        }
        ("env", "wavm_set_globalstate_bytes32") => {
            ty = FunctionType::new(vec![ArbValueType::I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(SetGlobalStateBytes32);
        }
        ("env", "wavm_get_globalstate_u64") => {
            ty = FunctionType::new(vec![ArbValueType::I32], vec![ArbValueType::I64]);
            opcode!(LocalGet, 0);
            opcode!(GetGlobalStateU64);
        }
        ("env", "wavm_set_globalstate_u64") => {
            ty = FunctionType::new(vec![ArbValueType::I32, ArbValueType::I64], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(SetGlobalStateU64);
        }
        ("env", "wavm_read_pre_image") => {
            ty = FunctionType::new(vec![ArbValueType::I32; 2], vec![ArbValueType::I32]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(ReadPreImage);
        }
        ("env", "wavm_read_inbox_message") => {
            ty = FunctionType::new(
                vec![ArbValueType::I64, ArbValueType::I32, ArbValueType::I32],
                vec![ArbValueType::I32],
            );
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(LocalGet, 2);
            opcode!(ReadInboxMessage, InboxIdentifier::Sequencer as u64);
        }
        ("env", "wavm_read_delayed_inbox_message") => {
            ty = FunctionType::new(
                vec![ArbValueType::I64, ArbValueType::I32, ArbValueType::I32],
                vec![ArbValueType::I32],
            );
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(LocalGet, 2);
            opcode!(ReadInboxMessage, InboxIdentifier::Delayed as u64);
        }
        ("env", "wavm_halt_and_set_finished") => {
            ty = FunctionType::default();
            opcode!(HaltAndSetFinished);
        }
        _ => eyre::bail!("Unsupported import of {:?} {:?}", module, name),
    }

    let append = |code: &mut Vec<Instruction>| {
        code.extend(out);
        Ok(())
    };

    Function::new(&[], append, ty, &[])
}
