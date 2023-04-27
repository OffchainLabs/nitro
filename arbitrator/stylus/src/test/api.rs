// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    native::{self, NativeInstance},
    run::RunProgram,
};
use arbutil::{
    evm::{api::EvmApi, user::UserOutcomeKind, EvmData},
    Bytes20, Bytes32, Color,
};
use eyre::Result;
use parking_lot::Mutex;
use prover::programs::prelude::*;
use std::{collections::HashMap, sync::Arc};

/*#[derive(Clone)]
pub(crate) struct TestEvmContracts {
    contracts: Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>,
    return_data: Arc<Mutex<Vec<u8>>>,
    compile: CompileConfig,
    config: StylusConfig,
}

impl TestEvmContracts {
    pub fn new(compile: CompileConfig, config: StylusConfig) -> Self {
        Self {
            contracts: Arc::new(Mutex::new(HashMap::new())),
            return_data: Arc::new(Mutex::new(vec![])),
            compile,
            config,
        }
    }

    pub fn insert(&mut self, address: Bytes20, name: &str) -> Result<()> {
        let file = format!("tests/{name}/target/wasm32-unknown-unknown/release/{name}.wasm");
        let wasm = std::fs::read(file)?;
        let module = native::module(&wasm, self.compile.clone())?;
        self.contracts.lock().insert(address, module);
        Ok(())
    }
}*/

#[derive(Default)]
pub(crate) struct TestEvmApi {
    storage: HashMap<Bytes20, HashMap<Bytes32, Bytes32>>,
    program: Bytes20,
    return_data: Vec<u8>,
}

impl TestEvmApi {
    pub fn new() -> (TestEvmApi, EvmData) {
        (Self::default(), EvmData::default())
    }
}

impl EvmApi for TestEvmApi {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let storage = self.storage.get_mut(&self.program).unwrap();
        let value = storage.get(&key).unwrap().to_owned();
        (value, 2100)
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let storage = self.storage.get_mut(&self.program).unwrap();
        let value = storage.insert(key, value);
        Ok(22100)
    }

    fn contract_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!()
    }

    fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!("delegate call not yet supported")
    }

    fn static_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!("static call not yet supported")
    }

    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!("create1 not supported")
    }

    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!("create2 not supported")
    }

    fn get_return_data(&mut self) -> Vec<u8> {
        self.return_data.clone()
    }

    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()> {
        Ok(()) // pretend a log was emitted
    }
}

/*impl NativeInstance<TestEvmApi> {
    pub(crate) fn set_test_evm_api(
        &mut self,
        address: Bytes20,
        storage: TestEvmStorage,
        contracts: TestEvmContracts,
    ) -> TestEvmStorage {
        let get_bytes32 = storage.getter(address);
        let set_bytes32 = storage.setter(address);
        let moved_storage = storage.clone();
        let moved_contracts = contracts.clone();

        let contract_call = Box::new(
            move |address: Bytes20, input: Vec<u8>, gas, _value| unsafe {
                // this call function is for testing purposes only and deviates from onchain behavior
                let contracts = moved_contracts.clone();
                let compile = contracts.compile.clone();
                let config = contracts.config;
                *contracts.return_data.lock() = vec![];

                let mut native = match contracts.contracts.lock().get(&address) {
                    Some(module) => NativeInstance::deserialize(module, compile.clone()).unwrap(),
                    None => panic!("No contract at address {}", address.red()),
                };

                native.set_test_evm_api(address, moved_storage.clone(), contracts.clone());
                let ink = config.pricing.gas_to_ink(gas);

                let outcome = native.run_main(&input, config, ink).unwrap();
                let (status, outs) = outcome.into_data();
                let outs_len = outs.len() as u32;

                let ink_left: u64 = native.ink_left().into();
                let gas_left = config.pricing.ink_to_gas(ink_left);
                *contracts.return_data.lock() = outs;
                (outs_len, gas - gas_left, status)
            },
        );
        storage
    }
}
*/
