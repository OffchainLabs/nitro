// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Warning: this code is for testing only and has not been audited

#![cfg_attr(not(feature = "export-abi"), no_main, no_std)]
extern crate alloc;

use crate::erc20::{Erc20, Erc20Error, Erc20Params, IErc20};
use alloc::{string::String, vec, vec::Vec};
use stylus_sdk::{alloy_primitives::{Address, U256}, call::transfer::transfer_eth, prelude::*};

mod erc20;

struct WethParams;

/// Immutable definitions
impl Erc20Params for WethParams {
    const NAME: &'static str = "Wrapped Ether Example";
    const SYMBOL: &'static str = "WETH";
    const DECIMALS: u8 = 18;
}

// The contract
sol_storage! {
    #[entrypoint]
    struct Weth {
        #[borrow] // Allows erc20 to access Weth's storage and make calls
        Erc20<WethParams> erc20;
    }
}

// Another contract we'd like to call
sol_interface! {
    interface IMath {
        function sumValues(uint256[] values) external pure returns (uint256);
    }
}

#[public]
#[implements(IErc20)]
impl Weth {
    #[payable]
    pub fn mint(&mut self) -> Result<(), Vec<u8>> {
        self.erc20
            .mint(self.vm().msg_sender(), self.vm().msg_value());
        Ok(())
    }

    pub fn burn(&mut self, amount: U256) -> Result<(), Vec<u8>> {
        self.erc20.burn(self.vm().msg_sender(), amount)?;

        // send the user their funds
        transfer_eth(self.vm(), self.vm().msg_sender(), amount)
    }

    // sums numbers
    pub fn sum_values(values: Vec<U256>) -> Result<U256, Vec<u8>> {
        Ok(values.iter().sum())
    }

    // calls the sum_values() method from the interface
    pub fn sum_with_helper(&self, helper: IMath, values: Vec<U256>) -> Result<U256, Vec<u8>> {
        let sum = helper.sum_values(self.vm(), Call::new(), values)?;
        Ok(sum)
    }
}

#[public]
impl IErc20 for Weth {
    fn name(&self) -> Result<String, Erc20Error> {
        Ok(Erc20::<WethParams>::name())
    }

    fn symbol(&self) -> Result<String, Erc20Error> {
        Ok(Erc20::<WethParams>::symbol())
    }

    fn decimals(&self) -> Result<u8, Erc20Error> {
        Ok(Erc20::<WethParams>::decimals())
    }

    fn balance_of(&self, address: Address) -> Result<U256, Erc20Error> {
        Ok(self.erc20.balances.get(address))
    }

    fn transfer(&mut self, to: Address, value: U256) -> Result<bool, Erc20Error> {
        self.erc20.transfer_impl(self.vm().msg_sender(), to, value)?;
        Ok(true)
    }

    fn approve(&mut self, spender: Address, value: U256) -> Result<bool, Erc20Error> {
        self.erc20.allowances
            .setter(self.vm().msg_sender())
            .insert(spender, value);
        self.vm().log(erc20::Approval {
            owner: self.vm().msg_sender(),
            spender,
            value,
        });
        Ok(true)
    }

    fn transfer_from(
        &mut self,
        from: Address,
        to: Address,
        value: U256,
    ) -> Result<bool, Erc20Error> {
        let sender = self.vm().msg_sender();
        let mut sender_allowances = self.erc20.allowances.setter(from);
        let mut allowance = sender_allowances.setter(sender);
        let old_allowance = allowance.get();
        if old_allowance < value {
            return Err(Erc20Error::InsufficientAllowance(erc20::InsufficientAllowance {
                owner: from,
                spender: sender,
                have: old_allowance,
                want: value,
            }));
        }
        allowance.set(old_allowance - value);
        self.erc20.transfer_impl(from, to, value)?;
        Ok(true)
    }

    fn allowance(&self, owner: Address, spender: Address) -> Result<U256, Erc20Error> {
        Ok(self.erc20.allowances.getter(owner).get(spender))
    }
}
