// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::{Escape, WasmEnvMut};

pub fn proc_exit(mut _env: WasmEnvMut, code: u32) -> Result<(), Escape> {
    Err(Escape::Exit(code))
}
