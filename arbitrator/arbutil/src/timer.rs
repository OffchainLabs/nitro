// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::time::{Duration, Instant};
use derivative::Derivative;

#[derive(Derivative, Clone, Copy)]
#[derivative(Debug)]
pub struct Timer {
    pub instant: Option<Instant>,
    pub elapsed: Option<Duration>,
    pub cycles_start: Option<u64>,
    pub cycles_total: Option<u64>,
}
