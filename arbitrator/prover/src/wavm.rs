// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    binary::{BlockType, FloatBinOp, FloatInstruction, FloatRelOp, FloatType, HirInstruction},
    utils::Bytes32,
    value::{ArbValueType, IntegerValType},
};
use digest::Digest;
use eyre::{bail, ensure, Result};
use fnv::FnvHashMap as HashMap;
use sha3::Keccak256;
use std::convert::TryFrom;
use wasmparser::Operator;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
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

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
#[repr(u8)]
pub enum IUnOpType {
    Clz = 0,
    Ctz,
    Popcnt,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
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

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Opcode {
    Unreachable,
    Nop,
    Block,
    // Loop and If are wrapped into Block
    Branch,
    BranchIf,

    Return,
    Call,
    CallIndirect,

    Drop,
    Select,

    LocalGet,
    LocalSet,
    GlobalGet,
    GlobalSet,

    MemoryLoad {
        /// The type we are loading into.
        ty: ArbValueType,
        /// How many bytes in memory we are loading from.
        bytes: u8,
        /// When bytes matches the type's size, this is irrelevant and should be false.
        signed: bool,
    },
    MemoryStore {
        /// The type we are storing from.
        ty: ArbValueType,
        /// How many bytes in memory we are storing into.
        bytes: u8,
    },

    MemorySize,
    MemoryGrow,

    I32Const,
    I64Const,
    F32Const,
    F64Const,

    I32Eqz,
    I64Eqz,
    IRelOp(IntegerValType, IRelOpType, bool),

    I32WrapI64,
    I64ExtendI32(bool),

    /// Parameterized by destination type, then source type
    Reinterpret(ArbValueType, ArbValueType),

    /// Parameterized by the number of source bits
    I32ExtendS(u8),
    /// Parameterized by the number of source bits
    I64ExtendS(u8),

    IUnOp(IntegerValType, IUnOpType),
    IBinOp(IntegerValType, IBinOpType),

    // Custom opcodes not in WASM. Documented more in "Custom opcodes.md".
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
    /// Call a function in a different module
    CrossModuleCall,
    /// Call a caller module's internal method with a given function offset
    CallerModuleInternalCall,
    /// Gets bytes32 from global state
    GetGlobalStateBytes32,
    /// Sets bytes32 in global state
    SetGlobalStateBytes32,
    /// Gets u64 from global state
    GetGlobalStateU64,
    /// Sets u64 in global state
    SetGlobalStateU64,
    /// Reads the preimage of a hash in-place into the pointer on the stack at an offset
    ReadPreImage,
    /// Reads the current inbox message into the pointer on the stack at an offset
    ReadInboxMessage,
    /// Stop exexcuting the machine and move to the finished status
    HaltAndSetFinished,
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
            Opcode::CallIndirect => 0x11,
            Opcode::Drop => 0x1A,
            Opcode::Select => 0x1B,
            Opcode::LocalGet => 0x20,
            Opcode::LocalSet => 0x21,
            Opcode::GlobalGet => 0x23,
            Opcode::GlobalSet => 0x24,
            Opcode::MemoryLoad { ty, bytes, signed } => match (ty, bytes, signed) {
                (ArbValueType::I32, 4, false) => 0x28,
                (ArbValueType::I64, 8, false) => 0x29,
                (ArbValueType::F32, 4, false) => 0x2A,
                (ArbValueType::F64, 8, false) => 0x2B,
                (ArbValueType::I32, 1, true) => 0x2C,
                (ArbValueType::I32, 1, false) => 0x2D,
                (ArbValueType::I32, 2, true) => 0x2E,
                (ArbValueType::I32, 2, false) => 0x2F,
                (ArbValueType::I64, 1, true) => 0x30,
                (ArbValueType::I64, 1, false) => 0x31,
                (ArbValueType::I64, 2, true) => 0x32,
                (ArbValueType::I64, 2, false) => 0x33,
                (ArbValueType::I64, 4, true) => 0x34,
                (ArbValueType::I64, 4, false) => 0x35,
                _ => panic!(
                    "Unsupported memory load of type {:?} from {} bytes with signed {}",
                    ty, bytes, signed,
                ),
            },
            Opcode::MemoryStore { ty, bytes } => match (ty, bytes) {
                (ArbValueType::I32, 4) => 0x36,
                (ArbValueType::I64, 8) => 0x37,
                (ArbValueType::F32, 4) => 0x38,
                (ArbValueType::F64, 8) => 0x39,
                (ArbValueType::I32, 1) => 0x3A,
                (ArbValueType::I32, 2) => 0x3B,
                (ArbValueType::I64, 1) => 0x3C,
                (ArbValueType::I64, 2) => 0x3D,
                (ArbValueType::I64, 4) => 0x3E,
                _ => panic!(
                    "Unsupported memory store of type {:?} to {} bytes",
                    ty, bytes,
                ),
            },
            Opcode::MemorySize => 0x3F,
            Opcode::MemoryGrow => 0x40,
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
            Opcode::Reinterpret(dest, source) => match (dest, source) {
                (ArbValueType::I32, ArbValueType::F32) => 0xBC,
                (ArbValueType::I64, ArbValueType::F64) => 0xBD,
                (ArbValueType::F32, ArbValueType::I32) => 0xBE,
                (ArbValueType::F64, ArbValueType::I64) => 0xBF,
                _ => panic!("Unsupported reinterpret to {:?} from {:?}", dest, source),
            },
            Opcode::I32ExtendS(x) => match x {
                8 => 0xC0,
                16 => 0xC1,
                _ => panic!("Unsupported {:?}", self),
            },
            Opcode::I64ExtendS(x) => match x {
                8 => 0xC2,
                16 => 0xC3,
                32 => 0xC4,
                _ => panic!("Unsupported {:?}", self),
            },
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
            Opcode::CrossModuleCall => 0x8009,
            Opcode::CallerModuleInternalCall => 0x800A,
            Opcode::GetGlobalStateBytes32 => 0x8010,
            Opcode::SetGlobalStateBytes32 => 0x8011,
            Opcode::GetGlobalStateU64 => 0x8012,
            Opcode::SetGlobalStateU64 => 0x8013,
            Opcode::ReadPreImage => 0x8020,
            Opcode::ReadInboxMessage => 0x8021,
            Opcode::HaltAndSetFinished => 0x8022,
        }
    }

    pub fn is_host_io(self) -> bool {
        matches!(
            self,
            Opcode::GetGlobalStateBytes32
                | Opcode::SetGlobalStateBytes32
                | Opcode::GetGlobalStateU64
                | Opcode::SetGlobalStateU64
                | Opcode::ReadPreImage
                | Opcode::ReadInboxMessage
        )
    }
}

pub type FloatingPointImpls = HashMap<FloatInstruction, (u32, u32)>;

#[derive(Clone, Copy, Debug)]
pub struct FunctionCodegenState<'a> {
    return_values: usize,
    block_depth: usize,
    floating_point_impls: &'a FloatingPointImpls,
}

impl<'a> FunctionCodegenState<'a> {
    pub fn new(return_values: usize, floating_point_impls: &'a FloatingPointImpls) -> Self {
        FunctionCodegenState {
            return_values,
            block_depth: 0,
            floating_point_impls,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Instruction {
    pub opcode: Opcode,
    pub argument_data: u64,
    pub proving_argument_data: Option<Bytes32>,
}

fn pack_call_indirect(table: u32, ty: u32) -> u64 {
    u64::from(table) | (u64::from(ty) << 32)
}

pub fn unpack_call_indirect(data: u64) -> (u32, u32) {
    (data as u32, (data >> 32) as u32)
}

pub fn pack_cross_module_call(func: u32, module: u32) -> u64 {
    u64::from(func) | (u64::from(module) << 32)
}

pub fn unpack_cross_module_call(data: u64) -> (u32, u32) {
    (data as u32, (data >> 32) as u32)
}

impl Instruction {
    pub fn simple(opcode: Opcode) -> Instruction {
        Instruction {
            opcode,
            argument_data: 0,
            proving_argument_data: None,
        }
    }

    pub fn with_data(opcode: Opcode, argument_data: u64) -> Instruction {
        Instruction {
            opcode,
            argument_data,
            proving_argument_data: None,
        }
    }

    pub fn get_proving_argument_data(self) -> Bytes32 {
        if let Some(data) = self.proving_argument_data {
            data
        } else {
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
    ) -> Result<()> {
        match inst {
            HirInstruction::Block(_, insts) => {
                let block_idx = ops.len();
                ops.push(Instruction::simple(Opcode::Block));
                state.block_depth += 1;
                for inst in insts {
                    Self::extend_from_hir(ops, state, inst)?;
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
                    Self::extend_from_hir(ops, state, inst)?;
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
                    Self::extend_from_hir(ops, state, inst)?;
                }
                ops.push(Instruction::simple(Opcode::Branch));

                ops[jump_idx].argument_data = ops.len() as u64;
                for inst in else_insts {
                    Self::extend_from_hir(ops, state, inst)?;
                }
                ops.push(Instruction::simple(Opcode::EndBlock));
                ops[block_idx].argument_data = ops.len() as u64;
            }
            HirInstruction::Branch(x) => {
                assert!(x < state.block_depth as u32);
                for _ in 0..x {
                    ops.push(Instruction::simple(Opcode::EndBlock));
                }
                ops.push(Instruction::simple(Opcode::Branch));
            }
            HirInstruction::BranchIf(x) => {
                assert!(x < state.block_depth as u32);
                for _ in 0..x {
                    ops.push(Instruction::simple(Opcode::EndBlockIf));
                }
                ops.push(Instruction::simple(Opcode::BranchIf));
            }
            HirInstruction::BranchTable(options, default) => {
                let mut option_jumps = Vec::new();
                // Build an equivalent HirInstruction sequence without BranchTable
                for (i, option) in options.iter().enumerate() {
                    let i = match u32::try_from(i) {
                        Ok(x) => x,
                        _ => break,
                    };
                    // Evaluate this branch
                    ops.push(Instruction::simple(Opcode::Dup));
                    ops.push(Instruction::with_data(Opcode::I32Const, i.into()));
                    ops.push(Instruction::simple(Opcode::IBinOp(
                        IntegerValType::I32,
                        IBinOpType::Sub,
                    )));
                    // Jump if the subtraction resulted in 0, i.e. it matched the index
                    ops.push(Instruction::simple(Opcode::I32Eqz));
                    option_jumps.push((ops.len(), *option));
                    ops.push(Instruction::simple(Opcode::ArbitraryJumpIf));
                }
                // Nothing matched. Drop the index and jump to the default.
                ops.push(Instruction::simple(Opcode::Drop));
                Instruction::extend_from_hir(ops, state, HirInstruction::Branch(default))?;
                // Make a jump table of branches
                for (source, branch) in option_jumps {
                    ops[source].argument_data = ops.len() as u64;
                    // Drop the index and branch the target depth
                    ops.push(Instruction::simple(Opcode::Drop));
                    Instruction::extend_from_hir(ops, state, HirInstruction::Branch(branch))?;
                }
            }
            HirInstruction::LocalTee(x) => {
                // Translate into a dup then local.set
                Self::extend_from_hir(ops, state, HirInstruction::Simple(Opcode::Dup))?;
                Self::extend_from_hir(ops, state, HirInstruction::WithIdx(Opcode::LocalSet, x))?;
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
                            | Opcode::CallerModuleInternalCall
                            | Opcode::ReadInboxMessage
                    ),
                    "WithIdx HirInstruction has bad WithIdx opcode {:?}",
                    op,
                );
                ops.push(Instruction::with_data(op, x.into()));
            }
            HirInstruction::CallIndirect(table, ty) => {
                ops.push(Instruction::with_data(
                    Opcode::CallIndirect,
                    pack_call_indirect(table, ty),
                ));
            }
            HirInstruction::CrossModuleCall(module, func) => {
                ops.push(Instruction::with_data(
                    Opcode::CrossModuleCall,
                    pack_cross_module_call(func, module),
                ));
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
            HirInstruction::FloatingPointOp(inst) => {
                if let Some(&(module, func)) = state.floating_point_impls.get(&inst) {
                    let sig = inst.signature();
                    // Reinterpret float args into ints
                    for &arg in sig.inputs.iter().rev() {
                        if arg == ArbValueType::F32 {
                            ops.push(Instruction::simple(Opcode::Reinterpret(
                                ArbValueType::I32,
                                ArbValueType::F32,
                            )));
                        } else if arg == ArbValueType::F64 {
                            ops.push(Instruction::simple(Opcode::Reinterpret(
                                ArbValueType::I64,
                                ArbValueType::F64,
                            )));
                        }
                        ops.push(Instruction::simple(Opcode::MoveFromStackToInternal));
                    }
                    for _ in sig.inputs.iter() {
                        ops.push(Instruction::simple(Opcode::MoveFromInternalToStack));
                    }
                    Self::extend_from_hir(
                        ops,
                        state,
                        HirInstruction::CrossModuleCall(module, func),
                    )?;
                    // Reinterpret returned ints that should be floats into floats
                    assert!(
                        sig.outputs.len() <= 1,
                        "Floating point inst has multiple outputs"
                    );
                    let output = sig.outputs.get(0).cloned();
                    if output == Some(ArbValueType::F32) {
                        ops.push(Instruction::simple(Opcode::Reinterpret(
                            ArbValueType::F32,
                            ArbValueType::I32,
                        )));
                    } else if output == Some(ArbValueType::F64) {
                        ops.push(Instruction::simple(Opcode::Reinterpret(
                            ArbValueType::F64,
                            ArbValueType::I64,
                        )));
                    }
                } else {
                    bail!("No implementation for floating point operation {:?}", inst);
                }
            }
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
                )?;
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
        Ok(())
    }
}

pub fn wasm_to_wavm<'a>(
    code: &[Operator<'a>],
    out: &mut Vec<Instruction>,
    fp_impls: &FloatingPointImpls,
) -> Result<()> {
    use Operator::*;

    macro_rules! op {
        ($first:ident $(,$opcode:ident)*) => {
            $first $(| $opcode)*
        };
    }
    macro_rules! dot {
        ($first:ident $(,$opcode:ident)*) => {
            $first { .. } $(| $opcode { .. })*
        };
    }
    macro_rules! opcode {
        ($opcode:ident ($($inside:expr),*)) => {{
            out.push(Instruction::simple(Opcode::$opcode($($inside,)*)))
        }};
        ($opcode:ident) => {{
            out.push(Instruction::simple(Opcode::$opcode))
        }};
        ($opcode:ident, $value:expr) => {
            out.push(Instruction::with_data(Opcode::$opcode, $value))
        };
        (@cross, $func:expr, $module:expr) => {
            out.push(Instruction::with_data(
                Opcode::CrossModuleCall,
                u64::from($func) | (u64::from($module) << 32),
            ));
        };
    }
    macro_rules! load {
        ($type:ident, $memory:expr, $bytes:expr, $signed:ident) => {{
            ensure!($memory.memory == 0, "multi-memory proposal not supported");
            let op = Opcode::MemoryLoad {
                ty: ArbValueType::$type,
                bytes: $bytes,
                signed: $signed,
            };
            Instruction::with_data(op, $memory.offset);
        }};
    }
    macro_rules! store {
        ($type:ident, $memory:expr, $bytes:expr) => {{
            ensure!($memory.memory == 0, "multi-memory proposal not supported");
            let op = Opcode::MemoryStore {
                ty: ArbValueType::$type,
                bytes: $bytes,
            };
            Instruction::with_data(op, $memory.offset);
        }};
    }
    macro_rules! compare {
        ($type:ident, $rel:ident, $signed:expr) => {{
            let op = Opcode::IRelOp(IntegerValType::$type, IRelOpType::$rel, $signed);
            out.push(Instruction::simple(op));
        }};
    }
    macro_rules! unary {
        ($type:ident, $op:ident) => {{
            let op = Opcode::IUnOp(IntegerValType::$type, IUnOpType::$op);
            out.push(Instruction::simple(op));
        }};
    }
    macro_rules! binary {
        ($type:ident, $op:ident) => {{
            let op = Opcode::IBinOp(IntegerValType::$type, IBinOpType::$op);
            out.push(Instruction::simple(op));
        }};
    }
    macro_rules! reinterpret {
        ($dest:ident, $source:ident) => {{
            let op = Opcode::Reinterpret(ArbValueType::$dest, ArbValueType::$source);
            out.push(Instruction::simple(op));
        }};
    }
    macro_rules! float {
        ($func:ident) => {
            float!(@impl $func)
        };
        ($func:ident $(,$data:ident)+) => {
            float!(@impl $func($($data),+))
        };
        (@impl $func:expr) => {{
            #[allow(unused_imports)]
            use crate::{
                binary::{FloatInstruction::*, FloatType::*, FloatUnOp::*, FloatBinOp::*, FloatRelOp::*},
                value::IntegerValType::*,
            };

            let func = $func;
            let sig = func.signature();
            let (module, func) = match fp_impls.get(&func) {
                Some((module, func)) => (module, func),
                None => bail!("No implementation for floating point operation {:?}", &func),
            };

            // Reinterpret float args into ints
            for &arg in sig.inputs.iter().rev() {
                match arg {
                    ArbValueType::I32 | ArbValueType::I64 => {}
                    ArbValueType::F32 => opcode!(Reinterpret(ArbValueType::I32, ArbValueType::F32)),
                    ArbValueType::F64 => opcode!(Reinterpret(ArbValueType::I64, ArbValueType::F64)),
                    _ => bail!("Floating point operation {:?} has bad args", &func),
                }
                opcode!(MoveFromStackToInternal)
            }
            for _ in &sig.inputs {
                opcode!(MoveFromInternalToStack)
            }
            opcode!(@cross, *module, *func);

            // Reinterpret returned ints that should be floats into floats
            let outputs = sig.outputs;
            match outputs.get(0) {
                Some(ArbValueType::F32) if outputs.len() == 1 => reinterpret!(F32, I32),
                Some(ArbValueType::F64) if outputs.len() == 1 => reinterpret!(F64, I64),
                _ => panic!("Floating point op {:?} should have 1 output but has {}", func, outputs.len()),
            }
        }};
    }

    for op in code {
        #[rustfmt::skip]
        match op {
            Unreachable => opcode!(Unreachable),
            Nop => opcode!(Nop),
            Block { ty } => {}
            Loop { ty } => {}
            If { ty } => {}
            Else => {}

            unsupported @ dot!(Try, Catch, Throw, Rethrow) => {
                bail!("exception-handling extension not supported {:?}", unsupported)
            },

            End => {}
            Br { relative_depth } => {}
            BrIf { relative_depth } => {}
            BrTable { table } => {}
            Return => {}
            Call { function_index } => {}
            CallIndirect {
                index,
                table_index,
                table_byte,
            } => {}
            ReturnCall { function_index } => {}
            ReturnCallIndirect { index, table_index } => {}

            unsupported @ (dot!(Delegate) | op!(CatchAll)) => {
                bail!("exception-handling extension not supported {:?}", unsupported)
            },

            Drop => opcode!(Drop),
            Select => opcode!(Select),

            unsupported @ dot!(TypedSelect) => {
                bail!("reference-types extension not supported {:?}", unsupported)
            },

            LocalGet { local_index } => opcode!(LocalGet, *local_index as u64),
            LocalSet { local_index } => opcode!(LocalSet, *local_index as u64),
            LocalTee { local_index } => {
                opcode!(LocalSet, *local_index as u64);
                opcode!(LocalGet, *local_index as u64);
            },
            GlobalGet { global_index } => opcode!(GlobalGet, *global_index as u64),
            GlobalSet { global_index } => opcode!(GlobalSet, *global_index as u64),
            I32Load { memarg } => load!(I32, memarg, 4, false),
            I64Load { memarg } => load!(I64, memarg, 8, false),
            F32Load { memarg } => load!(F32, memarg, 4, false),
            F64Load { memarg } => load!(F64, memarg, 8, false),
            I32Load8S { memarg } => load!(I32, memarg, 1, true),
            I32Load8U { memarg } => load!(I32, memarg, 1, false),
            I32Load16S { memarg } => load!(I32, memarg, 2, true),
            I32Load16U { memarg } => load!(I32, memarg, 1, false),
            I64Load8S { memarg } => load!(I64, memarg, 1, true),
            I64Load8U { memarg } => load!(I64, memarg, 1, false),
            I64Load16S { memarg } => load!(I64, memarg, 2, true),
            I64Load16U { memarg } => load!(I64, memarg, 2, false),
            I64Load32S { memarg } => load!(I64, memarg, 4, true),
            I64Load32U { memarg } => load!(I64, memarg, 4, false),
            I32Store { memarg } => store!(I32, memarg, 4),
            I64Store { memarg } => store!(I64, memarg, 8),
            F32Store { memarg } => store!(F32, memarg, 4),
            F64Store { memarg } => store!(F64, memarg, 8),
            I32Store8 { memarg } => store!(I32, memarg, 1),
            I32Store16 { memarg } => store!(I32, memarg, 2),
            I64Store8 { memarg } => store!(I64, memarg, 1),
            I64Store16 { memarg } => store!(I64, memarg, 2),
            I64Store32 { memarg } => store!(I64, memarg, 4),
            MemorySize { mem, mem_byte } => {
                ensure!(*mem == 0 && *mem_byte == 0, "MemorySize args must be 0");
                opcode!(MemorySize)
            }
            MemoryGrow { mem, mem_byte } => {
                ensure!(*mem == 0 && *mem_byte == 0, "MemoryGrow args must be 0");
                opcode!(MemoryGrow)
            }
            I32Const { value } => opcode!(I32Const, *value as u64),
            I64Const { value } => opcode!(I64Const, *value as u64),
            F32Const { value } => opcode!(F32Const, value.bits() as u64),
            F64Const { value } => opcode!(F64Const, value.bits()),

            unsupported @ (dot!(RefNull) | op!(RefIsNull) | dot!(RefFunc)) => {
                bail!("reference-types extension not supported {:?}", unsupported)
            },

            I32Eqz => opcode!(I32Eqz),
            I32Eq => compare!(I32, Eq, false),
            I32Ne => compare!(I32, Ne, false),
            I32LtS => compare!(I32, Lt, true),
            I32LtU => compare!(I32, Lt, false),
            I32GtS => compare!(I32, Gt, true),
            I32GtU => compare!(I32, Gt, false),
            I32LeS => compare!(I32, Le, true),
            I32LeU => compare!(I32, Le, false),
            I32GeS => compare!(I32, Ge, true),
            I32GeU => compare!(I32, Ge, false),
            I64Eqz => opcode!(I64Eqz),
            I64Eq => compare!(I64, Eq, false),
            I64Ne => compare!(I64, Ne, false),
            I64LtS => compare!(I64, Lt, true),
            I64LtU => compare!(I64, Lt, false),
            I64GtS => compare!(I64, Gt, true),
            I64GtU => compare!(I64, Gt, false),
            I64LeS => compare!(I64, Le, true),
            I64LeU => compare!(I64, Le, false),
            I64GeS => compare!(I64, Ge, true),
            I64GeU => compare!(I64, Ge, false),
            F32Eq => float!(RelOp, F32, Eq),
            F32Ne => float!(RelOp, F32, Ne),
            F32Lt => float!(RelOp, F32, Lt),
            F32Gt => float!(RelOp, F32, Gt),
            F32Le => float!(RelOp, F32, Le),
            F32Ge => float!(RelOp, F32, Ge),
            F64Eq => float!(RelOp, F64, Eq),
            F64Ne => float!(RelOp, F64, Ne),
            F64Lt => float!(RelOp, F64, Lt),
            F64Gt => float!(RelOp, F64, Gt),
            F64Le => float!(RelOp, F64, Le),
            F64Ge => float!(RelOp, F64, Ge),
            I32Clz => unary!(I32, Clz),
            I32Ctz => unary!(I32, Ctz),
            I32Popcnt => unary!(I32, Popcnt),
            I32Add => binary!(I32, Add),
            I32Sub => binary!(I32, Sub),
            I32Mul => binary!(I32, Mul),
            I32DivS => binary!(I32, DivS),
            I32DivU => binary!(I32, DivU),
            I32RemS => binary!(I32, RemS),
            I32RemU => binary!(I32, RemU),
            I32And => binary!(I32, And),
            I32Or => binary!(I32, Or),
            I32Xor => binary!(I32, Xor),
            I32Shl => binary!(I32, Shl),
            I32ShrS => binary!(I32, ShrS),
            I32ShrU => binary!(I32, ShrU),
            I32Rotl => binary!(I32, Rotl),
            I32Rotr => binary!(I32, Rotr),
            I64Clz => unary!(I64, Clz),
            I64Ctz => unary!(I64, Ctz),
            I64Popcnt => unary!(I64, Popcnt),
            I64Add => binary!(I64, Add),
            I64Sub => binary!(I64, Sub),
            I64Mul => binary!(I64, Mul),
            I64DivS => binary!(I64, DivS),
            I64DivU => binary!(I64, DivU),
            I64RemS => binary!(I64, RemS),
            I64RemU => binary!(I64, RemU),
            I64And => binary!(I64, And),
            I64Or => binary!(I64, Or),
            I64Xor => binary!(I64, Xor),
            I64Shl => binary!(I64, Shl),
            I64ShrS => binary!(I64, ShrS),
            I64ShrU => binary!(I64, ShrU),
            I64Rotl => binary!(I64, Rotl),
            I64Rotr => binary!(I64, Rotr),
            F32Abs => float!(UnOp, F32, Abs),
            F32Neg => float!(UnOp, F32, Neg),
            F32Ceil => float!(UnOp, F32, Ceil),
            F32Floor => float!(UnOp, F32, Floor),
            F32Trunc => float!(UnOp, F32, Trunc),
            F32Nearest => float!(UnOp, F32, Nearest),
            F32Sqrt => float!(UnOp, F32, Sqrt),
            F32Add => float!(BinOp, F32, Add),
            F32Sub => float!(BinOp, F32, Sub),
            F32Mul => float!(BinOp, F32, Mul),
            F32Div => float!(BinOp, F32, Div),
            F32Min => float!(BinOp, F32, Min),
            F32Max => float!(BinOp, F32, Max),
            F32Copysign => float!(BinOp, F32, CopySign),
            F64Abs => float!(UnOp, F64, Abs),
            F64Neg => float!(UnOp, F64, Neg),
            F64Ceil => float!(UnOp, F64, Ceil),
            F64Floor => float!(UnOp, F64, Floor),
            F64Trunc => float!(UnOp, F64, Trunc),
            F64Nearest => float!(UnOp, F64, Nearest),
            F64Sqrt => float!(UnOp, F64, Sqrt),
            F64Add => float!(BinOp, F64, Add),
            F64Sub => float!(BinOp, F64, Sub),
            F64Mul => float!(BinOp, F64, Mul),
            F64Div => float!(BinOp, F64, Div),
            F64Min => float!(BinOp, F64, Min),
            F64Max => float!(BinOp, F64, Max),
            F64Copysign => float!(BinOp, F64, CopySign),
            I32WrapI64 => opcode!(I32WrapI64),
            I32TruncF32S => float!(TruncIntOp, I32, F32, true),
            I32TruncF32U => float!(TruncIntOp, I32, F32, false),
            I32TruncF64S => float!(TruncIntOp, I32, F64, true),
            I32TruncF64U => float!(TruncIntOp, I32, F64, false),
            I64ExtendI32S => opcode!(I64ExtendI32(true)),
            I64ExtendI32U => opcode!(I64ExtendI32(false)),
            I64TruncF32S => float!(TruncIntOp, I64, F32, true),
            I64TruncF32U => float!(TruncIntOp, I64, F32, false),
            I64TruncF64S => float!(TruncIntOp, I64, F64, true),
            I64TruncF64U => float!(TruncIntOp, I64, F64, false),
            F32ConvertI32S => float!(ConvertIntOp, F32, I32, true),
            F32ConvertI32U => float!(ConvertIntOp, F32, I32, false),
            F32ConvertI64S => float!(ConvertIntOp, F32, I64, true),
            F32ConvertI64U => float!(ConvertIntOp, F32, I64, false),
            F32DemoteF64 => float!(F32DemoteF64),
            F64ConvertI32S => float!(ConvertIntOp, F64, I32, true),
            F64ConvertI32U => float!(ConvertIntOp, F64, I32, false),
            F64ConvertI64S => float!(ConvertIntOp, F64, I64, true),
            F64ConvertI64U => float!(ConvertIntOp, F64, I64, false),
            F64PromoteF32 => float!(F64PromoteF32),
            I32ReinterpretF32 => reinterpret!(I32, F32),
            I64ReinterpretF64 => reinterpret!(I64, F64),
            F32ReinterpretI32 => reinterpret!(F32, I32),
            F64ReinterpretI64 => reinterpret!(F64, I64),
            I32Extend8S => opcode!(I32ExtendS(8)),
            I32Extend16S => opcode!(I32ExtendS(16)),
            I64Extend8S => opcode!(I64ExtendS(8)),
            I64Extend16S => opcode!(I64ExtendS(16)),
            I64Extend32S => opcode!(I64ExtendS(32)),
            I32TruncSatF32S => float!(TruncIntOp, I32, F32, true),
            I32TruncSatF32U => float!(TruncIntOp, I32, F32, false),
            I32TruncSatF64S => float!(TruncIntOp, I32, F64, true),
            I32TruncSatF64U => float!(TruncIntOp, I32, F64, false),
            I64TruncSatF32S => float!(TruncIntOp, I64, F32, true),
            I64TruncSatF32U => float!(TruncIntOp, I64, F32, false),
            I64TruncSatF64S => float!(TruncIntOp, I64, F64, true),
            I64TruncSatF64U => float!(TruncIntOp, I64, F64, false),

            unsupported @ (
                dot!(
                    MemoryInit, DataDrop, MemoryCopy, MemoryFill, TableInit, ElemDrop,
                    TableCopy, TableFill, TableGet, TableSet, TableGrow, TableSize
                )
            ) => bail!("bulk-memory-operations extension not supported {:?}", unsupported),

            unsupported @ (
                dot!(
                    MemoryAtomicNotify, MemoryAtomicWait32, MemoryAtomicWait64, AtomicFence, I32AtomicLoad,
                    I64AtomicLoad, I32AtomicLoad8U, I32AtomicLoad16U, I64AtomicLoad8U, I64AtomicLoad16U,
                    I64AtomicLoad32U, I32AtomicStore, I64AtomicStore, I32AtomicStore8, I32AtomicStore16,
                    I64AtomicStore8, I64AtomicStore16, I64AtomicStore32, I32AtomicRmwAdd, I64AtomicRmwAdd,
                    I32AtomicRmw8AddU, I32AtomicRmw16AddU, I64AtomicRmw8AddU, I64AtomicRmw16AddU, I64AtomicRmw32AddU,
                    I32AtomicRmwSub, I64AtomicRmwSub, I32AtomicRmw8SubU, I32AtomicRmw16SubU, I64AtomicRmw8SubU,
                    I64AtomicRmw16SubU, I64AtomicRmw32SubU, I32AtomicRmwAnd, I64AtomicRmwAnd, I32AtomicRmw8AndU,
                    I32AtomicRmw16AndU, I64AtomicRmw8AndU, I64AtomicRmw16AndU, I64AtomicRmw32AndU, I32AtomicRmwOr,
                    I64AtomicRmwOr, I32AtomicRmw8OrU, I32AtomicRmw16OrU, I64AtomicRmw8OrU, I64AtomicRmw16OrU,
                    I64AtomicRmw32OrU, I32AtomicRmwXor, I64AtomicRmwXor, I32AtomicRmw8XorU, I32AtomicRmw16XorU,
                    I64AtomicRmw8XorU, I64AtomicRmw16XorU, I64AtomicRmw32XorU, I32AtomicRmwXchg, I64AtomicRmwXchg,
                    I32AtomicRmw8XchgU, I32AtomicRmw16XchgU, I64AtomicRmw8XchgU, I64AtomicRmw16XchgU,
                    I64AtomicRmw32XchgU, I32AtomicRmwCmpxchg, I64AtomicRmwCmpxchg, I32AtomicRmw8CmpxchgU,
                    I32AtomicRmw16CmpxchgU, I64AtomicRmw8CmpxchgU, I64AtomicRmw16CmpxchgU, I64AtomicRmw32CmpxchgU
                )
            ) => bail!("threads extension not supported {:?}", unsupported),

            unsupported @ (
                dot!(
                    V128Load, V128Load8x8S, V128Load8x8U, V128Load16x4S, V128Load16x4U, V128Load32x2S, V128Load32x2U,
                    V128Load8Splat, V128Load16Splat, V128Load32Splat, V128Load64Splat, V128Load32Zero, V128Load64Zero,
                    V128Store, V128Load8Lane, V128Load16Lane, V128Load32Lane, V128Load64Lane, V128Store8Lane,
                    V128Store16Lane, V128Store32Lane, V128Store64Lane, V128Const,
                    I8x16Shuffle, I8x16ExtractLaneS, I8x16ExtractLaneU, I8x16ReplaceLane, I16x8ExtractLaneS,
                    I16x8ExtractLaneU, I16x8ReplaceLane, I32x4ExtractLane, I32x4ReplaceLane, I64x2ExtractLane,
                    I64x2ReplaceLane, F32x4ExtractLane, F32x4ReplaceLane, F64x2ExtractLane, F64x2ReplaceLane,
                    I8x16Swizzle, I8x16Splat, I16x8Splat, I32x4Splat, I64x2Splat, F32x4Splat, F64x2Splat, I8x16Eq,
                    I8x16Ne, I8x16LtS, I8x16LtU, I8x16GtS, I8x16GtU, I8x16LeS, I8x16LeU, I8x16GeS, I8x16GeU, I16x8Eq,
                    I16x8Ne, I16x8LtS, I16x8LtU, I16x8GtS, I16x8GtU, I16x8LeS, I16x8LeU, I16x8GeS, I16x8GeU, I32x4Eq,
                    I32x4Ne, I32x4LtS, I32x4LtU, I32x4GtS, I32x4GtU, I32x4LeS, I32x4LeU, I32x4GeS, I32x4GeU, I64x2Eq,
                    I64x2Ne, I64x2LtS, I64x2GtS, I64x2LeS, I64x2GeS,
                    F32x4Eq, F32x4Ne, F32x4Lt, F32x4Gt, F32x4Le, F32x4Ge,
                    F64x2Eq, F64x2Ne, F64x2Lt, F64x2Gt, F64x2Le, F64x2Ge,
                    V128Not, V128And, V128AndNot, V128Or, V128Xor, V128Bitselect, V128AnyTrue,
                    I8x16Abs, I8x16Neg, I8x16Popcnt, I8x16AllTrue, I8x16Bitmask, I8x16NarrowI16x8S, I8x16NarrowI16x8U,
                    I8x16Shl, I8x16ShrS, I8x16ShrU, I8x16Add, I8x16AddSatS, I8x16AddSatU, I8x16Sub, I8x16SubSatS,
                    I8x16SubSatU, I8x16MinS, I8x16MinU, I8x16MaxS, I8x16MaxU, I8x16RoundingAverageU,
                    I16x8ExtAddPairwiseI8x16S, I16x8ExtAddPairwiseI8x16U, I16x8Abs, I16x8Neg, I16x8Q15MulrSatS,
                    I16x8AllTrue, I16x8Bitmask, I16x8NarrowI32x4S, I16x8NarrowI32x4U, I16x8ExtendLowI8x16S,
                    I16x8ExtendHighI8x16S, I16x8ExtendLowI8x16U, I16x8ExtendHighI8x16U, I16x8Shl, I16x8ShrS, I16x8ShrU,
                    I16x8Add, I16x8AddSatS, I16x8AddSatU, I16x8Sub, I16x8SubSatS, I16x8SubSatU, I16x8Mul, I16x8MinS,
                    I16x8MinU, I16x8MaxS, I16x8MaxU, I16x8RoundingAverageU, I16x8ExtMulLowI8x16S,
                    I16x8ExtMulHighI8x16S, I16x8ExtMulLowI8x16U, I16x8ExtMulHighI8x16U, I32x4ExtAddPairwiseI16x8S,
                    I32x4ExtAddPairwiseI16x8U, I32x4Abs, I32x4Neg, I32x4AllTrue, I32x4Bitmask, I32x4ExtendLowI16x8S,
                    I32x4ExtendHighI16x8S, I32x4ExtendLowI16x8U, I32x4ExtendHighI16x8U, I32x4Shl, I32x4ShrS, I32x4ShrU,
                    I32x4Add, I32x4Sub, I32x4Mul, I32x4MinS, I32x4MinU, I32x4MaxS, I32x4MaxU, I32x4DotI16x8S,
                    I32x4ExtMulLowI16x8S, I32x4ExtMulHighI16x8S, I32x4ExtMulLowI16x8U, I32x4ExtMulHighI16x8U, I64x2Abs,
                    I64x2Neg, I64x2AllTrue, I64x2Bitmask, I64x2ExtendLowI32x4S, I64x2ExtendHighI32x4S,
                    I64x2ExtendLowI32x4U, I64x2ExtendHighI32x4U, I64x2Shl, I64x2ShrS, I64x2ShrU, I64x2Add, I64x2Sub,
                    I64x2Mul, I64x2ExtMulLowI32x4S, I64x2ExtMulHighI32x4S, I64x2ExtMulLowI32x4U, I64x2ExtMulHighI32x4U,
                    F32x4Ceil, F32x4Floor, F32x4Trunc, F32x4Nearest, F32x4Abs, F32x4Neg, F32x4Sqrt, F32x4Add, F32x4Sub,
                    F32x4Mul, F32x4Div, F32x4Min, F32x4Max, F32x4PMin, F32x4PMax, F64x2Ceil, F64x2Floor, F64x2Trunc,
                    F64x2Nearest, F64x2Abs, F64x2Neg, F64x2Sqrt, F64x2Add, F64x2Sub, F64x2Mul, F64x2Div, F64x2Min,
                    F64x2Max, F64x2PMin, F64x2PMax, I32x4TruncSatF32x4S, I32x4TruncSatF32x4U, F32x4ConvertI32x4S,
                    F32x4ConvertI32x4U, I32x4TruncSatF64x2SZero, I32x4TruncSatF64x2UZero, F64x2ConvertLowI32x4S,
                    F64x2ConvertLowI32x4U, F32x4DemoteF64x2Zero, F64x2PromoteLowF32x4, I8x16RelaxedSwizzle,
                    I32x4RelaxedTruncSatF32x4S, I32x4RelaxedTruncSatF32x4U, I32x4RelaxedTruncSatF64x2SZero,
                    I32x4RelaxedTruncSatF64x2UZero, F32x4Fma, F32x4Fms, F64x2Fma, F64x2Fms, I8x16LaneSelect,
                    I16x8LaneSelect, I32x4LaneSelect, I64x2LaneSelect, F32x4RelaxedMin, F32x4RelaxedMax,
                    F64x2RelaxedMin, F64x2RelaxedMax
                )
            ) => bail!("SIMD extension not supported {:?}", unsupported)
        };
    }

    Ok(())
}
