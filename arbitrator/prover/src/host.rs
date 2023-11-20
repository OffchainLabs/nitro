// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::vec_init_then_push)]

use crate::{
    binary, host,
    machine::{Function, InboxIdentifier},
    utils,
    value::{ArbValueType, FunctionType},
    wavm::{wasm_to_wavm, Instruction, Opcode},
};
use arbutil::{Color, PreimageType};
use eyre::{bail, ErrReport, Result};
use lazy_static::lazy_static;
use std::{collections::HashMap, str::FromStr};

/// Represents the internal hostio functions a module may have.
#[derive(Clone, Copy)]
#[repr(u64)]
pub enum InternalFunc {
    WavmCallerLoad8,
    WavmCallerLoad32,
    WavmCallerStore8,
    WavmCallerStore32,
    MemoryFill,
    MemoryCopy,
}

impl InternalFunc {
    pub fn ty(&self) -> FunctionType {
        use ArbValueType::*;
        use InternalFunc::*;
        match self {
            WavmCallerLoad8 | WavmCallerLoad32 => FunctionType::new(vec![I32], vec![I32]),
            WavmCallerStore8 | WavmCallerStore32 => FunctionType::new(vec![I32, I32], vec![]),
            MemoryFill | MemoryCopy => FunctionType::new(vec![I32, I32, I32], vec![]),
        }
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
    WavmReadInboxMessage,
    WavmReadHotShotHeader,
    WavmReadDelayedInboxMessage,
    WavmHaltAndSetFinished,
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
            ("env", "wavm_read_inbox_message") => WavmReadInboxMessage,
            ("env", "wavm_read_delayed_inbox_message") => WavmReadDelayedInboxMessage,
            ("env", "wavm_read_hotshot_header") => WavmReadHotShotHeader,
            ("env", "wavm_halt_and_set_finished") => WavmHaltAndSetFinished,
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
            WavmCallerLoad8             => InternalFunc::WavmCallerLoad8.ty(),
            WavmCallerLoad32            => InternalFunc::WavmCallerLoad32.ty(),
            WavmCallerStore8            => InternalFunc::WavmCallerStore8.ty(),
            WavmCallerStore32           => InternalFunc::WavmCallerStore32.ty(),
            WavmGetGlobalStateBytes32   => func!([I32, I32]),
            WavmSetGlobalStateBytes32   => func!([I32, I32]),
            WavmGetGlobalStateU64       => func!([I32], [I64]),
            WavmSetGlobalStateU64       => func!([I32, I64]),
            WavmReadHotShotHeader => func!([I32, I32]),
            WavmReadKeccakPreimage      => func!([I32, I32], [I32]),
            WavmReadSha256Preimage      => func!([I32, I32], [I32]),
            WavmReadInboxMessage        => func!([I64, I32, I32], [I32]),
            WavmReadDelayedInboxMessage => func!([I64, I32, I32], [I32]),
            WavmHaltAndSetFinished      => func!(),
        };
        ty
    }

    pub fn body(&self) -> Vec<Instruction> {
        let mut body = vec![];

        macro_rules! opcode {
            ($opcode:expr) => {
                body.push(Instruction::simple($opcode))
            };
            ($opcode:expr, $value:expr) => {
                body.push(Instruction::with_data($opcode, $value as u64))
            };
        }

        use Hostio::*;
        use Opcode::*;
        match self {
            WavmCallerLoad8 => {
                opcode!(LocalGet, 0);
                opcode!(CallerModuleInternalCall, InternalFunc::WavmCallerLoad8);
            }
            WavmCallerLoad32 => {
                opcode!(LocalGet, 0);
                opcode!(CallerModuleInternalCall, InternalFunc::WavmCallerLoad32);
            }
            WavmCallerStore8 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(CallerModuleInternalCall, InternalFunc::WavmCallerStore8);
            }
            WavmCallerStore32 => {
                opcode!(LocalGet, 0);
                opcode!(LocalGet, 1);
                opcode!(CallerModuleInternalCall, InternalFunc::WavmCallerStore32);
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
            WavmReadHotShotHeader => {
                unimplemented!()
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
        }
        body
    }
}

pub fn get_impl(module: &str, name: &str) -> Result<Function> {
    let hostio: Hostio = format!("{module}__{name}").parse()?;

    let append = |code: &mut Vec<Instruction>| {
        code.extend(hostio.body());
        Ok(())
    };
    Function::new(&[], append, hostio.ty(), &[])
}

/// Adds internal functions to a module.
/// Note: the order of the functions must match that of the `InternalFunc` enum
pub fn new_internal_funcs() -> Vec<Function> {
    use ArbValueType::*;
    use InternalFunc::*;
    use Opcode::*;

    fn code_func(code: Vec<Instruction>, ty: FunctionType) -> Function {
        let mut wavm = vec![Instruction::simple(InitFrame)];
        wavm.extend(code);
        wavm.push(Instruction::simple(Return));
        Function::new_from_wavm(wavm, ty, vec![])
    }

    fn op_func(opcode: Opcode, func: InternalFunc) -> Function {
        code_func(vec![Instruction::simple(opcode)], func.ty())
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
    funcs
}

lazy_static! {
    static ref BULK_MEMORY_FUNCS: [Function; 2] = {
        use host::InternalFunc::*;

        let data = include_bytes!("bulk_memory.wat");
        let wasm = wat::parse_bytes(data).expect("failed to parse bulk_memory.wat");
        let bin = binary::parse(&wasm).expect("failed to parse bulk_memory.wasm");
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
                ),
                ty.clone(),
                &[] // impls don't make calls
            );
            func.expect("failed to create bulk memory func")
        })
    };
}
