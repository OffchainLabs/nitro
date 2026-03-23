// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use eyre::ErrReport;
use wasmer_types::Pages;

pub struct MemoryType {
    pub min: Pages,
    pub max: Option<Pages>,
}

impl MemoryType {
    pub fn new(min: Pages, max: Option<Pages>) -> Self {
        Self { min, max }
    }
}

impl From<&wasmer_types::MemoryType> for MemoryType {
    fn from(value: &wasmer_types::MemoryType) -> Self {
        Self::new(value.minimum, value.maximum)
    }
}

impl TryFrom<&wasmparser::MemoryType> for MemoryType {
    type Error = ErrReport;

    fn try_from(value: &wasmparser::MemoryType) -> std::result::Result<Self, Self::Error> {
        Ok(Self {
            min: Pages(value.initial.try_into()?),
            max: value.maximum.map(|x| x.try_into()).transpose()?.map(Pages),
        })
    }
}
