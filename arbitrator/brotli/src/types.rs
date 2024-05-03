// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(dead_code)]

use num_enum::{IntoPrimitive, TryFromPrimitive};

/// The default window size used during compression.
pub const DEFAULT_WINDOW_SIZE: u32 = 22;

/// Represents the outcome of a brotli operation.
#[derive(Debug, PartialEq, IntoPrimitive, TryFromPrimitive)]
#[repr(u32)]
pub enum BrotliStatus {
    Failure,
    Success,
    NeedsMoreInput,
    NeedsMoreOutput,
}

impl BrotliStatus {
    /// Whether the outcome of the operation was successful.
    pub fn is_ok(&self) -> bool {
        self == &Self::Success
    }

    /// Whether the outcome of the operation was an error of any kind.
    pub fn is_err(&self) -> bool {
        !self.is_ok()
    }
}

/// A portable `bool`.
#[derive(PartialEq)]
#[repr(usize)]
pub(super) enum BrotliBool {
    False,
    True,
}

impl BrotliBool {
    /// Whether the type is `True`. This function exists since the API conflates `BrotliBool` and `BrotliStatus` at times.
    pub fn is_ok(&self) -> bool {
        self == &Self::True
    }

    /// Whether the type is `False`. This function exists since the API conflates `BrotliBool` and `BrotliStatus` at times.
    pub fn is_err(&self) -> bool {
        !self.is_ok()
    }
}

/// The dictionary policy.
#[repr(C)]
pub(super) enum BrotliEncoderMode {
    /// Start with an empty dictionary.
    Generic,
    /// Use the pre-built dictionary for text.
    Text,
    /// Use the pre-built dictionary for fonts.
    Font,
}

/// Configuration options for brotli compression.
#[repr(C)]
pub(super) enum BrotliEncoderParameter {
    /// The dictionary policy.
    Mode,
    /// The brotli level. Ranges from 0 to 11.
    Quality,
    /// The size of the window. Defaults to 22.
    WindowSize,
    BlockSize,
    DisableContextModeling,
    SizeHint,
    LargeWindowMode,
    PostfixBits,
    DirectDistanceCodes,
    StreamOffset,
}

/// Streaming operations for use when encoding.
#[repr(C)]
pub(super) enum BrotliEncoderOperation {
    /// Produce as much output as possible.
    Process,
    /// Flush the contents of the encoder.
    Flush,
    /// Flush and finalize the contents of the encoder.
    Finish,
    /// Emit metadata info.
    Metadata,
}

/// Type of custom dictionary.
#[repr(C)]
pub(super) enum BrotliSharedDictionaryType {
    /// LZ77 prefix dictionary
    Raw,
    /// Serialized dictionary
    Serialized,
}
