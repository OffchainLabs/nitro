// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

// Warning: this code is for testing only and has not been audited

use alloc::{string::String, vec::Vec};
use core::marker::PhantomData;
use stylus_sdk::{
    alloy_primitives::{Address, U256},
    alloy_sol_types::{sol, SolError},
    evm, msg,
    prelude::*,
};

pub trait Erc20Params {
    const NAME: &'static str;
    const SYMBOL: &'static str;
    const DECIMALS: u8;
}

sol_storage! {
    /// Erc20 implements all ERC-20 methods.
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

pub enum Erc20Error {
    InsufficientBalance(InsufficientBalance),
    InsufficientAllowance(InsufficientAllowance),
}

// We will soon provide a #[derive(SolidityError)] to clean this up
impl From<Erc20Error> for Vec<u8> {
    fn from(err: Erc20Error) -> Vec<u8> {
        match err {
            Erc20Error::InsufficientBalance(e) => e.encode(),
            Erc20Error::InsufficientAllowance(e) => e.encode(),
        }
    }
}

// These methods aren't exposed to other contracts
// Note: modifying storage will become much prettier soon
impl<T: Erc20Params> Erc20<T> {
    pub fn transfer_impl(
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
        evm::log(Transfer { from, to, value });
        Ok(())
    }

    pub fn mint(&mut self, address: Address, value: U256) {
        let mut balance = self.balances.setter(address);
        let new_balance = balance.get() + value;
        balance.set(new_balance);
        self.total_supply.set(self.total_supply.get() + value);
        evm::log(Transfer {
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
        evm::log(Transfer {
            from: address,
            to: Address::ZERO,
            value,
        });
        Ok(())
    }
}

// These methods are external to other contracts
// Note: modifying storage will become much prettier soon
#[external]
impl<T: Erc20Params> Erc20<T> {
    pub fn name() -> Result<String, Erc20Error> {
        Ok(T::NAME.into())
    }

    pub fn symbol() -> Result<String, Erc20Error> {
        Ok(T::SYMBOL.into())
    }

    pub fn decimals() -> Result<u8, Erc20Error> {
        Ok(T::DECIMALS)
    }

    pub fn balance_of(&self, address: Address) -> Result<U256, Erc20Error> {
        Ok(self.balances.get(address))
    }

    pub fn transfer(&mut self, to: Address, value: U256) -> Result<bool, Erc20Error> {
        self.transfer_impl(msg::sender(), to, value)?;
        Ok(true)
    }

    pub fn approve(&mut self, spender: Address, value: U256) -> Result<bool, Erc20Error> {
        self.allowances.setter(msg::sender()).insert(spender, value);
        evm::log(Approval {
            owner: msg::sender(),
            spender,
            value,
        });
        Ok(true)
    }

    pub fn transfer_from(
        &mut self,
        from: Address,
        to: Address,
        value: U256,
    ) -> Result<bool, Erc20Error> {
        let mut sender_allowances = self.allowances.setter(from);
        let mut allowance = sender_allowances.setter(msg::sender());
        let old_allowance = allowance.get();
        if old_allowance < value {
            return Err(Erc20Error::InsufficientAllowance(InsufficientAllowance {
                owner: from,
                spender: msg::sender(),
                have: old_allowance,
                want: value,
            }));
        }
        allowance.set(old_allowance - value);
        self.transfer_impl(from, to, value)?;
        Ok(true)
    }

    pub fn allowance(&self, owner: Address, spender: Address) -> Result<U256, Erc20Error> {
        Ok(self.allowances.getter(owner).get(spender))
    }
}
