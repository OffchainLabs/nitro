use nitro_arbutil::{Bytes20, Bytes32, evm::user::UserOutcomeKind};

use crate::{Gas, Ink};

/// Response from a CREATE operation.
pub enum CreateResponse {
    Success(Bytes20),
    Fail(String),
}

/// The EVM API trait that Stylus programs use to interact with EVM state.
///
/// This is the bridge between the WASM runtime and the EVM execution environment.
/// Implementations are provided by the block executor. Signatures mirror
/// `arbutil::evm::api::EvmApi`; return-data types still differ (owned `Vec<u8>`
/// instead of the `DataReader` generic) until the trait is replaced outright.
pub trait EvmApi: Send + 'static {
    /// Read a storage slot. Returns the value and access cost.
    fn get_bytes32(
        &mut self,
        key: Bytes32,
        evm_api_gas_to_use: Gas,
    ) -> eyre::Result<(Bytes32, Gas)>;

    /// Cache a storage value for later flushing.
    fn cache_bytes32(&mut self, key: Bytes32, value: Bytes32) -> eyre::Result<Gas>;

    /// Flush the storage cache to EVM state.
    fn flush_storage_cache(
        &mut self,
        clear: bool,
        gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)>;

    /// Read a transient storage slot.
    fn get_transient_bytes32(&mut self, key: Bytes32) -> eyre::Result<Bytes32>;

    /// Write a transient storage slot.
    fn set_transient_bytes32(
        &mut self,
        key: Bytes32,
        value: Bytes32,
    ) -> eyre::Result<UserOutcomeKind>;

    /// Execute a CALL. Returns return data length, gas cost, and outcome.
    fn contract_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        value: Bytes32,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Execute a DELEGATECALL.
    fn delegate_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Execute a STATICCALL.
    fn static_call(
        &mut self,
        contract: Bytes20,
        calldata: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> eyre::Result<(u32, Gas, UserOutcomeKind)>;

    /// Deploy via CREATE.
    fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)>;

    /// Deploy via CREATE2.
    fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: Gas,
    ) -> eyre::Result<(CreateResponse, u32, Gas)>;

    /// Get the return data from the last call.
    fn get_return_data(&self) -> Vec<u8>;

    /// Emit a log with the given data and number of topics.
    fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> eyre::Result<()>;

    /// Get an account's balance. Returns balance and access cost.
    fn account_balance(&mut self, address: Bytes20) -> eyre::Result<(Bytes32, Gas)>;

    /// Get an account's code. Returns code and access cost.
    fn account_code(
        &mut self,
        arbos_version: u64,
        address: Bytes20,
        gas_left: Gas,
    ) -> eyre::Result<(Vec<u8>, Gas)>;

    /// Get an account's code hash. Returns hash and access cost.
    fn account_codehash(&mut self, address: Bytes20) -> eyre::Result<(Bytes32, Gas)>;

    /// Charge for allocating WASM memory pages. The implementation owns the
    /// page accounting (open/ever counters and the memory model), mirroring
    /// the Go side of `arbutil`'s `EvmApi`.
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
