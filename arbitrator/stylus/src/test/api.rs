// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{LoadBytes32, StoreBytes32},
    native::NativeInstance,
};
use parking_lot::Mutex;
use prover::utils::{Bytes20, Bytes32};
use std::{collections::HashMap, sync::Arc};

#[derive(Clone, Default)]
pub(crate) struct TestEvmContracts(Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>);

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
            Ok(22100)
        })
    }
}

impl NativeInstance {
    pub(crate) fn set_test_evm_api(
        &mut self,
        address: Bytes20,
        storage: TestEvmStorage,
        _contracts: TestEvmContracts,
    ) -> TestEvmStorage {
        let get_bytes32 = storage.getter(address);
        let set_bytes32 = storage.setter(address);

        let call = Box::new(move |_address, _input, _gas, _value| unimplemented!("contract call"));
        self.env_mut().set_evm_api(get_bytes32, set_bytes32, call);
        storage
    }
}
