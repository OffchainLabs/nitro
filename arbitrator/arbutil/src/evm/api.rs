// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{evm::user::UserOutcomeKind, Bytes20, Bytes32};
use eyre::Result;

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
    AddressBalance,
    AddressCodeHash,
    BlockHash,
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
    fn address_balance(&mut self, address: Bytes20) -> (Bytes32, u64);
    fn address_code_hash(&mut self, address: Bytes20) -> (Bytes32, u64);
    fn block_hash(&mut self, block: Bytes32) -> (Bytes32, u64);
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
