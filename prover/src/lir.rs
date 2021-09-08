use crate::{binary::HirInstruction, utils::Bytes32};
use digest::Digest;
use sha3::Keccak256;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u16)]
pub enum Opcode {
    Unreachable,
    Nop,
    Block,
    // Loop and If are wrapped into Block

    Branch = 0x0C,
    BranchIf,

    LocalGet = 0x20,
    LocalSet,
    GlobalGet = 0x23,
    GlobalSet,

    I32Const = 0x41,
    I64Const,
    F32Const,
    F64Const,

    I32Eqz,

    I32Add = 0x6A,

    I64Add = 0x7C,

    Drop = 0x1A,

    // Custom opcodes:
    /// Custom opcode not in wasm.
    /// Branch is partially split up into these.
    EndBlock = 0x8000,
    /// Custom opcode not in wasm.
    /// Like "EndBlock" but conditional.
    /// Keeps its condition on the stack.
    EndBlockIf,
    /// Custom opcode not in wasm.
    InitFrame,
    /// Conditional jump to an arbitrary point in code.
    ArbitraryJumpIf,
}

impl Opcode {
    pub fn repr(self) -> u16 {
        self as u16
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

    pub fn extend_from_hir(ops: &mut Vec<Instruction>, inst: HirInstruction) {
        match inst {
            HirInstruction::Simple(op) => ops.push(Instruction::simple(op)),
            HirInstruction::Block(_, insts) => {
                let block_idx = ops.len();
                ops.push(Instruction::simple(Opcode::Block));
                for inst in insts {
                    Self::extend_from_hir(ops, inst);
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
                    Self::extend_from_hir(ops, inst);
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
                    Self::extend_from_hir(ops, inst);
                }
                ops.push(Instruction::simple(Opcode::Branch));

                ops[jump_idx].argument_data = ops.len() as u64;
                for inst in else_insts {
                    Self::extend_from_hir(ops, inst);
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
            HirInstruction::LocalGet(x) => {
                ops.push(Instruction {
                    opcode: Opcode::LocalGet,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
            HirInstruction::LocalSet(x) => {
                ops.push(Instruction {
                    opcode: Opcode::LocalSet,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
            HirInstruction::GlobalGet(x) => {
                ops.push(Instruction {
                    opcode: Opcode::GlobalGet,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
            HirInstruction::GlobalSet(x) => {
                ops.push(Instruction {
                    opcode: Opcode::GlobalSet,
                    argument_data: x.into(),
                    proving_argument_data: None,
                });
            }
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
        }
    }
}
