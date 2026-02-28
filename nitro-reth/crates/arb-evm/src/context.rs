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

    /// Insert a hash, returning `true` if it was already present.
    pub fn insert(&mut self, hash: B256) -> bool {
        let was_present = if let Some(pos) = self.hashes.iter().position(|h| *h == hash) {
            self.hashes.remove(pos);
            true
        } else {
            false
        };
        self.hashes.push(hash);
        if self.hashes.len() > self.max_entries {
            self.hashes.remove(0);
        }
        was_present
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

impl ArbitrumExtraData {
    /// Record a WASM activation for the given module hash.
    ///
    /// Validates that if the same module hash was already activated in this block,
    /// the new activation has the same set of targets. This prevents inconsistent
    /// compilations for different architectures within a single block.
    pub fn activate_wasm(
        &mut self,
        module_hash: B256,
        asm: HashMap<String, Vec<u8>>,
        module: Vec<u8>,
    ) -> Result<(), String> {
        if let Some(existing) = self.activated_wasms.get(&module_hash) {
            // Validate target consistency: the new activation must have the
            // same set of targets as the prior one.
            let existing_targets: Vec<&String> = existing.asm.keys().collect();
            let new_targets: Vec<&String> = asm.keys().collect();
            if existing_targets.len() != new_targets.len()
                || !new_targets.iter().all(|t| existing.asm.contains_key(*t))
            {
                return Err(format!(
                    "inconsistent WASM targets for module {module_hash}: \
                     existing has {:?}, new has {:?}",
                    existing.asm.keys().collect::<Vec<_>>(),
                    asm.keys().collect::<Vec<_>>(),
                ));
            }
        }
        self.activated_wasms.insert(module_hash, ActivatedWasm { asm, module });
        Ok(())
    }

    /// Register a balance burn from SELFDESTRUCT or native token burn.
    ///
    /// Adjusts `unexpected_balance_delta` so that post-block balance verification
    /// accounts for the burned amount (adds to delta).
    pub fn expect_balance_burn(&mut self, amount: u128) {
        self.unexpected_balance_delta = self
            .unexpected_balance_delta
            .saturating_add(amount as i128);
    }

    /// Register a balance mint from native token minting.
    ///
    /// Adjusts `unexpected_balance_delta` so that post-block balance verification
    /// accounts for the minted amount (subtracts from delta).
    pub fn expect_balance_mint(&mut self, amount: u128) {
        self.unexpected_balance_delta = self
            .unexpected_balance_delta
            .saturating_sub(amount as i128);
    }

    /// Returns the current unexpected balance delta.
    pub fn unexpected_balance_delta(&self) -> i128 {
        self.unexpected_balance_delta
    }

    // --- Stylus WASM page tracking ---

    /// Returns (open_pages, ever_pages) for Stylus memory accounting.
    pub fn get_stylus_pages(&self) -> (u16, u16) {
        (self.open_wasm_pages, self.ever_wasm_pages)
    }

    /// Returns the current number of open WASM memory pages.
    pub fn get_stylus_pages_open(&self) -> u16 {
        self.open_wasm_pages
    }

    /// Sets the current number of open WASM memory pages.
    pub fn set_stylus_pages_open(&mut self, pages: u16) {
        self.open_wasm_pages = pages;
    }

    /// Adds WASM pages, saturating at u16::MAX.
    /// Returns the previous (open, ever) values.
    pub fn add_stylus_pages(&mut self, new_pages: u16) -> (u16, u16) {
        let prev = (self.open_wasm_pages, self.ever_wasm_pages);
        self.open_wasm_pages = self.open_wasm_pages.saturating_add(new_pages);
        self.ever_wasm_pages = self.ever_wasm_pages.max(self.open_wasm_pages);
        prev
    }

    /// Adds to the ever-pages high watermark, saturating at u16::MAX.
    pub fn add_stylus_pages_ever(&mut self, new_pages: u16) {
        self.ever_wasm_pages = self.ever_wasm_pages.saturating_add(new_pages);
    }

    /// Resets per-transaction Stylus page counters (called at tx start).
    pub fn reset_stylus_pages(&mut self) {
        self.open_wasm_pages = 0;
        self.ever_wasm_pages = 0;
    }

    // --- Transaction filter ---

    /// Mark transaction as filtered (will be excluded at commit).
    pub fn filter_tx(&mut self) {
        self.arb_tx_filter = true;
    }

    /// Clear the transaction filter flag.
    pub fn clear_tx_filter(&mut self) {
        self.arb_tx_filter = false;
    }

    /// Returns whether a transaction is currently filtered.
    pub fn is_tx_filtered(&self) -> bool {
        self.arb_tx_filter
    }

    /// Begin recording WASM modules for block validation.
    pub fn start_recording(&mut self) {
        self.user_wasms.clear();
    }

    /// Record a WASM module's compiled ASM for persistence.
    pub fn record_program(&mut self, module_hash: B256, wasm: ActivatedWasm) {
        self.user_wasms.insert(module_hash, wasm);
    }
}
