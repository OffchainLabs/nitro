// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{native, run::RunProgram};
use arbutil::{
    evm::{api::EvmApi, user::UserOutcomeKind, EvmData},
    Bytes20, Bytes32,
};
use eyre::Result;
use parking_lot::Mutex;
use prover::programs::{memory::MemoryModel, prelude::*};
use std::{collections::HashMap, sync::Arc};

use super::TestInstance;

#[derive(Clone, Debug)]
pub(crate) struct TestEvmApi {
    contracts: Arc<Mutex<HashMap<Bytes20, Vec<u8>>>>,
    storage: Arc<Mutex<HashMap<Bytes20, HashMap<Bytes32, Bytes32>>>>,
    program: Bytes20,
    write_result: Arc<Mutex<Vec<u8>>>,
    compile: CompileConfig,
    configs: Arc<Mutex<HashMap<Bytes20, StylusConfig>>>,
    evm_data: EvmData,
    pages: Arc<Mutex<(u16, u16)>>,
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
            write_result: Arc::new(Mutex::new(vec![])),
            compile,
            configs: Arc::new(Mutex::new(HashMap::new())),
            evm_data,
            pages: Arc::new(Mutex::new((0, 0))),
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

    pub fn set_pages(&mut self, open: u16) {
        let mut pages = self.pages.lock();
        pages.0 = open;
        pages.1 = open.max(pages.1);
    }
}

impl EvmApi for TestEvmApi {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let storage = &mut self.storage.lock();
        let storage = storage.get_mut(&self.program).unwrap();
        let value = storage.get(&key).cloned().unwrap_or_default();
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
        calldata: &[u8],
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
        let outcome = native.run_main(calldata, config, ink).unwrap();
        let (status, outs) = outcome.into_data();
        let outs_len = outs.len() as u32;

        let ink_left: u64 = native.ink_left().into();
        let gas_left = config.pricing.ink_to_gas(ink_left);
        *self.write_result.lock() = outs;
        (outs_len, gas - gas_left, status)
    }

    fn delegate_call(
        &mut self,
        _contract: Bytes20,
        _calldata: &[u8],
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!("delegate call not yet supported")
    }

    fn static_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        println!("note: overriding static call with call");
        self.contract_call(contract, calldata, gas, Bytes32::default())
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

    fn get_return_data(&mut self, offset: u32, size: u32) -> Vec<u8> {
        arbutil::slice_with_runoff(
            &self.write_result.lock().as_slice(),
            offset as usize,
            offset.saturating_add(size) as usize,
        )
        .to_vec()
    }

    fn emit_log(&mut self, _data: Vec<u8>, _topics: u32) -> Result<()> {
        Ok(()) // pretend a log was emitted
    }

    fn account_balance(&mut self, _address: Bytes20) -> (Bytes32, u64) {
        unimplemented!()
    }

    fn account_codehash(&mut self, _address: Bytes20) -> (Bytes32, u64) {
        unimplemented!()
    }

    fn add_pages(&mut self, new: u16) -> u64 {
        let model = MemoryModel::new(2, 1000);
        let (open, ever) = *self.pages.lock();

        let mut pages = self.pages.lock();
        pages.0 = pages.0.saturating_add(new);
        pages.1 = pages.1.max(pages.0);
        model.gas_cost(new, open, ever)
    }

    fn capture_hostio(
        &self,
        _name: &str,
        _args: &[u8],
        _outs: &[u8],
        _start_ink: u64,
        _after_ink: u64,
    ) {
        unimplemented!()
    }
}
