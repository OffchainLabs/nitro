use crate::{
    binary::{BlockType, HirInstruction},
    utils::Bytes32,
    value::{IntegerValType, ValueType},
};
use digest::Digest;
use sha3::Keccak256;
use std::convert::TryFrom;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum IRelOpType {
    Eq,
    Ne,
    Lt,
    Gt,
    Le,
    Ge,
}

fn irelop_type(t: IRelOpType, signed: bool) -> u16 {
    match (t, signed) {
        (IRelOpType::Eq, _) => 0,
        (IRelOpType::Ne, _) => 1,
        (IRelOpType::Lt, true) => 2,
        (IRelOpType::Lt, false) => 3,
        (IRelOpType::Gt, true) => 4,
        (IRelOpType::Gt, false) => 5,
        (IRelOpType::Le, true) => 6,
        (IRelOpType::Le, false) => 7,
        (IRelOpType::Ge, true) => 8,
        (IRelOpType::Ge, false) => 9,
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum IUnOpType {
    Clz = 0,
    Ctz,
    Popcnt,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum IBinOpType {
    Add = 0,
    Sub,
    Mul,
    DivS,
    DivU,
    RemS,
    RemU,
    And,
    Or,
    Xor,
    Shl,
    ShrS,
    ShrU,
    Rotl,
    Rotr,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
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
    Select,

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
    I64Eqz,
    IRelOp(IntegerValType, IRelOpType, bool),

    I32WrapI64,
    I64ExtendI32(bool),

    FuncRefConst,

    IUnOp(IntegerValType, IUnOpType),
    IBinOp(IntegerValType, IBinOpType),
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
    /// Duplicate the top value on the stack
    Dup,
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
            Opcode::Select => 0x1B,
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
            Opcode::I64Eqz => 0x50,
            Opcode::IRelOp(w, op, signed) => match w {
                IntegerValType::I32 => 0x46 + irelop_type(op, signed),
                IntegerValType::I64 => 0x51 + irelop_type(op, signed),
            },
            Opcode::IUnOp(w, op) => match w {
                IntegerValType::I32 => 0x67 + (op as u16),
                IntegerValType::I64 => 0x79 + (op as u16),
            },
            Opcode::IBinOp(w, op) => match w {
                IntegerValType::I32 => 0x6a + (op as u16),
                IntegerValType::I64 => 0x7c + (op as u16),
            },
            Opcode::I32WrapI64 => 0xA7,
            Opcode::I64ExtendI32(signed) => match signed {
                true => 0xac,
                false => 0xad,
            },
            Opcode::FuncRefConst => 0xD2,
            // Internal instructions:
            Opcode::EndBlock => 0x8000,
            Opcode::EndBlockIf => 0x8001,
            Opcode::InitFrame => 0x8002,
            Opcode::ArbitraryJumpIf => 0x8003,
            Opcode::PushStackBoundary => 0x8004,
            Opcode::MoveFromStackToInternal => 0x8005,
            Opcode::MoveFromInternalToStack => 0x8006,
            Opcode::IsStackBoundary => 0x8007,
            Opcode::Dup => 0x8008,
        }
    }
}

#[derive(Clone, Copy, Debug)]
pub struct FunctionCodegenState {
    return_values: usize,
    block_depth: usize,
}

impl FunctionCodegenState {
    pub fn new(return_values: usize) -> Self {
        FunctionCodegenState {
            return_values,
            block_depth: 0,
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

    pub fn extend_from_hir(
        ops: &mut Vec<Instruction>,
        mut state: FunctionCodegenState,
        inst: HirInstruction,
    ) {
        match inst {
            HirInstruction::Block(_, insts) => {
                let block_idx = ops.len();
                ops.push(Instruction::simple(Opcode::Block));
                state.block_depth += 1;
                for inst in insts {
                    Self::extend_from_hir(ops, state, inst);
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
                state.block_depth += 1;
                for inst in insts {
                    Self::extend_from_hir(ops, state, inst);
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
                state.block_depth += 1;
                ops.push(Instruction::simple(Opcode::I32Eqz));
                let jump_idx = ops.len();
                ops.push(Instruction::simple(Opcode::ArbitraryJumpIf));

                for inst in if_insts {
                    Self::extend_from_hir(ops, state, inst);
                }
                ops.push(Instruction::simple(Opcode::Branch));

                ops[jump_idx].argument_data = ops.len() as u64;
                for inst in else_insts {
                    Self::extend_from_hir(ops, state, inst);
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
            HirInstruction::BranchTable(options, default) => {
                // Build an equivalent HirInstruction sequence without BranchTable
                let mut equiv = Vec::new();
                for (i, option) in options.iter().enumerate() {
                    let i = match u32::try_from(i) {
                        Ok(x) => x,
                        _ => break,
                    };
                    // Evaluate this branch
                    equiv.push(HirInstruction::Simple(Opcode::Dup));
                    equiv.push(HirInstruction::I32Const(i as i32));
                    equiv.push(HirInstruction::Simple(Opcode::IBinOp(
                        IntegerValType::I32,
                        IBinOpType::Sub,
                    )));
                    // Jump if the subtraction resulted in 0, i.e. it matched the index
                    equiv.push(HirInstruction::Simple(Opcode::I32Eqz));
                    equiv.push(HirInstruction::BranchIf(*option));
                }
                // Nothing matched. Drop the index and jump to the default.
                equiv.push(HirInstruction::Simple(Opcode::Drop));
                equiv.push(HirInstruction::BranchIf(default));
                for inst in equiv {
                    Self::extend_from_hir(ops, state, inst);
                }
            }
            HirInstruction::LocalTee(x) => {
                // Translate into a dup then local.set
                Self::extend_from_hir(ops, state, HirInstruction::Simple(Opcode::Dup));
                Self::extend_from_hir(ops, state, HirInstruction::WithIdx(Opcode::LocalSet, x));
            }
            HirInstruction::WithIdx(op, x) => {
                assert!(
                    matches!(
                        op,
                        Opcode::LocalGet
                            | Opcode::LocalSet
                            | Opcode::GlobalGet
                            | Opcode::GlobalSet
                            | Opcode::Call
                    ),
                    "WithIdx HirInstruction has bad WithIdx opcode",
                );
                ops.push(Instruction {
                    opcode: op,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
            HirInstruction::CallIndirect(_, _) => {
                // TODO
                ops.push(Instruction::simple(Opcode::Unreachable));
            }
            HirInstruction::LoadOrStore(op, mem_arg) => ops.push(Instruction {
                opcode: op,
                argument_data: mem_arg.offset.into(), // we ignore the alignment
                proving_argument_data: None,
            }),
            HirInstruction::I32Const(x) => ops.push(Instruction {
                opcode: Opcode::I32Const,
                argument_data: x as u32 as u64,
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
            HirInstruction::FuncRefConst(x) => ops.push(Instruction {
                opcode: Opcode::FuncRefConst,
                argument_data: x.into(),
                proving_argument_data: None,
            }),
            HirInstruction::Simple(Opcode::Return) => {
                // Hold the return values on the internal stack while we drop extraneous stack values
                ops.extend(
                    std::iter::repeat(Instruction::simple(Opcode::MoveFromStackToInternal))
                        .take(state.return_values),
                );
                // Keep dropping values until we drop the stack boundary, then exit the loop
                Self::extend_from_hir(
                    ops,
                    state,
                    HirInstruction::Loop(
                        BlockType::Empty,
                        vec![
                            HirInstruction::Simple(Opcode::IsStackBoundary),
                            HirInstruction::Simple(Opcode::I32Eqz),
                            HirInstruction::Simple(Opcode::BranchIf),
                        ],
                    ),
                );
                for _ in 0..state.block_depth {
                    ops.push(Instruction::simple(Opcode::EndBlock));
                }
                // Move the return values back from the internal stack to the value stack
                ops.extend(
                    std::iter::repeat(Instruction::simple(Opcode::MoveFromInternalToStack))
                        .take(state.return_values),
                );
                ops.push(Instruction::simple(Opcode::Return));
            }
            HirInstruction::Simple(op) => ops.push(Instruction::simple(op)),
        }
    }
}
