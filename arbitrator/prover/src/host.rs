// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::vec_init_then_push)]

use crate::{
    binary,
    machine::{Function, InboxIdentifier},
    programs::{run::UserOutcomeKind, StylusGlobals},
    value::{ArbValueType, FunctionType, IntegerValType},
    wavm::{wasm_to_wavm, IBinOpType, Instruction, Opcode},
};
use arbutil::Color;
use eyre::{bail, Result};
use lazy_static::lazy_static;
use std::{collections::HashMap, path::Path};

/// Represents the internal hostio functions a module may have.
#[repr(u64)]
pub enum InternalFunc {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
    MemoryFill,
    MemoryCopy,
    UserGasLeft,
    UserGasStatus,
    UserSetGas,
    UserStackLeft,
    UserSetStack,
}

impl InternalFunc {
    pub fn ty(&self) -> FunctionType {
        use ArbValueType::*;
        use InternalFunc::*;
        macro_rules! func {
            ([$($args:expr),*], [$($outs:expr),*]) => {
                FunctionType::new(vec![$($args),*], vec![$($outs),*])
            };
        }
        match self {
            WavmCallerLoad8 | WavmCallerLoad32 => func!([I32], [I32]),
            WavmCallerStore8 | WavmCallerStore32 => func!([I32, I32], []),
            MemoryFill => func!([I32, I32, I32], []),
            MemoryCopy => func!([I32, I32, I32], []),
            UserGasLeft => func!([], [I64]),
            UserGasStatus => func!([], [I32]),
            UserSetGas => func!([I64, I32], []),
            UserStackLeft => func!([], [I32]),
            UserSetStack => func!([I32], []),
        }
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
        ("hostio", "link_module")        => func!([I32], [I32]),           // λ(module_hash) -> module
        ("hostio", "unlink_module")      => func!(),                       // λ()
        ("hostio", "user_gas_left")      => func!([], [I64]),              // λ() -> gas_left
        ("hostio", "user_gas_status")    => func!([], [I32]),              // λ() -> gas_status
        ("hostio", "user_set_gas")       => func!([I64, I32]),             // λ(gas_left, gas_status)
        ("hostio", "program_gas_left")   => func!([I32, I32], [I64]),      // λ(module, internals) -> gas_left
        ("hostio", "program_gas_status") => func!([I32, I32], [I32]),      // λ(module, internals) -> gas_status
        ("hostio", "program_stack_left") => func!([I32, I32], [I32]),      // λ(module, internals) -> stack_left
        ("hostio", "program_set_gas")    => func!([I32, I32, I64]),        // λ(module, internals, gas_left)
        ("hostio", "program_set_stack")  => func!([I32, I32, I32]),        // λ(module, internals, stack_left)
        ("hostio", "program_call_main")  => func!([I32, I32, I32], [I32]), // λ(module, main, args_len) -> status
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
        macro_rules! dynamic {
            ($func:expr) => {
                opcode!(LocalGet, 0); // module
                opcode!(LocalGet, 1); // internals offset
                opcode!(I32Const, $func); // relative position of the func
                opcode!(IBinOp(IntegerValType::I32, IBinOpType::Add)); // absolute position of the func
                opcode!(CrossModuleDynamicCall); // consumes module and func
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
            ("hostio", "user_gas_left") => {
                // λ() -> gas_left
                opcode!(CallerModuleInternalCall, UserGasLeft);
            }
            ("hostio", "user_gas_status") => {
                // λ() -> gas_status
                opcode!(CallerModuleInternalCall, UserGasStatus);
            }
            ("hostio", "user_set_gas") => {
                // λ(gas_left, gas_status)
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(CallerModuleInternalCall, UserSetGas);
            }
            ("hostio", "link_module") => {
                // λ(module_hash) -> module
                opcode!(LocalGet, 0);
                opcode!(LinkModule);
            }
            ("hostio", "unlink_module") => {
                // λ()
                opcode!(UnlinkModule);
            }
            ("hostio", "program_gas_left") => {
                // λ(module, internals) -> gas_left
                dynamic!(UserGasLeft);
            }
            ("hostio", "program_gas_status") => {
                // λ(module, internals) -> gas_status
                dynamic!(UserGasStatus);
            }
            ("hostio", "program_set_gas") => {
                // λ(module, internals, gas_left)
                opcode!(LocalGet, 2); // gas_left
                opcode!(I32Const, 0); // gas_status
                dynamic!(UserSetGas);
            }
            ("hostio", "program_stack_left") => {
                // λ(module, internals) -> stack_left
                dynamic!(UserStackLeft);
            }
            ("hostio", "program_set_stack") => {
                // λ(module, internals, stack_left)
                opcode!(LocalGet, 2); // stack_left
                dynamic!(UserSetStack);
            }
            ("hostio", "program_call_main") => {
                // λ(module, main, args_len) -> status
                opcode!(PushErrorGuard);
                opcode!(ArbitraryJumpIf, code.len() + 3);
                opcode!(I32Const, UserOutcomeKind::Failure as u32);
                opcode!(Return);

                // jumps here in the happy case
                opcode!(LocalGet, 2); // args_len
                opcode!(LocalGet, 0); // module
                opcode!(LocalGet, 1); // main
                opcode!(CrossModuleDynamicCall); // consumes module and main, passing args_len
                opcode!(PopErrorGuard);
            }
            _ => bail!("no such hostio {} in {}", name.red(), module.red()),
        }
        Ok(())
    };

    Function::new(&[], append, ty, &[])
}

/// Adds internal functions to a module.
/// Note: the order of the functions must match that of the `InternalFunc` enum
pub fn new_internal_funcs(globals: Option<StylusGlobals>) -> Vec<Function> {
    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;

    fn code_func(code: Vec<Instruction>, func: InternalFunc) -> Function {
        let mut wavm = vec![Instruction::simple(InitFrame)];
        wavm.extend(code);
        wavm.push(Instruction::simple(Return));
        Function::new_from_wavm(wavm, func.ty(), vec![])
    }

    fn op_func(opcode: Opcode, func: InternalFunc) -> Function {
        code_func(vec![Instruction::simple(opcode)], func)
    }

    let mut funcs = vec![];

    // order matters!
    funcs.push(op_func(
        MemoryLoad {
            ty: I32,
            bytes: 1,
            signed: false,
        },
        WavmCallerLoad8,
    ));
    funcs.push(op_func(
        MemoryLoad {
            ty: I32,
            bytes: 4,
            signed: false,
        },
        WavmCallerLoad32,
    ));
    funcs.push(op_func(MemoryStore { ty: I32, bytes: 1 }, WavmCallerStore8));
    funcs.push(op_func(
        MemoryStore { ty: I32, bytes: 4 },
        WavmCallerStore32,
    ));

    let [memory_fill, memory_copy] = (*BULK_MEMORY_FUNCS).clone();
    funcs.push(memory_fill);
    funcs.push(memory_copy);

    if let Some(globals) = globals {
        let (gas, status, depth) = globals.offsets();
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, gas)],
            UserGasLeft,
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, status)],
            UserGasStatus,
        ));
        funcs.push(code_func(
            vec![
                Instruction::with_data(GlobalSet, status),
                Instruction::with_data(GlobalSet, gas),
            ],
            UserSetGas,
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalGet, depth)],
            UserStackLeft,
        ));
        funcs.push(code_func(
            vec![Instruction::with_data(GlobalSet, depth)],
            UserSetStack,
        ));
    }

    funcs
}

lazy_static! {
    static ref BULK_MEMORY_FUNCS: [Function; 2] = {
        let data = include_bytes!("bulk_memory.wat");
        let wasm = wat::parse_bytes(data).expect("failed to parse bulk_memory.wat");
        let bin = binary::parse(&wasm, Path::new("internal")).expect("failed to parse bulk_memory.wasm");
        [0, 1].map(|i| {
            let code = &bin.codes[i];
            let ty = &bin.types[bin.functions[i] as usize];
            let func = Function::new(
                &code.locals,
                |wasm| wasm_to_wavm(
                    &code.expr,
                    wasm,
                    &HashMap::default(), // impls don't use floating point
                    &[],                // impls don't make calls
                    &[ty.clone()],      // only type needed is the func itself
                    0,                  // -----------------------------------
                    0,                  // impls don't use other internals
                ),
                ty.clone(),
                &[] // impls don't make calls
            );
            func.expect("failed to create bulk memory func")
        })
    };
}
