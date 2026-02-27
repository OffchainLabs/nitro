use alloy_primitives::{Address, B256, U256};
use std::collections::HashMap;

/// Arbitrum-specific block execution context.
///
/// Carries L1 information and Arbitrum state needed during block execution.
/// This is passed as `ExecutionCtx` through reth's block executor pipeline.
#[derive(Debug, Clone, Default)]
pub struct ArbBlockExecutionCtx {
    /// Hash of the parent block.
    pub parent_hash: B256,
    /// Parent beacon block root (for EIP-4788).
    pub parent_beacon_block_root: Option<B256>,
    /// Header extra data (carries send root).
    pub extra_data: Vec<u8>,
    /// Number of delayed messages read (from header nonce).
    pub delayed_messages_read: u64,
    /// L1 block number (from header mix_hash bytes 8-15).
    pub l1_block_number: u64,
    /// Chain ID.
    pub chain_id: u64,
    /// Block timestamp.
    pub block_timestamp: u64,
    /// Block base fee.
    pub basefee: U256,
    /// Time elapsed since parent block (seconds).
    pub time_passed: u64,
    /// L1 base fee from the incoming message header.
    pub l1_base_fee: U256,
    /// L1 pricing: price per unit from L1PricingState.
    pub l1_price_per_unit: U256,
    /// L1 pricing: brotli compression level from ArbOS state.
    pub brotli_compression_level: u64,
    /// ArbOS version.
    pub arbos_version: u64,
    /// Network fee account address.
    pub network_fee_account: Address,
    /// Infrastructure fee account address.
    pub infra_fee_account: Address,
    /// Minimum L2 base fee.
    pub min_base_fee: U256,
    /// Block coinbase (poster address / beneficiary).
    pub coinbase: Address,
}

/// Attributes for building the next Arbitrum block.
///
/// Contains values that cannot be derived from the parent block alone.
#[derive(Debug, Clone)]
pub struct ArbNextBlockEnvCtx {
    /// L1 poster address (becomes the coinbase).
    pub suggested_fee_recipient: Address,
    /// Block timestamp.
    pub timestamp: u64,
    /// Mix hash encoding L1 block info and ArbOS version.
    pub prev_randao: B256,
    /// Extra data (carries send root).
    pub extra_data: Vec<u8>,
    /// Parent beacon block root (for EIP-4788).
    pub parent_beacon_block_root: Option<B256>,
}

/// WASM activation info for a newly activated Stylus program.
#[derive(Debug, Clone)]
pub struct ActivatedWasm {
    /// Compiled ASM per target.
    pub asm: HashMap<String, Vec<u8>>,
    /// Raw WASM module.
    pub module: Vec<u8>,
}

/// LRU-style set of recently seen WASM module hashes.
///
/// Used to avoid redundant compilation of recently activated modules.
#[derive(Debug, Clone, Default)]
pub struct RecentWasms {
    hashes: Vec<B256>,
    max_entries: usize,
}

impl RecentWasms {
    pub fn new(max_entries: usize) -> Self {
        Self {
            hashes: Vec::new(),
            max_entries,
        }
    }

    pub fn insert(&mut self, hash: B256) {
        if let Some(pos) = self.hashes.iter().position(|h| *h == hash) {
            self.hashes.remove(pos);
        }
        self.hashes.push(hash);
        if self.hashes.len() > self.max_entries {
            self.hashes.remove(0);
        }
    }

    pub fn contains(&self, hash: &B256) -> bool {
        self.hashes.contains(hash)
    }
}

/// Extra per-block state tracked during Arbitrum execution.
///
/// In geth this is `ArbitrumExtraData` on StateDB. In reth it lives
/// alongside the block executor as mutable state.
#[derive(Debug, Clone, Default)]
pub struct ArbitrumExtraData {
    /// Net balance change across all accounts (tracks ETH minting/burning).
    pub unexpected_balance_delta: i128,
    /// WASM modules encountered during execution (for recording).
    pub user_wasms: HashMap<B256, ActivatedWasm>,
    /// Number of WASM memory pages currently open (Stylus).
    pub open_wasm_pages: u16,
    /// Peak number of WASM memory pages allocated during this tx.
    pub ever_wasm_pages: u16,
    /// Newly activated WASM modules during this block.
    pub activated_wasms: HashMap<B256, ActivatedWasm>,
    /// Recently activated WASM modules (LRU).
    pub recent_wasms: RecentWasms,
    /// Whether transaction filtering is active.
    pub arb_tx_filter: bool,
}
