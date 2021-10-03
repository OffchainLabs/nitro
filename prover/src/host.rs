use crate::{
    binary::{BlockType, Code, HirInstruction},
    machine::Function,
    value::{FunctionType, ValueType},
    wavm::{FloatingPointImpls, Opcode},
};

pub fn get_host_impl(module: &str, name: &str, btype: BlockType) -> Function {
    let mut insts = Vec::new();
    let ty;
    match (module, name) {
        ("env", "wavm_caller_load8") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 0));
        }
        ("env", "wavm_caller_load32") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 1));
        }
        ("env", "wavm_caller_store8") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 2));
        }
        ("env", "wavm_caller_store32") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 3));
        }

        ("env", "wavm_get_last_block_hash") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::Simple(Opcode::GetLastBlockHash));
        }
        ("env", "wavm_set_last_block_hash") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::Simple(Opcode::SetLastBlockHash));
        }
        ("env", "wavm_advance_inbox_position") => {
            ty = FunctionType {
                inputs: vec![],
                outputs: vec![],
            };
            insts.push(HirInstruction::Simple(Opcode::AdvanceInboxPosition));
        }
        ("env", "wavm_read_pre_image") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::ReadPreImage));
        }
        ("env", "wavm_read_inbox_message") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::Simple(Opcode::ReadInboxMessage));
        }
        ("env", "wavm_get_position_within_message") => {
            ty = FunctionType {
                inputs: vec![],
                outputs: vec![ValueType::I64],
            };
            insts.push(HirInstruction::Simple(Opcode::GetPositionWithinMessage));
        }
        ("env", "wavm_set_position_within_message") => {
            ty = FunctionType {
                inputs: vec![ValueType::I64],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::Simple(Opcode::SetPositionWithinMessage));
        }
        _ => panic!("Unsupported import of {:?} {:?}", module, name),
    }
    let code = Code {
        locals: Vec::new(),
        expr: insts,
    };
    Function::new(code, ty, btype, &[], &FloatingPointImpls::default())
}
