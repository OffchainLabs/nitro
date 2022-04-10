// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    binary::{BlockType, Code, HirInstruction},
    machine::{Function, InboxIdentifier},
    value::{ArbValueType, FunctionType},
    wavm::{FloatingPointImpls, Opcode},
};

pub fn get_host_impl(module: &str, name: &str, btype: BlockType) -> eyre::Result<Function> {
    let mut insts = Vec::new();
    let ty;
    match (module, name) {
        ("env", "wavm_caller_load8") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32],
                outputs: vec![ArbValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 0));
        }
        ("env", "wavm_caller_load32") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32],
                outputs: vec![ArbValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 1));
        }
        ("env", "wavm_caller_store8") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 2));
        }
        ("env", "wavm_caller_store32") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 3));
        }
        ("env", "wavm_get_globalstate_bytes32") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::GetGlobalStateBytes32));
        }
        ("env", "wavm_set_globalstate_bytes32") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::SetGlobalStateBytes32));
        }
        ("env", "wavm_get_globalstate_u64") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32],
                outputs: vec![ArbValueType::I64],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::Simple(Opcode::GetGlobalStateU64));
        }
        ("env", "wavm_set_globalstate_u64") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32, ArbValueType::I64],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::SetGlobalStateU64));
        }
        ("env", "wavm_read_pre_image") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I32; 2],
                outputs: vec![ArbValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::ReadPreImage));
        }
        ("env", "wavm_read_inbox_message") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I64, ArbValueType::I32, ArbValueType::I32],
                outputs: vec![ArbValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 2));
            insts.push(HirInstruction::WithIdx(
                Opcode::ReadInboxMessage,
                InboxIdentifier::Sequencer as u32,
            ));
        }
        ("env", "wavm_read_delayed_inbox_message") => {
            ty = FunctionType {
                inputs: vec![ArbValueType::I64, ArbValueType::I32, ArbValueType::I32],
                outputs: vec![ArbValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 2));
            insts.push(HirInstruction::WithIdx(
                Opcode::ReadInboxMessage,
                InboxIdentifier::Delayed as u32,
            ));
        }
        ("env", "wavm_halt_and_set_finished") => {
            ty = FunctionType::default();
            insts.push(HirInstruction::Simple(Opcode::HaltAndSetFinished));
        }
        _ => eyre::bail!("Unsupported import of {:?} {:?}", module, name),
    }
    let code = Code {
        locals: Vec::new(),
        expr: insts,
    };
    Function::new(code, ty, btype, &[], &FloatingPointImpls::default())
}
