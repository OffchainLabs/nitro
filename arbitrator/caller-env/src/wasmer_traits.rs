// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{brotli::Dictionary, BrotliStatus, Errno, GuestPtr};
use wasmer::{FromToNativeWasmType, WasmPtr};

unsafe impl FromToNativeWasmType for GuestPtr {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self(u32::from_native(native))
    }

    fn to_native(self) -> i32 {
        self.0.to_native()
    }
}

unsafe impl FromToNativeWasmType for Errno {
    type Native = i32;

    fn from_native(native: i32) -> Self {
        Self(u16::from_native(native))
    }

    fn to_native(self) -> i32 {
        self.0.to_native()
    }
}

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

impl<T> From<GuestPtr> for WasmPtr<T> {
    fn from(value: GuestPtr) -> Self {
        WasmPtr::new(value.0)
    }
}
