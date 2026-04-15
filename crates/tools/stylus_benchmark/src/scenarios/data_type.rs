// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use rand::RngExt;
use strum_macros::{Display, EnumString};

#[derive(Debug, Display, EnumString)]
#[strum(serialize_all = "lowercase")]
pub enum DataType {
    I32,
    I64,
}

pub trait Rand {
    fn r#gen(&self) -> usize;
}

impl Rand for DataType {
    fn r#gen(&self) -> usize {
        let mut rng = rand::rng();
        match self {
            DataType::I32 => rng.random_range(0..i32::MAX).try_into().unwrap(),
            DataType::I64 => rng.random_range(0..i64::MAX).try_into().unwrap(),
        }
    }
}
