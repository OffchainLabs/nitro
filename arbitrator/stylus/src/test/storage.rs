// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{LoadBytes32, StoreBytes32},
    native::NativeInstance,
};
use parking_lot::Mutex;
use prover::utils::Bytes32;
use std::{collections::HashMap, sync::Arc};

#[derive(Clone, Default)]
pub(crate) struct TestEvmAPI(Arc<Mutex<HashMap<Bytes32, Bytes32>>>);

impl TestEvmAPI {
    pub fn get(&self, key: &Bytes32) -> Option<Bytes32> {
        self.0.lock().get(key).cloned()
    }

    pub fn set(&self, key: Bytes32, value: Bytes32) {
        self.0.lock().insert(key, value);
    }

    pub fn getter(&self) -> LoadBytes32 {
        let storage = self.clone();
        Box::new(move |key| (storage.get(&key).unwrap().to_owned(), 2100))
    }

    pub fn setter(&self) -> StoreBytes32 {
        let storage = self.clone();
        Box::new(move |key, value| {
            drop(storage.set(key, value));
            (22100, false)
        })
    }
}

impl NativeInstance {
    pub(crate) fn set_test_evm_api(&mut self) -> TestEvmAPI {
        let api = TestEvmAPI::default();
        let call = Box::new(|_, _, _, _| panic!("can't call contracts"));
        self.env_mut().set_evm_api(api.getter(), api.setter(), call);
        api
    }
}
