// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    evm::{
        api::{EvmApi, EvmApiMethod},
        user::UserOutcomeKind,
    },
    Bytes20, Bytes32,
};
use eyre::{bail, eyre, Result};

pub struct JsEvmApi<T: RequestHandler> {
    handler: T,
    last_call_result: Vec<u8>,
}

pub trait RequestHandler: Send + 'static {
    fn handle_request(&mut self, _req_type: EvmApiMethod, _req_data: &[u8]) -> (Vec<u8>, u64);
}

impl<T: RequestHandler> JsEvmApi<T> {
    pub fn new(handler: T) -> Self {
        Self { 
            handler, 
            last_call_result: Vec::default(),
        }
    }

    fn call_request(
        &mut self,
        call_type: EvmApiMethod,
        contract: Bytes20,
        input: &[u8],
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        let mut request = vec![];
        request.extend_from_slice(contract.as_slice());
        request.extend_from_slice(value.as_slice());
        request.extend_from_slice(&gas.to_be_bytes());
        request.extend_from_slice(input);
        let (mut res, cost) = self.handler.handle_request(call_type, &request);
        let status: UserOutcomeKind = res[0].try_into().unwrap();
        self.last_call_result = res.drain(1..).collect();
        (self.last_call_result.len().try_into().unwrap(), cost, status)
    }

    pub fn request_handler(&mut self) -> &mut T {
        &mut self.handler
    }
}

impl<T: RequestHandler> EvmApi for JsEvmApi<T> {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let (res, cost) = self.handler.handle_request(EvmApiMethod::GetBytes32, key.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let mut request = vec![];
        request.extend_from_slice(key.as_slice());
        request.extend_from_slice(value.as_slice());
        let (res, cost) = self.handler.handle_request(EvmApiMethod::SetBytes32, &request);
        if res.len() == 1 && res[0] == 1 {
            Ok(cost)
        } else {
            bail!("set_bytes32 failed")
        }
    }
    
    fn contract_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        self.call_request(EvmApiMethod::ContractCall, contract, input, gas, value)
    }

    fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        self.call_request(EvmApiMethod::DelegateCall, contract, input, gas, Bytes32::default())
    }

    fn static_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        self.call_request(EvmApiMethod::StaticCall, contract, input, gas, Bytes32::default())
    }

    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        (Err(eyre!("TODO")), 0, 0)
    }

    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _salt: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        (Err(eyre!("TODO")), 0, 0)
    }

    fn get_return_data(&mut self, _offset: u32, _size: u32) -> Vec<u8> {
        self.last_call_result.clone()
    }

    fn emit_log(&mut self, _data: Vec<u8>, _topics: u32) -> Result<()> {
        Err(eyre!("TODO"))
    }

    fn account_balance(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, cost) = self.handler.handle_request(EvmApiMethod::AccountBalance, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, cost) = self.handler.handle_request(EvmApiMethod::AccountCodeHash, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn add_pages(&mut self, pages: u16) -> u64 {
        let (_, cost) = self.handler.handle_request(EvmApiMethod::AddPages, &pages.to_be_bytes());
        cost
    }

    fn capture_hostio(&self, name: &str, args: &[u8], outs: &[u8], start_ink: u64, _end_ink: u64) {
        let args = hex::encode(args);
        let outs = hex::encode(outs);
        println!(
            "Error: unexpected hostio tracing info for {name} while proving: {args}, {outs}, {start_ink}"
        );
    }
}
