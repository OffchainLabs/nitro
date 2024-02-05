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

pub struct EvmApiRequestor<T: RequestHandler> {
    handler: T,
    last_call_result: Vec<u8>,
}

pub trait RequestHandler: Send + 'static {
    fn handle_request(&mut self, _req_type: EvmApiMethod, _req_data: &[u8]) -> (Vec<u8>, u64);
}

impl<T: RequestHandler> EvmApiRequestor<T> {
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
        (
            self.last_call_result.len().try_into().unwrap(),
            cost,
            status,
        )
    }

    pub fn request_handler(&mut self) -> &mut T {
        &mut self.handler
    }

    fn create_request(
        &mut self,
        create_type: EvmApiMethod,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Option<Bytes32>,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        let mut request = vec![];
        request.extend_from_slice(&gas.to_be_bytes());
        request.extend_from_slice(endowment.as_slice());
        if let Some(salt) = salt {
            request.extend_from_slice(salt.as_slice());
        }
        request.extend_from_slice(&code);

        let (mut res, cost) = self.handler.handle_request(create_type, &request);
        if res.len() < 21 || res[0] == 0 {
            let mut err_string = String::from("create_response_malformed");
            if res.len() > 1 {
                let res = res.drain(1..).collect();
                match String::from_utf8(res) {
                    Ok(str) => err_string = str,
                    Err(_) => {}
                }
            };
            self.last_call_result = err_string.as_bytes().to_vec();
            return (
                Err(eyre!(err_string)),
                self.last_call_result.len() as u32,
                cost,
            );
        }
        let address = res.get(1..21).unwrap().try_into().unwrap();
        self.last_call_result = if res.len() > 21 {
            res.drain(21..).collect()
        } else {
            vec![]
        };
        return (Ok(address), self.last_call_result.len() as u32, cost);
    }
}

impl<T: RequestHandler> EvmApi for EvmApiRequestor<T> {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::GetBytes32, key.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let mut request = vec![];
        request.extend_from_slice(key.as_slice());
        request.extend_from_slice(value.as_slice());
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::SetBytes32, &request);
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
        self.call_request(
            EvmApiMethod::DelegateCall,
            contract,
            input,
            gas,
            Bytes32::default(),
        )
    }

    fn static_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        self.call_request(
            EvmApiMethod::StaticCall,
            contract,
            input,
            gas,
            Bytes32::default(),
        )
    }

    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        self.create_request(EvmApiMethod::Create1, code, endowment, None, gas)
    }

    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        self.create_request(EvmApiMethod::Create2, code, endowment, Some(salt), gas)
    }

    fn get_return_data(&mut self, offset: u32, size: u32) -> Vec<u8> {
        let data = self.last_call_result.as_slice();
        let data_len = data.len();
        let offset = offset as usize;
        let mut size = size as usize;
        if offset >= data_len {
            return vec![];
        }
        if offset + size > data_len {
            size = data_len - offset;
        }
        data[offset..size].to_vec()
    }

    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()> {
        let mut request = topics.to_be_bytes().to_vec();
        request.extend(data.iter());
        let (res, _) = self.handler.handle_request(EvmApiMethod::EmitLog, &request);
        if res.is_empty() {
            Ok(())
        } else {
            Err(eyre!(
                String::from_utf8(res).unwrap_or(String::from("malformed emit-log response"))
            ))
        }
    }

    fn account_balance(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::AccountBalance, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn account_code_size(&mut self, address: Bytes20, gas_left: u64) -> (u32, u64) {
        let mut req: Vec<u8> = address.as_slice().into();
        req.extend(gas_left.to_be_bytes());
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::AccountCodeSize, &req);
        (u32::from_be_bytes(res.try_into().unwrap()), cost)
    }

    fn account_code(
        &mut self,
        address: Bytes20,
        offset: u32,
        size: u32,
        gas_left: u64,
    ) -> (Vec<u8>, u64) {
        let mut req: Vec<u8> = address.as_slice().into();
        req.extend(gas_left.to_be_bytes());
        req.extend(offset.to_be_bytes());
        req.extend(size.to_be_bytes());
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::AccountCodeSize, &req);
        (res, cost)
    }

    fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, cost) = self
            .handler
            .handle_request(EvmApiMethod::AccountCodeHash, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn add_pages(&mut self, pages: u16) -> u64 {
        let (_, cost) = self
            .handler
            .handle_request(EvmApiMethod::AddPages, &pages.to_be_bytes());
        cost
    }

    fn capture_hostio(
        &mut self,
        name: &str,
        args: &[u8],
        outs: &[u8],
        start_ink: u64,
        end_ink: u64,
    ) {
        let mut request = vec![];

        request.extend_from_slice(&start_ink.to_be_bytes());
        request.extend_from_slice(&end_ink.to_be_bytes());
        request.extend_from_slice(&(name.len() as u16).to_be_bytes());
        request.extend_from_slice(&(args.len() as u16).to_be_bytes());
        request.extend_from_slice(&(outs.len() as u16).to_be_bytes());
        request.extend_from_slice(name.as_bytes());
        request.extend_from_slice(args);
        request.extend_from_slice(outs);
        self.handler
            .handle_request(EvmApiMethod::CaptureHostIO, &request);
    }
}
