//! Minimal ERC-20 for cross-language fuzz parity.
//!
//! Exposes:
//!   `mint(address,uint256)        -> ()`           storage write + log
//!   `transfer(address,uint256)    -> bool`         storage rw + log + msg_sender
//!   `balance_of(address)          -> uint256`      storage read
//!   `total_supply()               -> uint256`      storage read
//!
//! Storage layout uses direct slots to avoid heavy sol_storage codegen.
//! Slot 0: total supply (U256)
//! Slot keccak(address ++ 1): balanceOf(address)
//!
//! Rebuild: `cargo stylus get-initcode --output ../../prebuilt/erc20_mini.hex`.

#![cfg_attr(not(any(test, feature = "export-abi")), no_main)]
extern crate alloc;

use alloy_sol_types::{sol, SolEvent};
use stylus_sdk::{
    alloy_primitives::{Address, B256, U256},
    crypto::keccak,
    prelude::*,
};

sol! {
    event Transfer(address indexed from, address indexed to, uint256 value);
}

#[storage]
#[entrypoint]
pub struct Erc20Mini {}

const TOTAL_SUPPLY_SLOT: U256 = U256::ZERO;
const BALANCES_SUBSPACE: u8 = 1;

fn balance_slot(account: Address) -> U256 {
    let mut buf = [0u8; 52];
    buf[..20].copy_from_slice(account.as_slice());
    buf[51] = BALANCES_SUBSPACE;
    U256::from_be_bytes::<32>(keccak(buf).0)
}

#[public]
impl Erc20Mini {
    pub fn mint(&mut self, to: Address, amount: U256) {
        let slot = balance_slot(to);
        let prev = self.vm().storage_load_bytes32(slot);
        let new_bal = U256::from_be_bytes(prev.0).saturating_add(amount);
        unsafe {
            self.vm()
                .storage_cache_bytes32(slot, B256::from(new_bal.to_be_bytes::<32>()));
        }
        let supply = self.vm().storage_load_bytes32(TOTAL_SUPPLY_SLOT);
        let new_supply = U256::from_be_bytes(supply.0).saturating_add(amount);
        unsafe {
            self.vm().storage_cache_bytes32(
                TOTAL_SUPPLY_SLOT,
                B256::from(new_supply.to_be_bytes::<32>()),
            );
        }
        self.vm().flush_cache(false);
        emit_transfer(self.vm(), Address::ZERO, to, amount);
    }

    pub fn transfer(&mut self, to: Address, amount: U256) -> bool {
        let from = self.vm().msg_sender();
        let from_slot = balance_slot(from);
        let from_prev = U256::from_be_bytes(self.vm().storage_load_bytes32(from_slot).0);
        if from_prev < amount {
            return false;
        }
        unsafe {
            self.vm().storage_cache_bytes32(
                from_slot,
                B256::from((from_prev - amount).to_be_bytes::<32>()),
            );
        }
        let to_slot = balance_slot(to);
        let to_prev = U256::from_be_bytes(self.vm().storage_load_bytes32(to_slot).0);
        unsafe {
            self.vm().storage_cache_bytes32(
                to_slot,
                B256::from((to_prev + amount).to_be_bytes::<32>()),
            );
        }
        self.vm().flush_cache(false);
        emit_transfer(self.vm(), from, to, amount);
        true
    }

    pub fn balance_of(&self, who: Address) -> U256 {
        let v = self.vm().storage_load_bytes32(balance_slot(who));
        U256::from_be_bytes(v.0)
    }

    pub fn total_supply(&self) -> U256 {
        let v = self.vm().storage_load_bytes32(TOTAL_SUPPLY_SLOT);
        U256::from_be_bytes(v.0)
    }
}

fn emit_transfer<V: stylus_sdk::prelude::Host>(vm: &V, from: Address, to: Address, value: U256) {
    let evt = Transfer { from, to, value };
    let data = evt.encode_data();
    let topics = evt.encode_topics();
    let mut topic_bytes = Vec::with_capacity(topics.len() * 32);
    for t in &topics {
        topic_bytes.extend_from_slice(t.as_slice());
    }
    let mut buf = Vec::with_capacity(topic_bytes.len() + data.len());
    buf.extend_from_slice(&topic_bytes);
    buf.extend_from_slice(&data);
    vm.emit_log(&buf, topics.len());
}
