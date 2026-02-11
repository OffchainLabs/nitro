// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Warning: this code is for testing only and has not been audited

#![cfg_attr(not(feature = "export-abi"), no_main, no_std)]
extern crate alloc;

use crate::erc20::{Erc20, Erc20Params};
use alloc::{string::String, vec, vec::Vec};
use stylus_sdk::{alloy_primitives::U256, call::transfer::transfer_eth, prelude::*};

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
    #[entrypoint] // Makes Weth the entrypoint
    struct Weth {
        #[borrow] // Allows erc20 to access Weth's storage and make calls
        Erc20<WethParams> erc20;
    }
}

// Another contract we'd like to call
sol_interface! {
    interface IMath {
        function sumValues(uint256[] values) external pure returns (string, uint256);
    }
}

#[public]
#[inherit(Erc20<WethParams>)]
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
    pub fn sum_values(values: Vec<U256>) -> Result<(String, U256), Vec<u8>> {
        Ok(("sum".into(), values.iter().sum()))
    }

    // calls the sum_values() method from the interface
    pub fn sum_with_helper(&self, helper: IMath, values: Vec<U256>) -> Result<U256, Vec<u8>> {
        let (text, sum) = helper.sum_values(self.vm(), Call::new(), values)?;
        assert_eq!(&text, "sum");
        Ok(sum)
    }
}
