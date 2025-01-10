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
            // makes sure that the generated number fits a signed integer
            DataType::I32 => (rng.gen::<u32>() / 2 - 1).try_into().unwrap(),
            DataType::I64 => (rng.gen::<u64>() / 2 - 1).try_into().unwrap(),
        }
    }
}
