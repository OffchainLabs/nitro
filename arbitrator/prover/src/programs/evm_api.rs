// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    programs::run::UserOutcomeKind,
    utils::{Bytes20, Bytes32},
};
use eyre::Result;

#[derive(Debug, Default)]
#[repr(C)]
pub struct EvmData {
    pub origin: Bytes20,
    pub return_data_len: u32,
}

impl EvmData {
    pub fn new(origin: Bytes20) -> Self {
        Self {
            origin,
            return_data_len: 0,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum EvmApiStatus {
    Success,
    Failure,
}

impl From<EvmApiStatus> for UserOutcomeKind {
    fn from(value: EvmApiStatus) -> Self {
        match value {
            EvmApiStatus::Success => UserOutcomeKind::Success,
            EvmApiStatus::Failure => UserOutcomeKind::Revert,
        }
    }
}

impl From<u8> for EvmApiStatus {
    fn from(value: u8) -> Self {
        match value {
            0 => Self::Success,
            _ => Self::Failure,
        }
    }
}

#[repr(usize)]
pub enum EvmApiMethod {
    GetBytes32,
    SetBytes32,
    ContractCall,
    DelegateCall,
    StaticCall,
    Create1,
    Create2,
    GetReturnData,
    EmitLog,
}

pub trait EvmApi: Send + 'static {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64);
    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64>;
    fn contract_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind);
    fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind);
    fn static_call(
        &mut self,
        contract: Bytes20,
        input: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind);
    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64);
    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64);
    fn get_return_data(&mut self) -> Vec<u8>;
    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()>;
}
