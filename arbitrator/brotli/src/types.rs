// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(dead_code, clippy::len_without_is_empty)]

use num_enum::{IntoPrimitive, TryFromPrimitive};

pub const BROTLI_MODE_GENERIC: u32 = 0;
pub const DEFAULT_WINDOW_SIZE: u32 = 22;

#[derive(Debug, PartialEq, IntoPrimitive, TryFromPrimitive)]
#[repr(u32)]
pub enum BrotliStatus {
    Failure,
    Success,
    NeedsMoreInput,
    NeedsMoreOutput,
}

impl BrotliStatus {
    pub fn is_ok(&self) -> bool {
        self == &Self::Success
    }

    pub fn is_err(&self) -> bool {
        !self.is_ok()
    }
}

#[derive(PartialEq)]
#[repr(usize)]
pub(super) enum BrotliBool {
    False,
    True,
}

#[repr(C)]
pub(super) enum BrotliSharedDictionaryType {
    /// LZ77 prefix dictionary
    Raw,
    /// Serialized dictionary
    Serialized,
}

#[derive(Clone, Copy, PartialEq, IntoPrimitive, TryFromPrimitive)]
#[repr(u32)]
pub enum Dictionary {
    Empty,
    StylusProgram,
}

impl Dictionary {
    pub fn len(&self) -> usize {
        match self {
            Self::Empty => 0,
            Self::StylusProgram => todo!(),
        }
    }

    pub fn data(&self) -> *const u8 {
        match self {
            Self::Empty => [].as_ptr(),
            Self::StylusProgram => todo!(),
        }
    }
}
