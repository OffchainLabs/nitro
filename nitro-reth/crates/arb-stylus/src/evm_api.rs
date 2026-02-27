use alloy_primitives::{Address, B256, U256};
use crate::ink::{Gas, Ink};

/// Status codes returned by EVM API operations.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum EvmApiStatus {
    Success = 0,
    Failure = 1,
    OutOfGas = 2,
    WriteProtection = 3,
}

/// Outcome kind from a user program or call.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum UserOutcomeKind {
    Success = 0,
    Revert = 1,
    Failure = 2,
    OutOfInk = 3,
    OutOfStack = 4,
}

/// Response from a CREATE operation.
pub enum CreateResponse {
    Success(Address),
    Fail(String),
}

/// The EVM API trait that Stylus programs use to interact with EVM state.
///
/// This is the bridge between the WASM runtime and the EVM execution environment.
/// Implementations are provided by the block executor.
pub trait EvmApi: Send + 'static {
    /// Read a storage slot. Returns the value and access cost.
    fn get_bytes32(&mut self, key: B256, evm_api_gas_to_use: Gas) -> eyre::Result<(B256, Gas)>;

    /// Cache a storage value for later flushing.
    fn cache_bytes32(&mut self, key: B256, value: B256) -> eyre::Result<Gas>;

    /// Flush the storage cache to EVM state.
    fn flush_storage_cache(
        &mut self,
        clear: bool,
        gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)>;

    /// Read a transient storage slot.
    fn get_transient_bytes32(&mut self, key: B256) -> eyre::Result<B256>;

    /// Write a transient storage slot.
    fn set_transient_bytes32(
        &mut self,
        key: B256,
        value: B256,
    ) -> eyre::Result<UserOutcomeKind>;

    /// Execute a CALL. Returns return data length, gas cost, and outcome.
    fn contract_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        value: U256,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Execute a DELEGATECALL.
    fn delegate_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Execute a STATICCALL.
    fn static_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Deploy via CREATE.
    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)>;

    /// Deploy via CREATE2.
    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: U256,
        salt: B256,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)>;

    /// Get the return data from the last call.
    fn get_return_data(&self) -> Vec<u8>;

    /// Emit a log with the given data and number of topics.
    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> eyre::Result<()>;

    /// Get an account's balance. Returns balance and access cost.
    fn account_balance(&mut self, address: Address) -> eyre::Result<(U256, Gas)>;

    /// Get an account's code. Returns code and access cost.
    fn account_code(
        &mut self,
        arbos_version: u64,
        address: Address,
        gas_left: Gas,
    ) -> eyre::Result<(Vec<u8>, Gas)>;

    /// Get an account's code hash. Returns hash and access cost.
    fn account_codehash(&mut self, address: Address) -> eyre::Result<(B256, Gas)>;

    /// Determine cost of allocating additional WASM pages.
    fn add_pages(&mut self, pages: u16) -> eyre::Result<Gas>;

    /// Capture tracing information for host I/O calls.
    fn capture_hostio(
        &mut self,
        name: &str,
        args: &[u8],
        outs: &[u8],
        start_ink: Ink,
        end_ink: Ink,
    );
}
