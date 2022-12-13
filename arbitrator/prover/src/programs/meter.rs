// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use wasmer::{Instance, Store};

use super::GlobalMod;

#[derive(Debug, PartialEq)]
pub enum MachineMeter {
    Ready(u64),
    Exhausted,
}

impl Into<u64> for MachineMeter {
    fn into(self) -> u64 {
        match self {
            Self::Ready(gas) => gas,
            Self::Exhausted => 0,
        }
    }
}

pub trait MeteredMachine {
    fn gas_left(&self, store: &mut Store) -> MachineMeter;
    fn set_gas(&mut self, store: &mut Store, gas: u64);
}

impl MeteredMachine for Instance {
    fn gas_left(&self, store: &mut Store) -> MachineMeter {
        let gas = self.get_global(store, "polyglot_gas_left");
        let status = self.get_global(store, "polyglot_gas_status");
        match status {
            0 => MachineMeter::Ready(gas),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, store: &mut Store, gas: u64) {
        self.set_global(store, "polyglot_gas_left", gas);
        self.set_global(store, "polyglot_gas_status", 0);
    }
}
