use crate::{
    binary::{BlockType, Code, HirInstruction, MemoryArg},
    machine::Function,
    value::{FunctionType, IntegerValType, ValueType},
    wavm::{IBinOpType, Opcode},
};

const WASI_BAD_FD: i32 = 8;

pub fn get_host_impl(module: &str, name: &str, is_library: bool) -> Function {
    let mut insts = Vec::new();
    let mut locals = Vec::new();
    let ty;
    match (module, name) {
        ("env", "wavm_caller_module_memory_load8") => {
            assert!(is_library, "Only libraries are allowed to use {}", name);
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 0));
        }
        ("env", "wavm_caller_module_memory_load32") => {
            assert!(is_library, "Only libraries are allowed to use {}", name);
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 1));
        }
        ("env", "wavm_caller_module_memory_store8") => {
            assert!(is_library, "Only libraries are allowed to use {}", name);
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 2));
        }
        ("env", "wavm_caller_module_memory_store32") => {
            assert!(is_library, "Only libraries are allowed to use {}", name);
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![],
            };
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 0));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 1));
            insts.push(HirInstruction::WithIdx(Opcode::CallerModuleInternalCall, 3));
        }
        ("wasi_snapshot_preview1", "fd_write") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 4],
                outputs: vec![ValueType::I32],
            };
            locals.push(ValueType::I32); // stores the total size so far
            insts.push(HirInstruction::Loop(
                BlockType::Empty,
                vec![
                    // Check if we have anything more to process
                    HirInstruction::WithIdx(Opcode::LocalGet, 2),
                    HirInstruction::IfElse(
                        BlockType::Empty,
                        vec![
                            // Subtract 1 from the size
                            HirInstruction::I32Const(1),
                            HirInstruction::WithIdx(Opcode::LocalGet, 2),
                            HirInstruction::Simple(Opcode::IBinOp(
                                IntegerValType::I32,
                                IBinOpType::Sub,
                            )),
                            HirInstruction::LocalTee(2),
                            // Transform the array offset into a byte offset
                            HirInstruction::I32Const(8),
                            HirInstruction::Simple(Opcode::IBinOp(
                                IntegerValType::I32,
                                IBinOpType::Mul,
                            )),
                            // Lookup the corresponding data segment's length
                            HirInstruction::WithIdx(Opcode::LocalGet, 1),
                            HirInstruction::Simple(Opcode::IBinOp(
                                IntegerValType::I32,
                                IBinOpType::Add,
                            )),
                            HirInstruction::LoadOrStore(
                                Opcode::MemoryLoad {
                                    ty: ValueType::I32,
                                    bytes: 4,
                                    signed: false,
                                },
                                MemoryArg {
                                    alignment: 0,
                                    offset: 4,
                                },
                            ),
                            // Add the length to the total size
                            HirInstruction::WithIdx(Opcode::LocalGet, 4),
                            HirInstruction::Simple(Opcode::IBinOp(
                                IntegerValType::I32,
                                IBinOpType::Add,
                            )),
                            HirInstruction::WithIdx(Opcode::LocalSet, 4),
                            // Continue the loop
                            HirInstruction::Branch(1),
                        ],
                        vec![],
                    ),
                ],
            ));
            // Set the return pointer to the total size
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 3));
            insts.push(HirInstruction::WithIdx(Opcode::LocalGet, 4));
            insts.push(HirInstruction::LoadOrStore(
                Opcode::MemoryStore {
                    ty: ValueType::I32,
                    bytes: 4,
                },
                MemoryArg {
                    alignment: 0,
                    offset: 0,
                },
            ));
            // Return 0, indicating no error
            insts.push(HirInstruction::I32Const(0));
        }
        ("wasi_snapshot_preview1", "fd_prestat_get") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::I32Const(WASI_BAD_FD));
        }
        ("wasi_snapshot_preview1", "fd_prestat_dir_name") => {
            ty = FunctionType {
                inputs: vec![ValueType::I32; 3],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::I32Const(WASI_BAD_FD));
        }
        _ => panic!("Unsupported import of {:?} {:?}", module, name),
    }
    let code = Code {
        locals,
        expr: insts,
    };
    Function::new(code, ty, &[])
}
