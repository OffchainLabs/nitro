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
    GetBytes32,
    SetBytes32,
    ContractCall,
    DelegateCall,
    StaticCall,
    Create1,
    Create2,
    GetReturnData,
    EmitLog,
    AddressBalance,
    AddressCodeHash,
    EvmBlockHash,
}

pub trait EvmApi: Send + 'static {
    /// Reads the 32-byte value in the EVM state trie at offset `key`.
    /// Returns the value and the access cost in gas.
    /// Analogous to `vm.SLOAD`.
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64);

    /// Stores the given value at the given key in the EVM state trie.
    /// Returns the access cost on success.
    /// Analogous to `vm.SSTORE`.
    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64>;

    /// Calls the contract at the given address.
    /// Returns the EVM return data's length, the gas cost, and whether the call succeeded.
    /// Analogous to `vm.CALL`.
    fn contract_call(
        &mut self,
        contract: Bytes20,
        calldata: Vec<u8>,
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind);

    /// Delegate-calls the contract at the given address.
    /// Returns the EVM return data's length, the gas cost, and whether the call succeeded.
    /// Analogous to `vm.DELEGATECALL`.
    fn delegate_call(
        &mut self,
        contract: Bytes20,
        calldata: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind);

    /// Static-calls the contract at the given address.
    /// Returns the EVM return data's length, the gas cost, and whether the call succeeded.
    /// Analogous to `vm.STATICCALL`.
    fn static_call(
        &mut self,
        contract: Bytes20,
        calldata: Vec<u8>,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind);

    /// Deploys a new contract using the init code provided.
    /// Returns the new contract's address on success, or the error reason on failure.
    /// In both cases the EVM return data's length and the overall gas cost are returned too.
    /// Analogous to `vm.CREATE`.
    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64);

    /// Deploys a new contract using the init code provided, with an address determined in part by the `salt`.
    /// Returns the new contract's address on success, or the error reason on failure.
    /// In both cases the EVM return data's length and the overall gas cost are returned too.
    /// Analogous to `vm.CREATE2`.
    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (eyre::Result<Bytes20>, u32, u64);

    /// Returns the EVM return data.
    /// Analogous to `vm.RETURNDATASIZE`.
    fn get_return_data(&mut self) -> Vec<u8>;

    /// Emits an EVM log with the given number of topics and data, the first bytes of which should be the topic data.
    /// Returns an error message on failure.
    /// Analogous to `vm.LOG(n)` where n ∈ [0, 4].
    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()>;
    fn address_balance(&mut self, address: Bytes20) -> (Bytes32, u64);
    fn address_codehash(&mut self, address: Bytes20) -> (Bytes32, u64);
    fn evm_blockhash(&mut self, block: Bytes32) -> (Bytes32, u64);
}
