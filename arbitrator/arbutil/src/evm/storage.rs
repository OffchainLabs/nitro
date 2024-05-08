// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::Bytes32;
use fnv::FnvHashMap as HashMap;
use std::ops::{Deref, DerefMut};

/// Represents the EVM word at a given key.
#[derive(Debug)]
pub struct StorageWord {
    /// The current value of the slot.
    pub value: Bytes32,
    /// The value in Geth, if known.
    pub known: Option<Bytes32>,
}

impl StorageWord {
    pub fn known(value: Bytes32) -> Self {
        let known = Some(value);
        Self { value, known }
    }

    pub fn unknown(value: Bytes32) -> Self {
        Self { value, known: None }
    }

    pub fn dirty(&self) -> bool {
        Some(self.value) != self.known
    }
}

#[derive(Default)]
pub struct StorageCache {
    pub(crate) slots: HashMap<Bytes32, StorageWord>,
    reads: usize,
    writes: usize,
}

impl StorageCache {
    pub const REQUIRED_ACCESS_GAS: u64 = 10;

    pub fn read_gas(&mut self) -> u64 {
        self.reads += 1;
        match self.reads {
            0..=32 => 0,
            33..=128 => 2,
            _ => 10,
        }
    }

    pub fn write_gas(&mut self) -> u64 {
        self.writes += 1;
        match self.writes {
            0..=8 => 0,
            9..=64 => 7,
            _ => 10,
        }
    }
}

impl Deref for StorageCache {
    type Target = HashMap<Bytes32, StorageWord>;

    fn deref(&self) -> &Self::Target {
        &self.slots
    }
}

impl DerefMut for StorageCache {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.slots
    }
}
