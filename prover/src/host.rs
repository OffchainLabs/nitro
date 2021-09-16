use crate::{
    binary::{BlockType, Code, HirInstruction, MemoryArg},
    machine::Function,
    value::{FunctionType, IntegerValType, ValueType},
    wavm::{IBinOpType, Opcode},
};

fn write_const(local: u32, val: i32) -> Vec<HirInstruction> {
    vec![
        HirInstruction::WithIdx(Opcode::LocalGet, local),
        HirInstruction::I32Const(val),
        HirInstruction::LoadOrStore(
            Opcode::MemoryStore {
                ty: ValueType::I32,
                bytes: 4,
            },
            MemoryArg {
                alignment: 0,
                offset: 0,
            },
        ),
    ]
}

pub fn get_host_impl(module: &str, name: &str) -> Function {
    let mut insts = Vec::new();
    let mut locals = Vec::new();
    let ty;
    let id;
    match (module, name) {
        ("wasi_snapshot_preview1", "environ_sizes_get") => {
            id = 0;
            ty = FunctionType {
                inputs: vec![ValueType::I32; 2],
                outputs: vec![ValueType::I32],
            };
            insts.extend(write_const(0, 0));
            insts.extend(write_const(1, 0));
            insts.push(HirInstruction::I32Const(0));
        }
        ("wasi_snapshot_preview1", "environ_get") => {
            id = 1;
            ty = FunctionType {
                inputs: vec![ValueType::I32, ValueType::I32],
                outputs: vec![ValueType::I32],
            };
            insts.push(HirInstruction::I32Const(28));
        }
        ("wasi_snapshot_preview1", "proc_exit") | ("env", "exit") => {
            id = 2;
            ty = FunctionType {
                inputs: vec![ValueType::I32],
                outputs: Vec::new(),
            };
            insts.push(HirInstruction::Simple(Opcode::Unreachable));
        }
        ("wasi_snapshot_preview1", "fd_write") => {
            id = 3;
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
        _ => panic!("Unsupported import of {:?} {:?}", module, name),
    }
    let code = Code {
        locals,
        expr: insts,
    };
    Function::new_advanced(code, ty, &[], Some(id))
}
