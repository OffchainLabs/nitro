// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::types::{BrotliStatus, Dictionary};
use wasmer::FromToNativeWasmType;

unsafe impl FromToNativeWasmType for BrotliStatus {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self::try_from(u32::from_native(native)).expect("unknown brotli status")
    }

    fn to_native(self) -> i32 {
        (self as u32).to_native()
    }
}

unsafe impl FromToNativeWasmType for Dictionary {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self::try_from(u32::from_native(native)).expect("unknown brotli dictionary")
    }

    fn to_native(self) -> i32 {
        (self as u32).to_native()
    }
}
