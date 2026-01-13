// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::evm::api::Ink;
use std::time::{Duration, Instant};

// Benchmark is used to track the performance of blocks of code in stylus
#[derive(Clone, Copy, Debug, Default)]
pub struct Benchmark {
    pub timer: Option<Instant>,
    pub elapsed_total: Duration,
    pub ink_start: Option<Ink>,
    pub ink_total: Ink,
}
