// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Warning: this code is for testing only and has not been audited

use alloc::{string::String, vec, vec::Vec};
use core::marker::PhantomData;
use stylus_sdk::{
    alloy_primitives::{Address, U256},
    alloy_sol_types::sol,
    prelude::*,
};

pub trait Erc20Params {
    const NAME: &'static str;
    const SYMBOL: &'static str;
    const DECIMALS: u8;
}

sol_storage! {
    /// Erc20 provides ERC-20 storage layout and internal helpers.
    pub struct Erc20<T> {
        /// Maps users to balances
        mapping(address => uint256) balances;
        /// Maps users to a mapping of each spender's allowance
        mapping(address => mapping(address => uint256)) allowances;
        /// The total supply of the token
        uint256 total_supply;
        /// Used to allow [`Erc20Params`]
        PhantomData<T> phantom;
    }
}

// Declare events and Solidity error types
sol! {
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    error InsufficientBalance(address from, uint256 have, uint256 want);
    error InsufficientAllowance(address owner, address spender, uint256 have, uint256 want);
}

#[derive(SolidityError)]
pub enum Erc20Error {
    InsufficientBalance(InsufficientBalance),
    InsufficientAllowance(InsufficientAllowance),
}

/// Trait defining the public ERC-20 interface.
#[public]
pub trait IErc20 {
    fn name(&self) -> String;
    fn symbol(&self) -> String;
    fn decimals(&self) -> u8;
    fn balance_of(&self, address: Address) -> U256;
    fn transfer(&mut self, to: Address, value: U256) -> Result<bool, Erc20Error>;
    fn approve(&mut self, spender: Address, value: U256) -> bool;
    fn transfer_from(
        &mut self,
        from: Address,
        to: Address,
        value: U256,
    ) -> Result<bool, Erc20Error>;
    fn allowance(&self, owner: Address, spender: Address) -> U256;
}

impl<T: Erc20Params> Erc20<T> {
    pub fn name() -> String {
        T::NAME.into()
    }

    pub fn symbol() -> String {
        T::SYMBOL.into()
    }

    pub fn decimals() -> u8 {
        T::DECIMALS
    }

    pub fn balance_of(&self, owner: Address) -> U256 {
        self.balances.get(owner)
    }

    pub fn transfer(&mut self, to: Address, value: U256) -> Result<bool, Erc20Error> {
        self.transfer_impl(self.vm().msg_sender(), to, value)?;
        Ok(true)
    }

    pub fn approve(&mut self, spender: Address, value: U256) -> bool {
        let msg_sender = self.vm().msg_sender();
        self.allowances.setter(msg_sender).insert(spender, value);
        self.vm().log(Approval {
            owner: msg_sender,
            spender,
            value,
        });
        true
    }

    pub fn transfer_from(
        &mut self,
        from: Address,
        to: Address,
        value: U256,
    ) -> Result<bool, Erc20Error> {
        let msg_sender = self.vm().msg_sender();
        let mut sender_allowances = self.allowances.setter(from);
        let mut allowance = sender_allowances.setter(msg_sender);
        let old_allowance = allowance.get();
        if old_allowance < value {
            return Err(Erc20Error::InsufficientAllowance(InsufficientAllowance {
                owner: from,
                spender: msg_sender,
                have: old_allowance,
                want: value,
            }));
        }
        allowance.set(old_allowance - value);
        self.transfer_impl(from, to, value)?;
        Ok(true)
    }

    pub fn allowance(&self, owner: Address, spender: Address) -> U256 {
        self.allowances.getter(owner).get(spender)
    }

    fn transfer_impl(
        &mut self,
        from: Address,
        to: Address,
        value: U256,
    ) -> Result<(), Erc20Error> {
        let mut sender_balance = self.balances.setter(from);
        let old_sender_balance = sender_balance.get();
        if old_sender_balance < value {
            return Err(Erc20Error::InsufficientBalance(InsufficientBalance {
                from,
                have: old_sender_balance,
                want: value,
            }));
        }
        sender_balance.set(old_sender_balance - value);
        let mut to_balance = self.balances.setter(to);
        let new_to_balance = to_balance.get() + value;
        to_balance.set(new_to_balance);
        self.vm().log(Transfer { from, to, value });
        Ok(())
    }

    pub fn mint(&mut self, address: Address, value: U256) {
        let mut balance = self.balances.setter(address);
        let new_balance = balance.get() + value;
        balance.set(new_balance);
        self.total_supply.set(self.total_supply.get() + value);
        self.vm().log(Transfer {
            from: Address::ZERO,
            to: address,
            value,
        });
    }

    pub fn burn(&mut self, address: Address, value: U256) -> Result<(), Erc20Error> {
        let mut balance = self.balances.setter(address);
        let old_balance = balance.get();
        if old_balance < value {
            return Err(Erc20Error::InsufficientBalance(InsufficientBalance {
                from: address,
                have: old_balance,
                want: value,
            }));
        }
        balance.set(old_balance - value);
        self.total_supply.set(self.total_supply.get() - value);
        self.vm().log(Transfer {
            from: address,
            to: Address::ZERO,
            value,
        });
        Ok(())
    }
}
