// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use std::io;

mod markers;
mod primitives;
mod receiver;
mod sender;
#[cfg(test)]
mod tests;

pub use receiver::*;
pub use sender::*;

pub type IOResult<T> = Result<T, io::Error>;
