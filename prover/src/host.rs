use crate::{
    binary::{Code, HirInstruction},
    machine::Function,
    value::{FunctionType, ValueType},
    wavm::Opcode,
};

pub fn get_host_impl(module: &str, name: &str) -> Function {
    let mut insts = Vec::new();
    let ty;
    match (module, name) {
        ("env", "wavm_caller_module_memory_load8") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 0));
        }
        ("env", "wavm_caller_module_memory_load32") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 1));
        }
        ("env", "wavm_caller_module_memory_store8") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 2));
        }
        ("env", "wavm_caller_module_memory_store32") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 3));
        }
        _ => panic!("Unsupported import of {:?} {:?}", module, name),
    }
    let code = Code {
        locals: Vec::new(),
        expr: insts,
    };
    Function::new(code, ty, &[])
}
