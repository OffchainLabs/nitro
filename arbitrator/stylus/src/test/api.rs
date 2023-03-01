// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{LoadBytes32, StoreBytes32},
    native::{self, NativeInstance},
    run::RunProgram,
};
use arbutil::Color;
use eyre::Result;
use parking_lot::Mutex;
use prover::{
    programs::prelude::*,
    utils::{Bytes20, Bytes32},
};
use std::{collections::HashMap, sync::Arc};

#[derive(Clone, Default)]
pub(crate) struct TestEvmContracts(Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>);

impl TestEvmContracts {
    pub fn insert(&mut self, address: Bytes20, name: &str, config: &StylusConfig) -> Result<()> {
        let file = format!("tests/{name}/target/wasm32-unknown-unknown/release/{name}.wasm");
        let wasm = std::fs::read(file)?;
        let module = native::module(&wasm, config.clone())?;
        self.0.lock().insert(address, module);
        Ok(())
    }
}

#[derive(Clone, Default)]
pub(crate) struct TestEvmStorage(Arc<Mutex<HashMap<Bytes20, HashMap<Bytes32, Bytes32>>>>);

impl TestEvmStorage {
    pub fn get_bytes32(&self, program: Bytes20, key: Bytes32) -> Option<Bytes32> {
        self.0.lock().entry(program).or_default().get(&key).cloned()
    }

    pub fn set_bytes32(&mut self, program: Bytes20, key: Bytes32, value: Bytes32) {
        self.0.lock().entry(program).or_default().insert(key, value);
    }

    pub fn getter(&self, program: Bytes20) -> LoadBytes32 {
        let storage = self.clone();
        Box::new(move |key| {
            let value = storage.get_bytes32(program, key).unwrap().to_owned();
            (value, 2100)
        })
    }

    pub fn setter(&self, program: Bytes20) -> StoreBytes32 {
        let mut storage = self.clone();
        Box::new(move |key, value| {
            drop(storage.set_bytes32(program, key, value));
            (22100, false)
        })
    }
}

impl NativeInstance {
    pub(crate) fn set_test_evm_api(
        &mut self,
        address: Bytes20,
        storage: TestEvmStorage,
        contracts: TestEvmContracts,
    ) -> TestEvmStorage {
        let get_bytes32 = storage.getter(address);
        let set_bytes32 = storage.setter(address);
        let config = self.config();
        let moved_storage = storage.clone();

        let call = Box::new(move |address: Bytes20, input: Vec<u8>, gas: u64, _value| {
            // this call function is for testing purposes only and deviates from onchain behavior

            let mut instance = match contracts.0.lock().get(&address) {
                Some(module) => unsafe {
                    NativeInstance::deserialize(module, config.clone()).unwrap()
                },
                None => panic!("No contract at address {}", address.red()),
            };

            instance.set_test_evm_api(address, moved_storage.clone(), contracts.clone());
            instance.set_gas(gas);

            let outcome = instance.run_main(&input, &config).unwrap();
            let gas_left: u64 = instance.gas_left().into();
            let (status, outs) = outcome.into_data();
            (outs, gas - gas_left, status)
        });
        self.env_mut().set_evm_api(get_bytes32, set_bytes32, call);
        storage
    }
}
