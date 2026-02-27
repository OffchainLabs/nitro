use alloy_primitives::{Address, B256};

/// Outcome of executing a Stylus WASM program.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum UserOutcome {
    Success = 0,
    Revert = 1,
    Failure = 2,
    OutOfInk = 3,
    OutOfStack = 4,
}

impl UserOutcome {
    /// Convert a raw status byte to a UserOutcome.
    pub fn from_u8(status: u8) -> Option<Self> {
        match status {
            0 => Some(Self::Success),
            1 => Some(Self::Revert),
            2 => Some(Self::Failure),
            3 => Some(Self::OutOfInk),
            4 => Some(Self::OutOfStack),
            _ => None,
        }
    }
}

/// Host I/O request types for Stylus WASM programs.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u32)]
pub enum RequestType {
    GetBytes32 = 0,
    SetTrieSlots = 1,
    GetTransientBytes32 = 2,
    SetTransientBytes32 = 3,
    ContractCall = 4,
    DelegateCall = 5,
    StaticCall = 6,
    Create1 = 7,
    Create2 = 8,
    EmitLog = 9,
    AccountBalance = 10,
    AccountCode = 11,
    AccountCodeHash = 12,
    AddPages = 13,
    CaptureHostIO = 14,
}

impl RequestType {
    /// Convert a raw u32 to a RequestType.
    pub fn from_u32(val: u32) -> Option<Self> {
        match val {
            0 => Some(Self::GetBytes32),
            1 => Some(Self::SetTrieSlots),
            2 => Some(Self::GetTransientBytes32),
            3 => Some(Self::SetTransientBytes32),
            4 => Some(Self::ContractCall),
            5 => Some(Self::DelegateCall),
            6 => Some(Self::StaticCall),
            7 => Some(Self::Create1),
            8 => Some(Self::Create2),
            9 => Some(Self::EmitLog),
            10 => Some(Self::AccountBalance),
            11 => Some(Self::AccountCode),
            12 => Some(Self::AccountCodeHash),
            13 => Some(Self::AddPages),
            14 => Some(Self::CaptureHostIO),
            _ => None,
        }
    }
}

/// EVM context data passed to the Stylus runtime during program execution.
#[derive(Debug, Clone)]
pub struct EvmData {
    pub arbos_version: u64,
    pub block_basefee: B256,
    pub chain_id: u64,
    pub block_coinbase: Address,
    pub block_gas_limit: u64,
    pub block_number: u64,
    pub block_timestamp: u64,
    pub contract_address: Address,
    pub module_hash: B256,
    pub msg_sender: Address,
    pub msg_value: B256,
    pub tx_gas_price: B256,
    pub tx_origin: Address,
    pub reentrant: u32,
    pub cached: bool,
    pub tracing: bool,
}

/// Parameters passed to the Stylus runtime for program execution.
#[derive(Debug, Clone, Copy)]
pub struct ProgParams {
    pub version: u16,
    pub max_depth: u32,
    pub ink_price: u32,
    pub debug_mode: bool,
}

/// Result of a Stylus program activation.
#[derive(Debug, Clone)]
pub struct ActivationResult {
    pub module_hash: B256,
    pub init_gas: u16,
    pub cached_init_gas: u16,
    pub asm_estimate: u32,
    pub footprint: u16,
}

/// Compute EVM memory expansion cost (matches geth's memory.go).
pub fn evm_memory_cost(size: u64) -> u64 {
    let words = to_word_size(size);
    const MEMORY_GAS: u64 = 3;
    const QUAD_COEFF_DIV: u64 = 512;
    let linear_cost = words.saturating_mul(MEMORY_GAS);
    let square_cost = (words.saturating_mul(words)) / QUAD_COEFF_DIV;
    linear_cost.saturating_add(square_cost)
}

/// Round up byte size to 32-byte word count.
pub fn to_word_size(size: u64) -> u64 {
    if size > u64::MAX - 31 {
        return u64::MAX / 32 + 1;
    }
    (size + 31) / 32
}
