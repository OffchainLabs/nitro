// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use super::{ArbValueType, FunctionType};
use num_derive::FromPrimitive;

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
