// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    machine::{Function, InboxIdentifier},
    programs::StylusGlobals,
    value::{ArbValueType, FunctionType, IntegerValType},
    wavm::{IBinOpType, Instruction, Opcode},
};
use arbutil::Color;

/// Represents the internal hostio functions a module may have.
#[repr(u64)]
enum InternalFunc {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
    UserGasLeft,
    UserGasStatus,
    UserSetGas,
    UserStackLeft,
    UserSetStack,
}

impl InternalFunc {
    fn ty(&self) -> FunctionType {
        use ArbValueType::*;
        FunctionType::new(vec![I32], vec![I32])
    }
}

pub fn get_host_impl(module: &str, name: &str) -> eyre::Result<Function> {
    let mut out = vec![];
    let ty;

    macro_rules! opcode {
        ($opcode:expr) => {
            out.push(Instruction::simple($opcode))
        };
        ($opcode:expr, $value:expr) => {
            out.push(Instruction::with_data($opcode, $value as u64))
        };
    }
    macro_rules! dynamic {
        ($func:expr) => {
            opcode!(LocalGet, 0); // module
            opcode!(LocalGet, 1); // internals offset
            opcode!(I32Const, $func); // relative position of the func
            opcode!(IBinOp(IntegerValType::I32, IBinOpType::Add)); // absolute position of the func
            opcode!(CrossModuleDynamicCall); // consumes module and func
        };
    }

    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;
    match (module, name) {
        ("env", "wavm_caller_load8") => {
            ty = FunctionType::new(vec![I32], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(CallerModuleInternalCall, WavmCallerLoad8);
        }
        ("env", "wavm_caller_load32") => {
            ty = FunctionType::new(vec![I32], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(CallerModuleInternalCall, WavmCallerLoad32);
        }
        ("env", "wavm_caller_store8") => {
            ty = FunctionType::new(vec![I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(CallerModuleInternalCall, WavmCallerStore8);
        }
        ("env", "wavm_caller_store32") => {
            ty = FunctionType::new(vec![I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(CallerModuleInternalCall, WavmCallerStore32);
        }
        ("env", "wavm_get_globalstate_bytes32") => {
            ty = FunctionType::new(vec![I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(GetGlobalStateBytes32);
        }
        ("env", "wavm_set_globalstate_bytes32") => {
            ty = FunctionType::new(vec![I32; 2], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(SetGlobalStateBytes32);
        }
        ("env", "wavm_get_globalstate_u64") => {
            ty = FunctionType::new(vec![I32], vec![I64]);
            opcode!(LocalGet, 0);
            opcode!(GetGlobalStateU64);
        }
        ("env", "wavm_set_globalstate_u64") => {
            ty = FunctionType::new(vec![I32, I64], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(SetGlobalStateU64);
        }
        ("env", "wavm_read_pre_image") => {
            ty = FunctionType::new(vec![I32; 2], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(ReadPreImage);
        }
        ("env", "wavm_read_inbox_message") => {
            ty = FunctionType::new(vec![I64, I32, I32], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(LocalGet, 2);
            opcode!(ReadInboxMessage, InboxIdentifier::Sequencer);
        }
        ("env", "wavm_read_delayed_inbox_message") => {
            ty = FunctionType::new(vec![I64, I32, I32], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(LocalGet, 2);
            opcode!(ReadInboxMessage, InboxIdentifier::Delayed);
        }
        ("env", "wavm_halt_and_set_finished") => {
            ty = FunctionType::default();
            opcode!(HaltAndSetFinished);
        }
        ("hostio", "user_gas_left") => {
            // λ() -> gas_left
            ty = FunctionType::new(vec![], vec![I64]);
            opcode!(CallerModuleInternalCall, UserGasLeft);
        }
        ("hostio", "user_gas_status") => {
            // λ() -> gas_status
            ty = FunctionType::new(vec![], vec![I32]);
            opcode!(CallerModuleInternalCall, UserGasStatus);
        }
        ("hostio", "user_set_gas") => {
            // λ(gas_left, gas_status)
            ty = FunctionType::new(vec![I64, I32], vec![]);
            opcode!(LocalGet, 0);
            opcode!(LocalGet, 1);
            opcode!(CallerModuleInternalCall, UserSetGas);
        }
        ("hostio", "link_module") => {
            // λ(module_hash)
            ty = FunctionType::new(vec![I32], vec![I32]);
            opcode!(LocalGet, 0);
            opcode!(LinkModule);
        }
        ("hostio", "unlink_module") => {
            // λ()
            ty = FunctionType::new(vec![], vec![]);
            opcode!(UnlinkModule);
        }
        ("hostio", "program_gas_left") => {
            // λ(module, internals) -> gas_left
            ty = FunctionType::new(vec![I32, I32], vec![I64]);
            dynamic!(UserGasLeft);
        }
        ("hostio", "program_gas_status") => {
            // λ(module, internals) -> gas_status
            ty = FunctionType::new(vec![I32, I32], vec![I32]);
            dynamic!(UserGasStatus);
        }
        ("hostio", "program_set_gas") => {
            // λ(module, internals, gas_left)
            ty = FunctionType::new(vec![I32, I32, I64], vec![]);
            opcode!(LocalGet, 2); // gas_left
            opcode!(I32Const, 0); // gas_status
            dynamic!(UserSetGas);
        }
        ("hostio", "program_stack_left") => {
            // λ(module, internals) -> stack_left
            ty = FunctionType::new(vec![I32, I32], vec![I32]);
            dynamic!(UserStackLeft);
        }
        ("hostio", "program_set_stack") => {
            // λ(module, internals, stack_left)
            ty = FunctionType::new(vec![I32, I32, I32], vec![]);
            opcode!(LocalGet, 2); // stack_left
            dynamic!(UserSetStack);
        }
        ("hostio", "program_call_main") => {
            // λ(module, main, args_len) -> status
            ty = FunctionType::new(vec![I32, I32, I32], vec![I32]);
            opcode!(LocalGet, 2); // args_len
            opcode!(LocalGet, 0); // module
            opcode!(LocalGet, 1); // main
            opcode!(CrossModuleDynamicCall) // consumes module and main, passing args_len
        }
        _ => eyre::bail!("no such hostio {} in {}", name.red(), module.red()),
    }

    let append = |code: &mut Vec<Instruction>| {
        code.extend(out);
        Ok(())
    };

    Function::new(&[], append, ty, &[])
}

/// Adds internal functions to a module.
/// Note: the order of the functions must match that of the `InternalFunc` enum
pub fn add_internal_funcs(
    funcs: &mut Vec<Function>,
    func_types: &mut Vec<FunctionType>,
    globals: Option<StylusGlobals>,
) {
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

    if let Some(globals) = globals {
        let (gas, status, depth) = globals.offsets();
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, gas)],
            host(UserGasLeft),
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, status)],
            host(UserGasStatus),
        ));
        funcs.push(code_func(
            vec![
                Instruction::with_data(GlobalSet, status),
                Instruction::with_data(GlobalSet, gas),
            ],
            host(UserSetGas),
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, depth)],
            host(UserStackLeft),
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalSet, depth)],
            host(UserSetStack),
        ));
    }
}
