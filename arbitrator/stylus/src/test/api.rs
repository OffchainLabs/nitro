// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{BlockHash, GetBytes32, SetBytes32},
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

#[derive(Clone)]
pub(crate) struct TestEvmContracts {
    contracts: Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>,
    return_data: Arc<Mutex<Vec<u8>>>,
    config: StylusConfig,
}

impl TestEvmContracts {
    pub fn new(config: &StylusConfig) -> Self {
        Self {
            contracts: Arc::new(Mutex::new(HashMap::new())),
            return_data: Arc::new(Mutex::new(vec![])),
            config: config.clone(),
        }
    }

    pub fn insert(&mut self, address: Bytes20, name: &str) -> Result<()> {
        let file = format!("tests/{name}/target/wasm32-unknown-unknown/release/{name}.wasm");
        let wasm = std::fs::read(file)?;
        let module = native::module(&wasm, self.config.clone())?;
        self.contracts.lock().insert(address, module);
        Ok(())
    }
}

#[derive(Clone, Default)]
pub(crate) struct TestEvmStorage(Arc<Mutex<HashMap<Bytes20, HashMap<Bytes32, Bytes32>>>>);

impl TestEvmStorage {
    pub fn block_hash(&self, program: Bytes20, block: Bytes32) -> Option<Bytes32> {
        self.0
            .lock()
            .entry(program)
            .or_default()
            .get(&block)
            .cloned()
    }

    pub fn get_bytes32(&self, program: Bytes20, key: Bytes32) -> Option<Bytes32> {
        self.0.lock().entry(program).or_default().get(&key).cloned()
    }

    pub fn set_bytes32(&mut self, program: Bytes20, key: Bytes32, value: Bytes32) {
        self.0.lock().entry(program).or_default().insert(key, value);
    }

    pub fn block_hasher(&self, program: Bytes20) -> BlockHash {
        let storage = self.clone();
        Box::new(move |key| {
            let value = storage.block_hash(program, key).unwrap().to_owned();
            (value, 20)
        })
    }

    pub fn getter(&self, program: Bytes20) -> GetBytes32 {
        let storage = self.clone();
        Box::new(move |key| {
            let value = storage.get_bytes32(program, key).unwrap().to_owned();
            (value, 2100)
        })
    }

    pub fn setter(&self, program: Bytes20) -> SetBytes32 {
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
        contracts: TestEvmContracts,
    ) -> TestEvmStorage {
        let block_hash = storage.block_hasher(address);
        let get_bytes32 = storage.getter(address);
        let set_bytes32 = storage.setter(address);
        let moved_storage = storage.clone();
        let moved_contracts = contracts.clone();

        let contract_call = Box::new(
            move |address: Bytes20, input: Vec<u8>, gas, _value| unsafe {
                // this call function is for testing purposes only and deviates from onchain behavior
                let contracts = moved_contracts.clone();
                let config = contracts.config.clone();
                *contracts.return_data.lock() = vec![];

                let mut instance = match contracts.contracts.lock().get(&address) {
                    Some(module) => NativeInstance::deserialize(module, config.clone()).unwrap(),
                    None => panic!("No contract at address {}", address.red()),
                };

                instance.set_test_evm_api(address, moved_storage.clone(), contracts.clone());
                instance.set_ink(config.pricing.gas_to_ink(gas));

                let outcome = instance.run_main(&input, &config).unwrap();
                let (status, outs) = outcome.into_data();
                let outs_len = outs.len() as u32;

                let ink_left: u64 = instance.ink_left().into();
                let gas_left = config.pricing.ink_to_gas(ink_left);
                *contracts.return_data.lock() = outs;
                (outs_len, gas - gas_left, status)
            },
        );
        let delegate_call =
            Box::new(move |_contract, _input, _gas| todo!("delegate call not yet supported"));
        let static_call =
            Box::new(move |_contract, _input, _gas| todo!("static call not yet supported"));
        let get_return_data =
            Box::new(move || -> Vec<u8> { contracts.clone().return_data.lock().clone() });
        let create1 =
            Box::new(move |_code, _endowment, _gas| unimplemented!("create1 not supported"));
        let create2 =
            Box::new(move |_code, _endowment, _salt, _gas| unimplemented!("create2 not supported"));
        let emit_log = Box::new(move |_data, _topics| Ok(()));

        self.env_mut().set_evm_api(
            block_hash,
            get_bytes32,
            set_bytes32,
            contract_call,
            delegate_call,
            static_call,
            create1,
            create2,
            get_return_data,
            emit_log,
        );
        storage
    }
}
