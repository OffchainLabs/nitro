// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::vec_init_then_push, clippy::redundant_closure)]

use crate::{
    binary, host,
    machine::{Function, InboxIdentifier},
    programs::StylusData,
    utils,
    value::{ArbValueType, FunctionType},
    wavm::{wasm_to_wavm, Instruction, Opcode},
};
use arbutil::{evm::user::UserOutcomeKind, Color, PreimageType};
use eyre::{bail, ErrReport, Result};
use lazy_static::lazy_static;
use num_derive::FromPrimitive;
use std::{collections::HashMap, path::Path, str::FromStr};

/// Represents the internal hostio functions a module may have.
#[derive(Clone, Copy, Debug, FromPrimitive)]
#[repr(u64)]
pub enum InternalFunc {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
    MemoryFill,
    MemoryCopy,
    UserInkLeft,
    UserInkStatus,
    UserSetInk,
    UserStackLeft,
    UserSetStack,
    UserMemorySize,
    CallMain,
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
        #[rustfmt::skip]
        let ty = match self {
            WavmCallerLoad8  | WavmCallerLoad32  => func!([I32], [I32]),
            WavmCallerStore8 | WavmCallerStore32 => func!([I32, I32], []),
            MemoryFill       | MemoryCopy        => func!([I32, I32, I32], []),
            UserInkLeft    => func!([], [I64]),      // λ() → ink_left
            UserInkStatus  => func!([], [I32]),      // λ() → ink_status
            UserSetInk     => func!([I64, I32], []), // λ(ink_left, ink_status)
            UserStackLeft  => func!([], [I32]),      // λ() → stack_left
            UserSetStack   => func!([I32], []),      // λ(stack_left)
            UserMemorySize => func!([], [I32]),      // λ() → memory_size
            CallMain       => func!([I32], [I32]),   // λ(args_len) → status
        };
        ty
    }
}

/// Represents the internal hostio functions a module may have.
pub enum Hostio {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
    WavmGetGlobalStateBytes32,
    WavmSetGlobalStateBytes32,
    WavmGetGlobalStateU64,
    WavmSetGlobalStateU64,
    WavmReadKeccakPreimage,
    WavmReadSha256Preimage,
    WavmReadEthVersionedHashPreimage,
    WavmReadInboxMessage,
    WavmReadDelayedInboxMessage,
    WavmHaltAndSetFinished,
    WavmLinkModule,
    WavmUnlinkModule,
    ProgramInkLeft,
    ProgramInkStatus,
    ProgramSetInk,
    ProgramStackLeft,
    ProgramSetStack,
    ProgramMemorySize,
    ProgramCallMain,
    ProgramRequest,
    ProgramContinue,
    ConsoleLogTxt,
    ConsoleLogI32,
    ConsoleLogI64,
    ConsoleLogF32,
    ConsoleLogF64,
    ConsoleTeeI32,
    ConsoleTeeI64,
    ConsoleTeeF32,
    ConsoleTeeF64,
    UserInkLeft,
    UserInkStatus,
    UserSetInk,
}

impl FromStr for Hostio {
    type Err = ErrReport;

    fn from_str(s: &str) -> Result<Self> {
        let (module, name) = utils::split_import(s)?;

        use Hostio::*;
        Ok(match (module, name) {
            ("env", "wavm_caller_load8") => WavmCallerLoad8,
            ("env", "wavm_caller_load32") => WavmCallerLoad32,
            ("env", "wavm_caller_store8") => WavmCallerStore8,
            ("env", "wavm_caller_store32") => WavmCallerStore32,
            ("env", "wavm_get_globalstate_bytes32") => WavmGetGlobalStateBytes32,
            ("env", "wavm_set_globalstate_bytes32") => WavmSetGlobalStateBytes32,
            ("env", "wavm_get_globalstate_u64") => WavmGetGlobalStateU64,
            ("env", "wavm_set_globalstate_u64") => WavmSetGlobalStateU64,
            ("env", "wavm_read_keccak_256_preimage") => WavmReadKeccakPreimage,
            ("env", "wavm_read_sha2_256_preimage") => WavmReadSha256Preimage,
            ("env", "wavm_read_eth_versioned_hash_preimage") => WavmReadEthVersionedHashPreimage,
            ("env", "wavm_read_inbox_message") => WavmReadInboxMessage,
            ("env", "wavm_read_delayed_inbox_message") => WavmReadDelayedInboxMessage,
            ("env", "wavm_halt_and_set_finished") => WavmHaltAndSetFinished,
            ("hostio", "wavm_link_module") => WavmLinkModule,
            ("hostio", "wavm_unlink_module") => WavmUnlinkModule,
            ("hostio", "program_ink_left") => ProgramInkLeft,
            ("hostio", "program_ink_status") => ProgramInkStatus,
            ("hostio", "program_set_ink") => ProgramSetInk,
            ("hostio", "program_stack_left") => ProgramStackLeft,
            ("hostio", "program_set_stack") => ProgramSetStack,
            ("hostio", "program_memory_size") => ProgramMemorySize,
            ("hostio", "program_call_main") => ProgramCallMain,
            ("hostio", "program_request") => ProgramRequest,
            ("hostio", "program_continue") => ProgramContinue,
            ("hostio", "user_ink_left") => UserInkLeft,
            ("hostio", "user_ink_status") => UserInkStatus,
            ("hostio", "user_set_ink") => UserSetInk,
            ("console", "log_txt") => ConsoleLogTxt,
            ("console", "log_i32") => ConsoleLogI32,
            ("console", "log_i64") => ConsoleLogI64,
            ("console", "log_f32") => ConsoleLogF32,
            ("console", "log_f64") => ConsoleLogF64,
            ("console", "tee_i32") => ConsoleTeeI32,
            ("console", "tee_i64") => ConsoleTeeI64,
            ("console", "tee_f32") => ConsoleTeeF32,
            ("console", "tee_f64") => ConsoleTeeF64,
            _ => bail!("no such hostio {} in {}", name.red(), module.red()),
        })
    }
}

impl Hostio {
    pub fn ty(&self) -> FunctionType {
        use ArbValueType::*;
        use Hostio::*;

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

        #[rustfmt::skip]
        let ty = match self {
            WavmCallerLoad8                  => InternalFunc::WavmCallerLoad8.ty(),
            WavmCallerLoad32                 => InternalFunc::WavmCallerLoad32.ty(),
            WavmCallerStore8                 => InternalFunc::WavmCallerStore8.ty(),
            WavmCallerStore32                => InternalFunc::WavmCallerStore32.ty(),
            WavmGetGlobalStateBytes32        => func!([I32, I32]),
            WavmSetGlobalStateBytes32        => func!([I32, I32]),
            WavmGetGlobalStateU64            => func!([I32], [I64]),
            WavmSetGlobalStateU64            => func!([I32, I64]),
            WavmReadKeccakPreimage           => func!([I32, I32], [I32]),
            WavmReadSha256Preimage           => func!([I32, I32], [I32]),
            WavmReadEthVersionedHashPreimage => func!([I32, I32], [I32]),
            WavmReadInboxMessage             => func!([I64, I32, I32], [I32]),
            WavmReadDelayedInboxMessage      => func!([I64, I32, I32], [I32]),
            WavmHaltAndSetFinished           => func!(),
            WavmLinkModule              => func!([I32], [I32]),      // λ(module_hash) → module
            WavmUnlinkModule            => func!(),                  // λ()
            ProgramInkLeft              => func!([I32], [I64]),      // λ(module) → ink_left
            ProgramInkStatus            => func!([I32], [I32]),      // λ(module) → ink_status
            ProgramSetInk               => func!([I32, I64]),        // λ(module, ink_left)
            ProgramStackLeft            => func!([I32], [I32]),      // λ(module) → stack_left
            ProgramSetStack             => func!([I32, I32]),        // λ(module, stack_left)
            ProgramMemorySize           => func!([I32], [I32]),      // λ(module) → memory_size
            ProgramCallMain             => func!([I32, I32], [I32]), // λ(module, args_len) → status
            ProgramRequest              => func!([I32], [I32]),      // λ(status) → response
            ProgramContinue             => func!([I32], [I32]), // λ(response) → status
            ConsoleLogTxt               => func!([I32, I32]),        // λ(text, len)
            ConsoleLogI32               => func!([I32]),             // λ(value)
            ConsoleLogI64               => func!([I64]),             // λ(value)
            ConsoleLogF32               => func!([F32]),             // λ(value)
            ConsoleLogF64               => func!([F64]),             // λ(value)
            ConsoleTeeI32               => func!([I32], [I32]),      // λ(value) → value
            ConsoleTeeI64               => func!([I64], [I64]),      // λ(value) → value
            ConsoleTeeF32               => func!([F32], [F32]),      // λ(value) → value
            ConsoleTeeF64               => func!([F64], [F64]),      // λ(value) → value
            UserInkLeft                 => InternalFunc::UserInkLeft.ty(),
            UserInkStatus               => InternalFunc::UserInkStatus.ty(),
            UserSetInk                  => InternalFunc::UserSetInk.ty(),
        };
        ty
    }

    pub fn body(&self, _prior: usize) -> Vec<Instruction> {
        let mut body = vec![];

        macro_rules! opcode {
            ($opcode:expr) => {
                body.push(Instruction::simple($opcode))
            };
            ($opcode:expr, $value:expr) => {
                body.push(Instruction::with_data($opcode, $value as u64))
            };
        }
        macro_rules! cross_internal {
            ($func:ident) => {
                opcode!(LocalGet, 0); // module
                opcode!(CrossModuleInternalCall, InternalFunc::$func); // consumes module
            };
        }
        macro_rules! intern {
            ($func:ident) => {
                opcode!(CallerModuleInternalCall, InternalFunc::$func);
            };
        }

        use Hostio::*;
        use Opcode::*;
        match self {
            WavmCallerLoad8 => {
                opcode!(LocalGet, 0);
                intern!(WavmCallerLoad8);
            }
            WavmCallerLoad32 => {
                opcode!(LocalGet, 0);
                intern!(WavmCallerLoad32);
            }
            WavmCallerStore8 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                intern!(WavmCallerStore8);
            }
            WavmCallerStore32 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                intern!(WavmCallerStore32);
            }
            WavmGetGlobalStateBytes32 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(GetGlobalStateBytes32);
            }
            WavmSetGlobalStateBytes32 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(SetGlobalStateBytes32);
            }
            WavmGetGlobalStateU64 => {
                opcode!(LocalGet, 0);
                opcode!(GetGlobalStateU64);
            }
            WavmSetGlobalStateU64 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(SetGlobalStateU64);
            }
            WavmReadKeccakPreimage => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(ReadPreImage, PreimageType::Keccak256);
            }
            WavmReadSha256Preimage => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(ReadPreImage, PreimageType::Sha2_256);
            }
            WavmReadEthVersionedHashPreimage => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(ReadPreImage, PreimageType::EthVersionedHash);
            }
            WavmReadInboxMessage => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(LocalGet, 2);
                opcode!(ReadInboxMessage, InboxIdentifier::Sequencer);
            }
            WavmReadDelayedInboxMessage => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(LocalGet, 2);
                opcode!(ReadInboxMessage, InboxIdentifier::Delayed);
            }
            WavmHaltAndSetFinished => {
                opcode!(HaltAndSetFinished);
            }
            WavmLinkModule => {
                // λ(module_hash) → module
                opcode!(LocalGet, 0);
                opcode!(LinkModule);
                opcode!(NewCoThread);
            }
            WavmUnlinkModule => {
                // λ()
                opcode!(UnlinkModule);
                opcode!(PopCoThread);
            }
            ProgramInkLeft => {
                // λ(module) → ink_left
                cross_internal!(UserInkLeft);
            }
            ProgramInkStatus => {
                // λ(module) → ink_status
                cross_internal!(UserInkStatus);
            }
            ProgramSetInk => {
                // λ(module, ink_left)
                opcode!(LocalGet, 1); // ink_left
                opcode!(I32Const, 0); // ink_status
                cross_internal!(UserSetInk);
            }
            ProgramStackLeft => {
                // λ(module) → stack_left
                cross_internal!(UserStackLeft);
            }
            ProgramSetStack => {
                // λ(module, stack_left)
                opcode!(LocalGet, 1); // stack_left
                cross_internal!(UserSetStack);
            }
            ProgramMemorySize => {
                // λ(module) → memory_size
                cross_internal!(UserMemorySize);
            }
            ProgramCallMain => {
                // caller sees: λ(module, args_len) → status
                opcode!(LocalGet, 1); // args_len
                opcode!(LocalGet, 0); // module
                opcode!(MoveFromStackToInternal);
                opcode!(MoveFromStackToInternal);
                opcode!(SwitchThread, 8);
                opcode!(MoveFromInternalToStack);
                opcode!(MoveFromInternalToStack);
                opcode!(CrossModuleInternalCall, InternalFunc::CallMain); // consumes module
                opcode!(MoveFromStackToInternal);
                opcode!(SwitchThread, 0);
                opcode!(MoveFromInternalToStack);
                opcode!(Return);

                // jumps here if errored while in thread 1
                opcode!(I32Const, UserOutcomeKind::Failure as u32)
            }
            ProgramContinue => {
                // caller sees: λ(return_data) → status (returns to caller of ProgramRequest)
                // code returns return_data to caller of ProgramRequest
                opcode!(LocalGet, 0); // return_data
                opcode!(MoveFromStackToInternal);
                opcode!(SwitchThread, 3);
                opcode!(MoveFromInternalToStack);
                opcode!(Return);

                // jumps here if errored while in cothread
                opcode!(I32Const, UserOutcomeKind::Failure as u32)
            }
            ProgramRequest => {
                // caller sees: λ(status) → response
                // code returns status of either ProgramContinue or ProgramCallMain
                opcode!(LocalGet, 0); // return_data
                opcode!(MoveFromStackToInternal);
                opcode!(SwitchThread, 0);
                opcode!(MoveFromInternalToStack);
            }
            UserInkLeft => {
                // λ() → ink_left
                intern!(UserInkLeft);
            }
            UserInkStatus => {
                // λ() → ink_status
                intern!(UserInkStatus);
            }
            UserSetInk => {
                // λ(ink_left, ink_status)
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                intern!(UserSetInk);
            }
            ConsoleLogTxt | ConsoleLogI32 | ConsoleLogI64 | ConsoleLogF32 | ConsoleLogF64 => {}
            ConsoleTeeI32 | ConsoleTeeI64 | ConsoleTeeF32 | ConsoleTeeF64 => {
                opcode!(LocalGet, 0);
            }
        }
        body
    }
}

pub fn get_impl(module: &str, name: &str) -> Result<(Function, bool)> {
    let hostio: Hostio = format!("{module}__{name}").parse()?;

    let append = |code: &mut Vec<Instruction>| {
        let len = code.len();
        code.extend(hostio.body(len));
        Ok(())
    };

    let debug = module == "console" || module == "debug";
    Function::new(&[], append, hostio.ty(), &[]).map(|x| (x, debug))
}

/// Adds internal functions to a module.
/// Note: the order of the functions must match that of the `InternalFunc` enum
pub fn new_internal_funcs(stylus_data: Option<StylusData>) -> Vec<Function> {
    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;

    fn code_func(code: &[Instruction], func: InternalFunc) -> Function {
        let mut wavm = vec![Instruction::simple(InitFrame)];
        wavm.extend(code);
        wavm.push(Instruction::simple(Return));
        Function::new_from_wavm(wavm, func.ty(), vec![])
    }

    fn op_func(opcode: Opcode, func: InternalFunc) -> Function {
        code_func(&[Instruction::simple(opcode)], func)
    }

    let mut funcs = vec![];
    let mut add_func = |func, internal| {
        assert_eq!(funcs.len(), internal as usize);
        funcs.push(func)
    };
    let mut add_op_func = |opcode, internal| add_func(op_func(opcode, internal), internal);

    // order matters!
    add_op_func(
        MemoryLoad {
            ty: I32,
            bytes: 1,
            signed: false,
        },
        WavmCallerLoad8,
    );
    add_op_func(
        MemoryLoad {
            ty: I32,
            bytes: 4,
            signed: false,
        },
        WavmCallerLoad32,
    );
    add_op_func(MemoryStore { ty: I32, bytes: 1 }, WavmCallerStore8);
    add_op_func(MemoryStore { ty: I32, bytes: 4 }, WavmCallerStore32);

    let [memory_fill, memory_copy] = (*BULK_MEMORY_FUNCS).clone();
    add_func(memory_fill, MemoryFill);
    add_func(memory_copy, MemoryCopy);

    let mut add_func = |code: &[_], internal| add_func(code_func(code, internal), internal);

    if let Some(info) = stylus_data {
        let (gas, status, depth) = info.global_offsets();
        let main = info.user_main.into();
        let inst = |op, data| Instruction::with_data(op, data);

        add_func(&[inst(GlobalGet, gas)], UserInkLeft);
        add_func(&[inst(GlobalGet, status)], UserInkStatus);
        add_func(&[inst(GlobalSet, status), inst(GlobalSet, gas)], UserSetInk);
        add_func(&[inst(GlobalGet, depth)], UserStackLeft);
        add_func(&[inst(GlobalSet, depth)], UserSetStack);
        add_func(&[inst(MemorySize, 0)], UserMemorySize);
        add_func(
            &[
                inst(Call, main),
                // force return value to boolean
                Instruction::simple(I32Eqz),
                Instruction::simple(I32Eqz),
            ],
            CallMain,
        );
    }
    funcs
}

lazy_static! {
    static ref BULK_MEMORY_FUNCS: [Function; 2] = {
        use host::InternalFunc::*;

        let data = include_bytes!("bulk_memory.wat");
        let wasm = wat::parse_bytes(data).expect("failed to parse bulk_memory.wat");
        let bin = binary::parse(&wasm, Path::new("internal")).expect("failed to parse bulk_memory.wasm");
        let types = [MemoryFill.ty(), MemoryCopy.ty()];
        let names = ["memory_fill", "memory_copy"];

        [0, 1].map(|i| {
            let code = &bin.codes[i];
            let name = bin.names.functions.get(&(i as u32)).unwrap();
            let ty = &bin.types[bin.functions[i] as usize];
            assert_eq!(ty, &types[i]);
            assert_eq!(name, names[i]);

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
                    &bin.names.module,
                ),
                ty.clone(),
                &[] // impls don't make calls
            );
            func.expect("failed to create bulk memory func")
        })
    };
}
