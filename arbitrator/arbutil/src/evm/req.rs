// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    evm::{
        api::{DataReader, EvmApi, EvmApiMethod},
        user::UserOutcomeKind,
    },
    Bytes20, Bytes32,
};
use eyre::{bail, eyre, Result};

pub trait RequestHandler<D: DataReader>: Send + 'static {
    fn handle_request(&mut self, _req_type: EvmApiMethod, _req_data: &[u8]) -> (Vec<u8>, D, u64);
}

pub struct EvmApiRequestor<D: DataReader, H: RequestHandler<D>> {
    handler: H,
    last_code: Option<(Bytes20, D)>,
    last_return_data: Option<D>,
}

impl<D: DataReader, H: RequestHandler<D>> EvmApiRequestor<D, H> {
    pub fn new(handler: H) -> Self {
        Self {
            handler,
            last_code: None,
            last_return_data: None,
        }
    }

    fn handle_request(&mut self, req_type: EvmApiMethod, req_data: &[u8]) -> (Vec<u8>, D, u64) {
        self.handler.handle_request(req_type, req_data)
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
        request.extend(contract.as_slice());
        request.extend(value.as_slice());
        request.extend(&gas.to_be_bytes());
        request.extend(input);
        let (res, data, cost) = self.handle_request(call_type, &request);
        let status: UserOutcomeKind = res[0].try_into().unwrap();
        let data_len = data.get().len() as u32;
        self.last_return_data = Some(data);
        (data_len, cost, status)
    }

    pub fn request_handler(&mut self) -> &mut H {
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
        request.extend(&gas.to_be_bytes());
        request.extend(endowment.as_slice());
        if let Some(salt) = salt {
            request.extend(salt.as_slice());
        }
        request.extend(&code);

        let (mut res, data, cost) = self.handle_request(create_type, &request);
        if res.len() != 21 || res[0] == 0 {
            if res.len() > 0 {
                res.drain(0..=0);
            }
            let err_string =
                String::from_utf8(res).unwrap_or(String::from("create_response_malformed"));
            return (Err(eyre!(err_string)), 0, cost);
        }
        res.drain(0..=0);
        let address = res.try_into().unwrap();
        let data_len = data.get().len() as u32;
        self.last_return_data = Some(data);
        (Ok(address), data_len, cost)
    }
}

impl<D: DataReader, H: RequestHandler<D>> EvmApi<D> for EvmApiRequestor<D, H> {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let (res, _, cost) = self.handle_request(EvmApiMethod::GetBytes32, key.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let mut request = vec![];
        request.extend(key.as_slice());
        request.extend(value.as_slice());
        let (res, _, cost) = self.handle_request(EvmApiMethod::SetBytes32, &request);
        if res.len() != 1 {
            bail!("bad response from set_bytes32")
        }
        if res[0] != 1 {
            bail!("write protected")
        }
        Ok(cost)
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

    fn get_return_data(&self) -> D {
        self.last_return_data
            .as_ref()
            .expect("get return data when no data")
            .clone()
    }

    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()> {
        let mut request = topics.to_be_bytes().to_vec();
        request.extend(data.iter());
        let (res, _, _) = self.handle_request(EvmApiMethod::EmitLog, &request);
        if !res.is_empty() {
            bail!(String::from_utf8(res).unwrap_or(String::from("malformed emit-log response")))
        }
        Ok(())
    }

    fn account_balance(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, _, cost) = self.handle_request(EvmApiMethod::AccountBalance, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn account_code(&mut self, address: Bytes20, gas_left: u64) -> (D, u64) {
        if let Some((stored_address, data)) = self.last_code.as_ref() {
            if stored_address.clone() == address {
                return (data.clone(), 0);
            }
        }
        let mut req: Vec<u8> = address.as_slice().into();
        req.extend(gas_left.to_be_bytes());
        let (_, data, cost) = self.handle_request(EvmApiMethod::AccountCode, &req);

        self.last_code = Some((address, data.clone()));
        (data, cost)
    }

    fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let (res, _, cost) = self.handle_request(EvmApiMethod::AccountCodeHash, address.as_slice());
        (res.try_into().unwrap(), cost)
    }

    fn add_pages(&mut self, pages: u16) -> u64 {
        let (_, _, cost) = self.handle_request(EvmApiMethod::AddPages, &pages.to_be_bytes());
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

        request.extend(&start_ink.to_be_bytes());
        request.extend(&end_ink.to_be_bytes());
        request.extend(&(name.len() as u16).to_be_bytes());
        request.extend(&(args.len() as u16).to_be_bytes());
        request.extend(&(outs.len() as u16).to_be_bytes());
        request.extend(name.as_bytes());
        request.extend(args);
        request.extend(outs);
        self.handle_request(EvmApiMethod::CaptureHostIO, &request);
    }
}
