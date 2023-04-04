// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    machine::{Function, InboxIdentifier},
    value::{ArbValueType, FunctionType},
    wavm::{Instruction, Opcode},
};
use arbutil::Color;
use eyre::{bail, Result};

/// Represents the internal hostio functions a module may have.
#[repr(u64)]
enum InternalFunc {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
}

impl InternalFunc {
    fn ty(&self) -> FunctionType {
        use ArbValueType::*;
        FunctionType::new(vec![I32], vec![I32])
    }
}

pub fn get_impl(module: &str, name: &str) -> Result<Function> {
    macro_rules! func {
        () => {
            FunctionType::default()
        };
        ([$($args:expr),*]) => {
            FunctionType::new(vec![$($args),*], vec![])
        };
        ([$($args:expr),*], [$($outs:expr),*]) => {
            FunctionType::new(vec![$($args),*], vec![$($outs),*])
        };
    }

    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;
    #[rustfmt::skip]
    let ty = match (module, name) {
        ("env", "wavm_caller_load8")   => func!([I32], [I32]),
        ("env", "wavm_caller_load32")  => func!([I32], [I32]),
        ("env", "wavm_caller_store8")  => func!([I32, I32]),
        ("env", "wavm_caller_store32") => func!([I32, I32]),
        ("env", "wavm_get_globalstate_bytes32") => func!([I32, I32]),
        ("env", "wavm_set_globalstate_bytes32") => func!([I32, I32]),
        ("env", "wavm_get_globalstate_u64")     => func!([I32], [I64]),
        ("env", "wavm_set_globalstate_u64")     => func!([I32, I64]),
        ("env", "wavm_read_pre_image")          => func!([I32, I32], [I32]),
        ("env", "wavm_read_inbox_message")      => func!([I64, I32, I32], [I32]),
        ("env", "wavm_read_delayed_inbox_message") => func!([I64, I32, I32], [I32]),
        ("env", "wavm_halt_and_set_finished")      => func!(),
        _ => bail!("no such hostio {} in {}", name.red(), module.red()),
    };

    let append = |code: &mut Vec<Instruction>| {
        macro_rules! opcode {
            ($opcode:expr) => {
                code.push(Instruction::simple($opcode))
            };
            ($opcode:expr, $value:expr) => {
                code.push(Instruction::with_data($opcode, $value as u64))
            };
        }

        match (module, name) {
            ("env", "wavm_caller_load8") => {
                opcode!(LocalGet, 0);
                opcode!(CallerModuleInternalCall, WavmCallerLoad8);
            }
            ("env", "wavm_caller_load32") => {
                opcode!(LocalGet, 0);
                opcode!(CallerModuleInternalCall, WavmCallerLoad32);
            }
            ("env", "wavm_caller_store8") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(CallerModuleInternalCall, WavmCallerStore8);
            }
            ("env", "wavm_caller_store32") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(CallerModuleInternalCall, WavmCallerStore32);
            }
            ("env", "wavm_get_globalstate_bytes32") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(GetGlobalStateBytes32);
            }
            ("env", "wavm_set_globalstate_bytes32") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(SetGlobalStateBytes32);
            }
            ("env", "wavm_get_globalstate_u64") => {
                opcode!(LocalGet, 0);
                opcode!(GetGlobalStateU64);
            }
            ("env", "wavm_set_globalstate_u64") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(SetGlobalStateU64);
            }
            ("env", "wavm_read_pre_image") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(ReadPreImage);
            }
            ("env", "wavm_read_inbox_message") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(LocalGet, 2);
                opcode!(ReadInboxMessage, InboxIdentifier::Sequencer);
            }
            ("env", "wavm_read_delayed_inbox_message") => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(LocalGet, 2);
                opcode!(ReadInboxMessage, InboxIdentifier::Delayed);
            }
            ("env", "wavm_halt_and_set_finished") => {
                opcode!(HaltAndSetFinished);
            }
            _ => bail!("no such hostio {} in {}", name.red(), module.red()),
        }
        Ok(())
    };

    Function::new(&[], append, ty, &[])
}

/// Adds internal functions to a module.
/// Note: the order of the functions must match that of the `InternalFunc` enum
pub fn add_internal_funcs(funcs: &mut Vec<Function>, func_types: &mut Vec<FunctionType>) {
    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;

    fn code_func(code: Vec<Instruction>, ty: FunctionType) -> Function {
        let mut wavm = vec![Instruction::simple(InitFrame)];
        wavm.extend(code);
        wavm.push(Instruction::simple(Return));
        Function::new_from_wavm(wavm, ty, vec![])
    }

    fn op_func(opcode: Opcode, ty: FunctionType) -> Function {
        code_func(vec![Instruction::simple(opcode)], ty)
    }

    let mut host = |func: InternalFunc| -> FunctionType {
        let ty = func.ty();
        func_types.push(ty.clone());
        ty
    };

    // order matters!
    funcs.push(op_func(
        MemoryLoad {
            ty: I32,
            bytes: 1,
            signed: false,
        },
        host(WavmCallerLoad8),
    ));
    funcs.push(op_func(
        MemoryLoad {
            ty: I32,
            bytes: 4,
            signed: false,
        },
        host(WavmCallerLoad32),
    ));
    funcs.push(op_func(
        MemoryStore { ty: I32, bytes: 1 },
        host(WavmCallerStore8),
    ));
    funcs.push(op_func(
        MemoryStore { ty: I32, bytes: 4 },
        host(WavmCallerStore32),
    ));
}
