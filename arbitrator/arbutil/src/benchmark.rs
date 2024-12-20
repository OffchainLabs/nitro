// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::evm::api::Ink;
use derivative::Derivative;
use std::time::{Duration, Instant};

// Benchmark is used to track the performance of blocks of code in stylus
#[derive(Derivative, Clone, Copy, Default)]
#[derivative(Debug)]
pub struct Benchmark {
    pub timer: Option<Instant>,
    pub elapsed_total: Duration,
    pub ink_start: Option<Ink>,
    pub ink_total: Ink,
}
