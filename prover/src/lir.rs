use crate::{
    binary::{BlockType, HirInstruction},
    utils::Bytes32,
    value::ValueType,
};
use digest::Digest;
use sha3::Keccak256;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u16)]
pub enum Opcode {
    Unreachable,
    Nop,
    Block,
    // Loop and If are wrapped into Block
    Branch,
    BranchIf,

    Return,
    Call,

    Drop,

    LocalGet,
    LocalSet,
    GlobalGet,
    GlobalSet,

    MemoryLoad {
        /// The type we are loading into.
        ty: ValueType,
        /// How many bytes in memory we are loading from.
        bytes: u8,
        /// When bytes matches the type's size, this is irrelevant and should be false.
        signed: bool,
    },
    MemoryStore {
        /// The type we are storing from.
        ty: ValueType,
        /// How many bytes in memory we are storing into.
        bytes: u8,
    },

    I32Const,
    I64Const,
    F32Const,
    F64Const,

    I32Eqz,
    I32Add,
    I32Sub,
    I32Mul,

    I64Add,

    // Custom opcodes:
    /// Custom opcode not in wasm.
    /// Branch is partially split up into these.
    EndBlock,
    /// Custom opcode not in wasm.
    /// Like "EndBlock" but conditional.
    /// Keeps its condition on the stack.
    EndBlockIf,
    /// Custom opcode not in wasm.
    InitFrame,
    /// Conditional jump to an arbitrary point in code.
    ArbitraryJumpIf,
    /// Push a Value::StackBoundary to the stack
    PushStackBoundary,
    /// Pop a value from the value stack and push it to the internal stack
    MoveFromStackToInternal,
    /// Pop a value from the internal stack and push it to the value stack
    MoveFromInternalToStack,
    /// Pop a value from the value stack, then push an I32 1 if it's a stack boundary, I32 0 otherwise.
    IsStackBoundary,
}

impl Opcode {
    pub fn repr(self) -> u16 {
        match self {
            Opcode::Unreachable => 0x00,
            Opcode::Nop => 0x01,
            Opcode::Block => 0x02,
            Opcode::Branch => 0x0C,
            Opcode::BranchIf => 0x0D,
            Opcode::Return => 0x0F,
            Opcode::Call => 0x10,
            Opcode::Drop => 0x1A,
            Opcode::LocalGet => 0x20,
            Opcode::LocalSet => 0x21,
            Opcode::GlobalGet => 0x23,
            Opcode::GlobalSet => 0x24,
            Opcode::MemoryLoad { ty, bytes, signed } => match (ty, bytes, signed) {
                (ValueType::I32, 4, false) => 0x28,
                (ValueType::I64, 8, false) => 0x29,
                (ValueType::F32, 4, false) => 0x2A,
                (ValueType::F64, 8, false) => 0x2B,
                (ValueType::I32, 1, true) => 0x2C,
                (ValueType::I32, 1, false) => 0x2D,
                (ValueType::I32, 2, true) => 0x2E,
                (ValueType::I32, 2, false) => 0x2F,
                (ValueType::I64, 1, true) => 0x30,
                (ValueType::I64, 1, false) => 0x31,
                (ValueType::I64, 2, true) => 0x32,
                (ValueType::I64, 2, false) => 0x33,
                (ValueType::I64, 4, true) => 0x34,
                (ValueType::I64, 4, false) => 0x35,
                _ => panic!(
                    "Unsupported memory load of type {:?} from {} bytes with signed {}",
                    ty, bytes, signed,
                ),
            },
            Opcode::MemoryStore { ty, bytes } => match (ty, bytes) {
                (ValueType::I32, 4) => 0x36,
                (ValueType::I64, 8) => 0x37,
                (ValueType::F32, 4) => 0x38,
                (ValueType::F64, 8) => 0x39,
                (ValueType::I32, 1) => 0x3A,
                (ValueType::I32, 2) => 0x3B,
                (ValueType::I64, 1) => 0x3C,
                (ValueType::I64, 2) => 0x3D,
                (ValueType::I64, 4) => 0x3E,
                _ => panic!(
                    "Unsupported memory store of type {:?} to {} bytes",
                    ty, bytes,
                ),
            },
            Opcode::I32Const => 0x41,
            Opcode::I64Const => 0x42,
            Opcode::F32Const => 0x43,
            Opcode::F64Const => 0x44,
            Opcode::I32Eqz => 0x45,
            Opcode::I32Add => 0x6A,
            Opcode::I64Add => 0x7C,

            // Internal instructions:
            Opcode::EndBlock => 0x8000,
            Opcode::EndBlockIf => 0x8001,
            Opcode::InitFrame => 0x8002,
            Opcode::ArbitraryJumpIf => 0x8003,
            Opcode::PushStackBoundary => 0x8004,
            Opcode::MoveFromStackToInternal => 0x8005,
            Opcode::MoveFromInternalToStack => 0x8006,
            Opcode::IsStackBoundary => 0x8007,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Instruction {
    pub opcode: Opcode,
    pub argument_data: u64,
    pub proving_argument_data: Option<Bytes32>,
}

impl Instruction {
    pub fn simple(opcode: Opcode) -> Instruction {
        Instruction {
            opcode,
            argument_data: 0,
            proving_argument_data: None,
        }
    }

    pub fn get_proving_argument_data(self) -> Bytes32 {
        if let Some(data) = self.proving_argument_data {
            data
        } else {
            assert!(
                self.opcode != Opcode::Block,
                "Block missing proving argument data",
            );
            let mut b = [0u8; 32];
            b[24..].copy_from_slice(&self.argument_data.to_be_bytes());
            Bytes32(b)
        }
    }

    pub fn serialize_for_proof(self) -> [u8; 34] {
        let mut ret = [0u8; 34];
        ret[..2].copy_from_slice(&self.opcode.repr().to_be_bytes());
        ret[2..].copy_from_slice(&*self.get_proving_argument_data());
        ret
    }

    pub fn hash(self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update(b"Instruction:");
        h.update(self.opcode.repr().to_be_bytes());
        h.update(self.get_proving_argument_data());
        h.finalize().into()
    }

    pub fn extend_from_hir(ops: &mut Vec<Instruction>, return_values: usize, inst: HirInstruction) {
        match inst {
            HirInstruction::Block(_, insts) => {
                let block_idx = ops.len();
                ops.push(Instruction::simple(Opcode::Block));
                for inst in insts {
                    Self::extend_from_hir(ops, return_values, inst);
                }
                ops.push(Instruction::simple(Opcode::EndBlock));
                ops[block_idx].argument_data = ops.len() as u64;
            }
            HirInstruction::Loop(_, insts) => {
                ops.push(Instruction {
                    opcode: Opcode::Block,
                    argument_data: ops.len() as u64,
                    proving_argument_data: None,
                });
                for inst in insts {
                    Self::extend_from_hir(ops, return_values, inst);
                }
                ops.push(Instruction::simple(Opcode::EndBlock));
            }
            HirInstruction::IfElse(_, if_insts, else_insts) => {
                // begin block with endpoint end
                //   conditional jump to else
                //   [instructions inside if statement]
                //   branch
                //   else: [instructions inside else statement]
                // end

                let block_idx = ops.len();
                ops.push(Instruction::simple(Opcode::Block));
                ops.push(Instruction::simple(Opcode::I32Eqz));
                let jump_idx = ops.len();
                ops.push(Instruction::simple(Opcode::ArbitraryJumpIf));

                for inst in if_insts {
                    Self::extend_from_hir(ops, return_values, inst);
                }
                ops.push(Instruction::simple(Opcode::Branch));

                ops[jump_idx].argument_data = ops.len() as u64;
                for inst in else_insts {
                    Self::extend_from_hir(ops, return_values, inst);
                }
                ops.push(Instruction::simple(Opcode::EndBlock));
                ops[block_idx].argument_data = ops.len() as u64;
            }
            HirInstruction::Branch(x) => {
                for _ in 0..x {
                    ops.push(Instruction::simple(Opcode::EndBlock));
                }
                ops.push(Instruction::simple(Opcode::Branch));
            }
            HirInstruction::BranchIf(x) => {
                for _ in 0..x {
                    ops.push(Instruction::simple(Opcode::EndBlockIf));
                }
                ops.push(Instruction::simple(Opcode::BranchIf));
            }
            HirInstruction::WithIdx(op, x) => {
                ops.push(Instruction {
                    opcode: op,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
            HirInstruction::LoadOrStore(op, mem_arg) => ops.push(Instruction {
                opcode: op,
                argument_data: mem_arg.offset.into(), // we ignore the alignment
                proving_argument_data: None,
            }),
            HirInstruction::I32Const(x) => ops.push(Instruction {
                opcode: Opcode::I32Const,
                argument_data: x as u64,
                proving_argument_data: None,
            }),
            HirInstruction::I64Const(x) => ops.push(Instruction {
                opcode: Opcode::I64Const,
                argument_data: x as u64,
                proving_argument_data: None,
            }),
            HirInstruction::F32Const(x) => ops.push(Instruction {
                opcode: Opcode::F32Const,
                argument_data: x.to_bits().into(),
                proving_argument_data: None,
            }),
            HirInstruction::F64Const(x) => ops.push(Instruction {
                opcode: Opcode::F64Const,
                argument_data: x.to_bits(),
                proving_argument_data: None,
            }),
            HirInstruction::Simple(Opcode::Return) => {
                // Hold the return values on the internal stack while we drop extraneous stack values
                ops.extend(
                    std::iter::repeat(Instruction::simple(Opcode::MoveFromStackToInternal))
                        .take(return_values),
                );
                // Keep dropping values until we drop the stack boundary, then exit the loop
                Self::extend_from_hir(
                    ops,
                    return_values,
                    HirInstruction::Loop(
                        BlockType::Empty,
                        vec![
                            HirInstruction::Simple(Opcode::IsStackBoundary),
                            HirInstruction::Simple(Opcode::I32Eqz),
                            HirInstruction::Simple(Opcode::BranchIf),
                        ],
                    ),
                );
                // Move the return values back from the internal stack to the value stack
                ops.extend(
                    std::iter::repeat(Instruction::simple(Opcode::MoveFromInternalToStack))
                        .take(return_values),
                );
                ops.push(Instruction::simple(Opcode::Return));
            }
            HirInstruction::Simple(op) => ops.push(Instruction::simple(op)),
        }
    }
}
