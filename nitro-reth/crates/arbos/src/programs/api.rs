use alloy_primitives::{Address, B256, U256};

/// Offset applied to EVM API method request IDs in the Stylus protocol.
pub const EVM_API_METHOD_REQ_OFFSET: u32 = 0x1000_0000;

/// Status codes returned by host I/O operations to the Stylus runtime.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum ApiStatus {
    Success = 0,
    Failure = 1,
    OutOfGas = 2,
    WriteProtection = 3,
}

impl ApiStatus {
    pub fn from_u8(val: u8) -> Option<Self> {
        match val {
            0 => Some(Self::Success),
            1 => Some(Self::Failure),
            2 => Some(Self::OutOfGas),
            3 => Some(Self::WriteProtection),
            _ => None,
        }
    }
}

/// Host I/O operations available to Stylus WASM programs.
///
/// Each method corresponds to a `RequestType` variant and provides
/// the bridge between the WASM runtime and the EVM state.
pub trait HostIo {
    type Error;

    /// Read a storage slot.
    fn get_bytes32(&mut self, key: B256) -> Result<(B256, u64), Self::Error>;

    /// Write storage slots. Returns an API status and gas cost.
    fn set_trie_slots(&mut self, data: &[u8], gas_left: &mut u64)
        -> Result<ApiStatus, Self::Error>;

    /// Read a transient storage slot.
    fn get_transient_bytes32(&mut self, key: B256) -> Result<B256, Self::Error>;

    /// Write a transient storage slot.
    fn set_transient_bytes32(
        &mut self,
        key: B256,
        value: B256,
    ) -> Result<ApiStatus, Self::Error>;

    /// Execute a CALL-family opcode.
    fn contract_call(
        &mut self,
        contract: Address,
        calldata: &[u8],
        gas_left: u64,
        gas_req: u64,
        value: U256,
        call_type: super::types::RequestType,
    ) -> Result<(Vec<u8>, u64), Self::Error>;

    /// Execute CREATE or CREATE2.
    fn create(
        &mut self,
        code: &[u8],
        endowment: U256,
        salt: Option<U256>,
        gas: u64,
    ) -> Result<(Address, Vec<u8>, u64), Self::Error>;

    /// Emit a log.
    fn emit_log(&mut self, topics: &[B256], data: &[u8]) -> Result<(), Self::Error>;

    /// Get an account's balance.
    fn account_balance(&mut self, address: Address) -> Result<(U256, u64), Self::Error>;

    /// Get an account's code.
    fn account_code(&mut self, address: Address, gas: u64) -> Result<(Vec<u8>, u64), Self::Error>;

    /// Get an account's code hash.
    fn account_code_hash(&mut self, address: Address) -> Result<(B256, u64), Self::Error>;

    /// Request additional WASM memory pages.
    fn add_pages(&mut self, pages: u16) -> Result<u64, Self::Error>;

    /// Capture host I/O for tracing.
    fn capture_hostio(
        &mut self,
        name: &str,
        args: &[u8],
        outs: &[u8],
        start_ink: u64,
        end_ink: u64,
    );
}

/// Dispatch a host I/O request to the appropriate HostIo method.
///
/// This is the Rust equivalent of Go's `newApiClosures` return function,
/// which takes a `RequestType` and input bytes and returns
/// `(result, extra_data, gas_cost)`.
pub fn dispatch_request<H: HostIo>(
    host: &mut H,
    req: super::types::RequestType,
    input: &[u8],
) -> Result<(Vec<u8>, Vec<u8>, u64), H::Error> {
    use super::types::RequestType;

    let mut parser = RequestParser::new(input);

    match req {
        RequestType::GetBytes32 => {
            let key = parser.take_hash().expect("expected hash");
            let (out, cost) = host.get_bytes32(key)?;
            Ok((out.as_slice().to_vec(), Vec::new(), cost))
        }
        RequestType::SetTrieSlots => {
            let gas_left = parser.take_u64().expect("expected u64");
            let mut gas = gas_left;
            let status = host.set_trie_slots(parser.take_rest(), &mut gas)?;
            Ok((vec![status as u8], Vec::new(), gas_left - gas))
        }
        RequestType::GetTransientBytes32 => {
            let key = parser.take_hash().expect("expected hash");
            let out = host.get_transient_bytes32(key)?;
            Ok((out.as_slice().to_vec(), Vec::new(), 0))
        }
        RequestType::SetTransientBytes32 => {
            let key = parser.take_hash().expect("expected hash");
            let value = parser.take_hash().expect("expected hash");
            let status = host.set_transient_bytes32(key, value)?;
            Ok((vec![status as u8], Vec::new(), 0))
        }
        RequestType::ContractCall | RequestType::DelegateCall | RequestType::StaticCall => {
            let contract = parser.take_address().expect("expected address");
            let value = parser.take_u256().expect("expected value");
            let gas_left = parser.take_u64().expect("expected gas");
            let gas_req = parser.take_u64().expect("expected gas req");
            let calldata = parser.take_rest();

            let (ret, cost) =
                host.contract_call(contract, calldata, gas_left, gas_req, value, req)?;
            let status: u8 = 0; // success
            Ok((vec![status], ret, cost))
        }
        RequestType::Create1 | RequestType::Create2 => {
            let gas = parser.take_u64().expect("expected gas");
            let endowment = parser.take_u256().expect("expected endowment");
            let salt = if req == RequestType::Create2 {
                Some(parser.take_u256().expect("expected salt"))
            } else {
                None
            };
            let code = parser.take_rest();

            let (addr, ret_val, cost) = host.create(code, endowment, salt, gas)?;
            let mut res = vec![1u8];
            res.extend_from_slice(addr.as_slice());
            Ok((res, ret_val, cost))
        }
        RequestType::EmitLog => {
            let topics = parser.take_u32().expect("expected topic count");
            let mut hashes = Vec::with_capacity(topics as usize);
            for _ in 0..topics {
                hashes.push(parser.take_hash().expect("expected topic hash"));
            }
            host.emit_log(&hashes, parser.take_rest())?;
            Ok((Vec::new(), Vec::new(), 0))
        }
        RequestType::AccountBalance => {
            let address = parser.take_address().expect("expected address");
            let (balance, cost) = host.account_balance(address)?;
            Ok((balance.to_be_bytes::<32>().to_vec(), Vec::new(), cost))
        }
        RequestType::AccountCode => {
            let address = parser.take_address().expect("expected address");
            let gas = parser.take_u64().expect("expected gas");
            let (code, cost) = host.account_code(address, gas)?;
            Ok((Vec::new(), code, cost))
        }
        RequestType::AccountCodeHash => {
            let address = parser.take_address().expect("expected address");
            let (hash, cost) = host.account_code_hash(address)?;
            Ok((hash.as_slice().to_vec(), Vec::new(), cost))
        }
        RequestType::AddPages => {
            let pages = parser.take_u16().expect("expected pages");
            let cost = host.add_pages(pages)?;
            Ok((Vec::new(), Vec::new(), cost))
        }
        RequestType::CaptureHostIO => {
            let start_ink = parser.take_u64().expect("expected start ink");
            let end_ink = parser.take_u64().expect("expected end ink");
            let name_len = parser.take_u32().expect("expected name len");
            let args_len = parser.take_u32().expect("expected args len");
            let outs_len = parser.take_u32().expect("expected outs len");
            let name_bytes = parser.take_fixed(name_len as usize).expect("expected name");
            let name = core::str::from_utf8(name_bytes).unwrap_or("");
            let args = parser.take_fixed(args_len as usize).expect("expected args");
            let outs = parser.take_fixed(outs_len as usize).expect("expected outs");
            host.capture_hostio(name, args, outs, start_ink, end_ink);
            Ok((Vec::new(), Vec::new(), 0))
        }
    }
}

/// Helper for parsing binary request payloads from the Stylus runtime.
pub struct RequestParser<'a> {
    data: &'a [u8],
    offset: usize,
}

impl<'a> RequestParser<'a> {
    pub fn new(data: &'a [u8]) -> Self {
        Self { data, offset: 0 }
    }

    pub fn remaining(&self) -> &'a [u8] {
        &self.data[self.offset..]
    }

    pub fn take_fixed(&mut self, n: usize) -> Option<&'a [u8]> {
        if self.offset + n > self.data.len() {
            return None;
        }
        let slice = &self.data[self.offset..self.offset + n];
        self.offset += n;
        Some(slice)
    }

    pub fn take_address(&mut self) -> Option<Address> {
        let bytes = self.take_fixed(20)?;
        Some(Address::from_slice(bytes))
    }

    pub fn take_hash(&mut self) -> Option<B256> {
        let bytes = self.take_fixed(32)?;
        Some(B256::from_slice(bytes))
    }

    pub fn take_u256(&mut self) -> Option<U256> {
        let bytes = self.take_fixed(32)?;
        Some(U256::from_be_slice(bytes))
    }

    pub fn take_u64(&mut self) -> Option<u64> {
        let bytes = self.take_fixed(8)?;
        Some(u64::from_be_bytes(bytes.try_into().ok()?))
    }

    pub fn take_u32(&mut self) -> Option<u32> {
        let bytes = self.take_fixed(4)?;
        Some(u32::from_be_bytes(bytes.try_into().ok()?))
    }

    pub fn take_u16(&mut self) -> Option<u16> {
        let bytes = self.take_fixed(2)?;
        Some(u16::from_be_bytes(bytes.try_into().ok()?))
    }

    pub fn take_rest(&mut self) -> &'a [u8] {
        let rest = &self.data[self.offset..];
        self.offset = self.data.len();
        rest
    }
}
