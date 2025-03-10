// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::Rng;
use strum_macros::{Display, EnumString};

#[derive(Debug, Display, EnumString)]
#[strum(serialize_all = "lowercase")]
pub enum DataType {
    I32,
    I64,
}

pub trait Rand {
    fn gen(&self) -> usize;
}

impl Rand for DataType {
    fn gen(&self) -> usize {
        let mut rng = rand::thread_rng();
        match self {
            DataType::I32 => rng.gen_range(0..i32::MAX).try_into().unwrap(),
            DataType::I64 => rng.gen_range(0..i64::MAX).try_into().unwrap(),
        }
    }
}
