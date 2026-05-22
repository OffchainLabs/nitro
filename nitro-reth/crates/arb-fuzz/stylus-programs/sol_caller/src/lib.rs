//! Stylus contract that bridges into arbitrary Solidity contracts.
//!
//! Exposes:
//!   `forward(address target, bytes data)        -> bytes`     CALL forward
//!   `forward_static(address target, bytes data) -> bytes`     STATICCALL forward
//!   `last_return()                              -> bytes`     persisted last return
//!   `call_count()                               -> uint256`   bump on every fwd
//!
//! Designed to stress hostio call_contract / static_call_contract and the
//! Stylus -> Solidity gas accounting boundary.
//!
//! Rebuild: `cargo stylus get-initcode --output ../../prebuilt/sol_caller.hex`.

#![cfg_attr(not(any(test, feature = "export-abi")), no_main)]
extern crate alloc;

use alloc::vec::Vec;
use stylus_sdk::{
    alloy_primitives::{Address, Bytes, B256, U256},
    call,
    deploy::RawDeploy,
    prelude::*,
};

sol_storage! {
    #[entrypoint]
    pub struct SolCaller {
        bytes last_return;
        uint256 call_count;
    }
}

#[public]
impl SolCaller {
    pub fn forward(&mut self, target: Address, data: Bytes) -> Result<Bytes, Vec<u8>> {
        let ctx = Call::new_mutating(self);
        let host = self.vm();
        let out = call::call(host, ctx, target, data.as_ref()).map_err(|_| Vec::<u8>::new())?;
        self.last_return.set_bytes(out.clone());
        let c = self.call_count.get();
        self.call_count.set(c + U256::from(1));
        Ok(out.into())
    }

    pub fn forward_static(
        &mut self,
        target: Address,
        data: Bytes,
    ) -> Result<Bytes, Vec<u8>> {
        let ctx = Call::new();
        let host = self.vm();
        let out = call::static_call(host, ctx, target, data.as_ref()).map_err(|_| Vec::<u8>::new())?;
        self.last_return.set_bytes(out.clone());
        Ok(out.into())
    }

    pub fn last_return(&self) -> Bytes {
        self.last_return.get_bytes().into()
    }

    pub fn call_count(&self) -> U256 {
        self.call_count.get()
    }

    pub fn forward_delegate(&mut self, target: Address, data: Bytes) -> Result<Bytes, Vec<u8>> {
        let ctx = Call::new_mutating(self);
        let host = self.vm();
        let out = unsafe {
            call::delegate_call(host, ctx, target, data.as_ref())
                .map_err(|_| Vec::<u8>::new())?
        };
        self.last_return.set_bytes(out.clone());
        Ok(out.into())
    }

    pub fn do_create(&mut self, endowment: U256, init_code: Bytes) -> Result<Address, Vec<u8>> {
        let deployer = RawDeploy::new();
        let host = self.vm();
        let addr = unsafe {
            deployer
                .deploy(host, init_code.as_ref(), endowment)
                .map_err(|_| Vec::<u8>::new())?
        };
        Ok(addr)
    }

    pub fn do_create2(
        &mut self,
        endowment: U256,
        salt: B256,
        init_code: Bytes,
    ) -> Result<Address, Vec<u8>> {
        let deployer = RawDeploy::new().salt(salt);
        let host = self.vm();
        let addr = unsafe {
            deployer
                .deploy(host, init_code.as_ref(), endowment)
                .map_err(|_| Vec::<u8>::new())?
        };
        Ok(addr)
    }
}
