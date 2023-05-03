// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{native, run::RunProgram};
use arbutil::{
    evm::{api::EvmApi, user::UserOutcomeKind, EvmData},
    Bytes20, Bytes32,
};
use eyre::Result;
use parking_lot::Mutex;
use prover::programs::prelude::*;
use std::{collections::HashMap, sync::Arc};

use super::TestInstance;

#[derive(Clone)]
pub(crate) struct TestEvmApi {
    contracts: Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>,
    storage: Arc<Mutex<HashMap<Bytes20, HashMap<Bytes32, Bytes32>>>>,
    program: Bytes20,
    return_data: Arc<Mutex<Vec<u8>>>,
    compile: CompileConfig,
    configs: Arc<Mutex<HashMap<Bytes20, StylusConfig>>>,
    evm_data: EvmData,
}

impl TestEvmApi {
    pub fn new(compile: CompileConfig) -> (TestEvmApi, EvmData) {
        let program = Bytes20::default();
        let evm_data = EvmData::default();

        let mut storage = HashMap::new();
        storage.insert(program, HashMap::new());

        let api = TestEvmApi {
            contracts: Arc::new(Mutex::new(HashMap::new())),
            storage: Arc::new(Mutex::new(storage)),
            program,
            return_data: Arc::new(Mutex::new(vec![])),
            compile,
            configs: Arc::new(Mutex::new(HashMap::new())),
            evm_data,
        };
        (api, evm_data)
    }

    pub fn deploy(&mut self, address: Bytes20, config: StylusConfig, name: &str) -> Result<()> {
        let file = format!("tests/{name}/target/wasm32-unknown-unknown/release/{name}.wasm");
        let wasm = std::fs::read(file)?;
        let module = native::module(&wasm, self.compile.clone())?;
        self.contracts.lock().insert(address, module);
        self.configs.lock().insert(address, config);
        Ok(())
    }
}

impl EvmApi for TestEvmApi {
    fn address_balance(&mut self, address: Bytes20) -> (Bytes32, u64) {
        self.address_balance(address)
    }

    fn address_code_hash(&mut self, address: Bytes20) -> (Bytes32, u64) {
        self.address_code_hash(address)
    }

    fn block_hash(&mut self, block: Bytes32) -> (Bytes32, u64) {
        self.block_hash(block)
    }
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let storage = &mut self.storage.lock();
        let storage = storage.get_mut(&self.program).unwrap();
        let value = storage.get(&key).unwrap().to_owned();
        (value, 2100) // pretend worst case
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let storage = &mut self.storage.lock();
        let storage = storage.get_mut(&self.program).unwrap();
        storage.insert(key, value);
        Ok(22100) // pretend worst case
    }

    /// Simulates a contract call.
    /// Note: this call function is for testing purposes only and deviates from onchain behavior.
    fn contract_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
        _value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        let compile = self.compile.clone();
        let evm_data = self.evm_data;
        let config = *self.configs.lock().get(&contract).unwrap();

        let mut native = unsafe {
            let contracts = self.contracts.lock();
            let module = contracts.get(&contract).unwrap();
            TestInstance::deserialize(module, compile, self.clone(), evm_data).unwrap()
        };

        let ink = config.pricing.gas_to_ink(gas);
        let outcome = native.run_main(&input, config, ink).unwrap();
        let (status, outs) = outcome.into_data();
        let outs_len = outs.len() as u32;

        let ink_left: u64 = native.ink_left().into();
        let gas_left = config.pricing.ink_to_gas(ink_left);
        *self.return_data.lock() = outs;
        (outs_len, gas - gas_left, status)
    }

    fn delegate_call(
        &mut self,
        _contract: Bytes20,
        _input: Vec<u8>,
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!("delegate call not yet supported")
    }

    fn static_call(
        &mut self,
        _contract: Bytes20,
        _input: Vec<u8>,
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!("static call not yet supported")
    }

    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!("create1 not supported")
    }

    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _salt: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!("create2 not supported")
    }

    fn get_return_data(&mut self) -> Vec<u8> {
        self.return_data.lock().clone()
    }

    fn emit_log(&mut self, _data: Vec<u8>, _topics: u32) -> Result<()> {
        Ok(()) // pretend a log was emitted
    }
}
