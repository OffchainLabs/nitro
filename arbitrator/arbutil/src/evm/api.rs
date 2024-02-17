// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{evm::user::UserOutcomeKind, Bytes20, Bytes32};
use eyre::Result;
use std::sync::Arc;

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

#[derive(Clone, Copy, Debug)]
#[repr(u32)]
pub enum EvmApiMethod {
    GetBytes32,
    SetBytes32,
    ContractCall,
    DelegateCall,
    StaticCall,
    Create1,
    Create2,
    EmitLog,
    AccountBalance,
    AccountCode,
    AccountCodeHash,
    AddPages,
    CaptureHostIO,
}

// This offset is added to EvmApiMethod when sending a request
// in WASM - program done is also indicated by a "request", with the
// id below that offset, indicating program status
pub const EVM_API_METHOD_REQ_OFFSET: u32 = 0x10000000;

// note: clone should not clone actual data, just the reader
pub trait DataReader: Clone + Send + 'static {
    fn get(&self) -> &[u8];
}

// simple implementation for DataReader, in case data comes from a Vec
#[derive(Clone, Debug)]
pub struct VecReader(Arc<Vec<u8>>);

impl VecReader {
    pub fn new(data: Vec<u8>) -> Self {
        Self(Arc::new(data))
    }
}

impl DataReader for VecReader {
    fn get(&self) -> &[u8] {
        self.0.as_slice()
    }
}

pub trait EvmApi<D: DataReader>: Send + 'static {
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
        calldata: &[u8],
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind);

    /// Delegate-calls the contract at the given address.
    /// Returns the EVM return data's length, the gas cost, and whether the call succeeded.
    /// Analogous to `vm.DELEGATECALL`.
    fn delegate_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind);

    /// Static-calls the contract at the given address.
    /// Returns the EVM return data's length, the gas cost, and whether the call succeeded.
    /// Analogous to `vm.STATICCALL`.
    fn static_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
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
    fn get_return_data(&self) -> D;

    /// Emits an EVM log with the given number of topics and data, the first bytes of which should be the topic data.
    /// Returns an error message on failure.
    /// Analogous to `vm.LOG(n)` where n ∈ [0, 4].
    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<()>;

    /// Gets the balance of the given account.
    /// Returns the balance and the access cost in gas.
    /// Analogous to `vm.BALANCE`.
    fn account_balance(&mut self, address: Bytes20) -> (Bytes32, u64);

    /// Returns the code and the access cost in gas.
    /// Analogous to `vm.EXTCODECOPY`.
    fn account_code(&mut self, address: Bytes20, gas_left: u64) -> (D, u64);

    /// Gets the hash of the given address's code.
    /// Returns the hash and the access cost in gas.
    /// Analogous to `vm.EXTCODEHASH`.
    fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, u64);

    /// Determines the cost in gas of allocating additional wasm pages.
    /// Note: has the side effect of updating Geth's memory usage tracker.
    /// Not analogous to any EVM opcode.
    fn add_pages(&mut self, pages: u16) -> u64;

    /// Captures tracing information for hostio invocations during native execution.
    fn capture_hostio(
        &mut self,
        name: &str,
        args: &[u8],
        outs: &[u8],
        start_ink: u64,
        end_ink: u64,
    );
}
