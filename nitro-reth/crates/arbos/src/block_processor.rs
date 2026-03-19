use alloy_primitives::{Address, B256, U256};

use arb_chainspec::arbos_version as arb_ver;

use crate::{header::ArbHeaderInfo, internal_tx::L1Info, l2_pricing::GETH_BLOCK_GAS_LIMIT};

/// Standard Ethereum transaction gas.
const TX_GAS: u64 = 21_000;

// =====================================================================
// Conditional options
// =====================================================================

/// Conditional options that may be attached to a transaction.
#[derive(Debug, Clone, Default)]
pub struct ConditionalOptions {
    pub known_accounts: Vec<(Address, Option<B256>)>,
    pub block_number_min: Option<u64>,
    pub block_number_max: Option<u64>,
    pub timestamp_min: Option<u64>,
    pub timestamp_max: Option<u64>,
}

// =====================================================================
// Sequencing hooks
// =====================================================================

/// Hooks for the sequencer to control block production.
pub trait SequencingHooks {
    /// Returns the next transaction to include, or None if the block is complete.
    fn next_tx_to_sequence(&mut self) -> Option<Vec<u8>>;

    /// Filters a transaction before execution.
    fn pre_tx_filter(&self, tx: &[u8]) -> Result<(), String>;

    /// Filters a transaction after execution.
    fn post_tx_filter(&self, tx: &[u8], result: &[u8]) -> Result<(), String>;

    /// Determines whether to discard invalid txs early.
    fn discard_invalid_txs_early(&self) -> bool;

    /// Block-level filter applied after all transactions are processed.
    fn block_filter(
        &self,
        _header: &NewHeaderResult,
        _txs: &[Vec<u8>],
        _receipts: &[Vec<u8>],
    ) -> Result<(), String> {
        Ok(())
    }

    /// Inserts the error for the last tx.
    fn insert_last_tx_error(&mut self, _err: String) {}
}

/// Default no-op implementation for sequencing hooks.
pub struct NoopSequencingHooks;

impl SequencingHooks for NoopSequencingHooks {
    fn next_tx_to_sequence(&mut self) -> Option<Vec<u8>> {
        None
    }

    fn pre_tx_filter(&self, _tx: &[u8]) -> Result<(), String> {
        Ok(())
    }

    fn post_tx_filter(&self, _tx: &[u8], _result: &[u8]) -> Result<(), String> {
        Ok(())
    }

    fn discard_invalid_txs_early(&self) -> bool {
        false
    }
}

// =====================================================================
// Block production types
// =====================================================================

/// The result of block production.
#[derive(Debug, Clone)]
pub struct BlockProductionResult {
    pub l1_info: L1Info,
    pub num_txs: usize,
    pub gas_used: u64,
}

/// Parameters for creating a new block header.
#[derive(Debug, Clone)]
pub struct NewHeaderParams {
    pub parent_hash: B256,
    pub parent_number: u64,
    pub parent_timestamp: u64,
    pub parent_extra_data: Vec<u8>,
    pub parent_mix_hash: B256,
    pub coinbase: Address,
    pub timestamp: u64,
    pub base_fee: U256,
}

/// Computed header fields from `create_new_header`.
#[derive(Debug, Clone)]
pub struct NewHeaderResult {
    pub parent_hash: B256,
    pub coinbase: Address,
    pub number: u64,
    pub gas_limit: u64,
    pub timestamp: u64,
    pub extra_data: Vec<u8>,
    pub mix_hash: B256,
    pub base_fee: U256,
    pub difficulty: U256,
}

// =====================================================================
// Header creation and finalization
// =====================================================================

/// Create new header fields for an Arbitrum block.
///
/// In reth, the actual header construction is done by the block builder;
/// this computes the Arbitrum-specific fields.
pub fn create_new_header(
    l1_info: Option<&L1Info>,
    prev_hash: B256,
    prev_number: u64,
    prev_timestamp: u64,
    prev_extra: &[u8],
    prev_mix_hash: B256,
    base_fee: U256,
) -> NewHeaderResult {
    let mut timestamp = 0u64;
    let mut coinbase = Address::ZERO;

    if let Some(info) = l1_info {
        timestamp = info.l1_timestamp;
        coinbase = info.poster;
    }

    if timestamp < prev_timestamp {
        timestamp = prev_timestamp;
    }

    let mut extra_data = vec![0u8; 32];
    let copy_len = prev_extra.len().min(32);
    extra_data[..copy_len].copy_from_slice(&prev_extra[..copy_len]);

    NewHeaderResult {
        parent_hash: prev_hash,
        coinbase,
        number: prev_number + 1,
        gas_limit: GETH_BLOCK_GAS_LIMIT,
        timestamp,
        extra_data,
        mix_hash: prev_mix_hash,
        base_fee,
        difficulty: U256::from(1),
    }
}

/// Compute the Arbitrum header info to finalize a block.
///
/// This corresponds to `FinalizeBlock` which sets header fields from ArbOS state.
/// We derive the info and let the block assembler apply it.
pub fn finalize_block_header_info(
    send_root: B256,
    send_count: u64,
    l1_block_number: u64,
    arbos_version: u64,
) -> ArbHeaderInfo {
    ArbHeaderInfo {
        send_root,
        send_count,
        l1_block_number,
        arbos_format_version: arbos_version,
    }
}

// =====================================================================
// Block production engine
// =====================================================================

/// The outcome of attempting to apply a single transaction.
#[derive(Debug)]
pub enum TxOutcome {
    /// Transaction was executed successfully.
    Success(TxResult),
    /// Transaction was invalid and should be skipped.
    Invalid(String),
}

/// A successfully executed transaction's metadata.
#[derive(Debug, Clone)]
pub struct TxResult {
    /// Gas used by this transaction (from header gas tracking).
    pub gas_used: u64,
    /// L1 poster data gas for this transaction.
    pub data_gas: u64,
    /// Whether the EVM execution itself succeeded (receipt status).
    pub evm_success: bool,
    /// Scheduled retryable redeems produced by this tx.
    pub scheduled_txs: Vec<Vec<u8>>,
    /// Whether the EVM reported an error (internal txs must not fail).
    pub evm_error: Option<String>,
}

/// Per-tx decision made by the block production loop.
#[derive(Debug)]
pub enum TxAction {
    /// Execute the internal start-block transaction.
    ExecuteStartBlock,
    /// Execute a retryable redeem.
    ExecuteRedeem(Vec<u8>),
    /// Execute a user/sequencer transaction.
    ExecuteUserTx(Vec<u8>),
    /// Block is complete.
    Done,
}

/// Tracks block-level state during production.
///
/// The block executor creates a `BlockProductionState` at the start of the
/// block, then calls `next_tx_action` in a loop to get transactions, and
/// `record_tx_outcome` after executing each one. After the loop, call
/// `finalize` for post-block checks.
#[derive(Debug)]
pub struct BlockProductionState {
    /// Block gas remaining for rate-limiting.
    pub block_gas_left: u64,
    /// Pending retryable redeems scheduled by prior transactions.
    redeems: Vec<Vec<u8>>,
    /// Whether the internal start-block tx has been produced yet.
    start_block_produced: bool,
    /// Count of user transactions processed.
    user_txs_processed: usize,
    /// Expected balance delta from L1 deposits/withdrawals.
    pub expected_balance_delta: i128,
    /// The ArbOS version (may be updated after internal tx).
    arbos_version: u64,
    /// Block timestamp.
    pub timestamp: u64,
    /// Block base fee.
    pub base_fee: U256,
}

impl BlockProductionState {
    /// Create a new block production state.
    pub fn new(
        per_block_gas_limit: u64,
        arbos_version: u64,
        timestamp: u64,
        base_fee: U256,
    ) -> Self {
        Self {
            block_gas_left: per_block_gas_limit,
            redeems: Vec::new(),
            start_block_produced: false,
            user_txs_processed: 0,
            expected_balance_delta: 0,
            arbos_version,
            timestamp,
            base_fee,
        }
    }

    /// Get the next transaction action. The block executor calls this in a loop.
    pub fn next_tx_action<H: SequencingHooks>(&mut self, hooks: &mut H) -> TxAction {
        if !self.start_block_produced {
            self.start_block_produced = true;
            return TxAction::ExecuteStartBlock;
        }

        // Process queued redeems first (FIFO).
        if !self.redeems.is_empty() {
            let redeem = self.redeems.remove(0);
            return TxAction::ExecuteRedeem(redeem);
        }

        // Ask the sequencer for the next transaction.
        match hooks.next_tx_to_sequence() {
            Some(tx_bytes) => {
                // If the block has no gas left, skip user txs.
                if self.block_gas_left < TX_GAS {
                    hooks.insert_last_tx_error("block gas limit reached".to_string());
                    return TxAction::Done;
                }
                TxAction::ExecuteUserTx(tx_bytes)
            }
            None => TxAction::Done,
        }
    }

    /// Check whether a user tx can fit in the remaining block gas.
    ///
    /// In ArbOS < 50, user txs whose compute gas exceeds block_gas_left
    /// are rejected (after the first tx). In ArbOS >= 50, per-tx gas limiting
    /// is handled in the gas charging hook instead.
    pub fn should_reject_for_block_gas(&self, compute_gas: u64, is_user_tx: bool) -> bool {
        self.arbos_version < arb_ver::ARBOS_VERSION_50
            && compute_gas > self.block_gas_left
            && is_user_tx
            && self.user_txs_processed > 0
    }

    /// Compute the poster data gas cost in L2 terms for block-level tracking.
    pub fn compute_data_gas(poster_cost: U256, base_fee: U256, tx_gas: u64) -> u64 {
        if base_fee.is_zero() {
            return 0;
        }

        let poster_cost_in_l2_gas = poster_cost / base_fee;
        let data_gas: u64 = poster_cost_in_l2_gas.try_into().unwrap_or(u64::MAX);

        // Cap to tx gas limit.
        data_gas.min(tx_gas)
    }

    /// Record the result of executing a transaction.
    ///
    /// Returns an error string if the internal start-block tx failed.
    pub fn record_tx_outcome(
        &mut self,
        action: &TxAction,
        outcome: TxOutcome,
    ) -> Result<(), String> {
        match outcome {
            TxOutcome::Invalid(err) => {
                // Invalid txs still consume a TX_GAS worth of block gas.
                match action {
                    TxAction::ExecuteUserTx(_) => {
                        self.block_gas_left = self.block_gas_left.saturating_sub(TX_GAS);
                        self.user_txs_processed += 1;
                    }
                    _ => {
                        self.block_gas_left = self.block_gas_left.saturating_sub(TX_GAS);
                    }
                }
                tracing::debug!(err, "tx invalid, skipped");
                Ok(())
            }
            TxOutcome::Success(result) => {
                // Internal start-block tx must not fail.
                if matches!(action, TxAction::ExecuteStartBlock) {
                    if let Some(ref err) = result.evm_error {
                        return Err(format!("internal tx failed: {err}"));
                    }
                }

                let tx_gas_used = result.gas_used;
                let data_gas = result.data_gas;

                // Subtract gas burned for scheduled redeems (ArbOS >= 4).
                if self.arbos_version >= arb_ver::ARBOS_VERSION_3 {
                    for scheduled in &result.scheduled_txs {
                        // Each scheduled retryable has gas reserved.
                        // The gas is embedded in the retryable tx encoding;
                        // the executor should subtract it from tx_gas_used.
                        let _ = scheduled; // gas deduction handled by executor
                    }
                }

                // Queue any scheduled redeems.
                self.redeems.extend(result.scheduled_txs);

                // Compute used compute gas for block rate limiting.
                let compute_used = if tx_gas_used >= data_gas {
                    let c = tx_gas_used - data_gas;
                    if c < TX_GAS {
                        TX_GAS
                    } else {
                        c
                    }
                } else {
                    tracing::error!(tx_gas_used, data_gas, "tx used less gas than expected");
                    TX_GAS
                };

                self.block_gas_left = self.block_gas_left.saturating_sub(compute_used);

                if matches!(action, TxAction::ExecuteUserTx(_)) {
                    self.user_txs_processed += 1;
                }

                Ok(())
            }
        }
    }

    /// Track deposit balance delta for post-block verification.
    pub fn track_deposit(&mut self, value: U256) {
        let value_i128: i128 = value.try_into().unwrap_or(i128::MAX);
        self.expected_balance_delta = self.expected_balance_delta.saturating_add(value_i128);
    }

    /// Track withdrawal balance delta from L2->L1 tx events.
    pub fn track_withdrawal(&mut self, value: U256) {
        let value_i128: i128 = value.try_into().unwrap_or(i128::MAX);
        self.expected_balance_delta = self.expected_balance_delta.saturating_sub(value_i128);
    }

    /// Update ArbOS version (called after internal tx execution may upgrade).
    pub fn set_arbos_version(&mut self, version: u64) {
        self.arbos_version = version;
    }

    /// Verify the post-block balance delta matches expected deposits/withdrawals.
    pub fn verify_balance_delta(
        &self,
        actual_balance_delta: i128,
        debug_mode: bool,
    ) -> Result<(), String> {
        if actual_balance_delta == self.expected_balance_delta {
            return Ok(());
        }

        if actual_balance_delta > self.expected_balance_delta || debug_mode {
            return Err(format!(
                "unexpected balance delta {} (expected {})",
                actual_balance_delta, self.expected_balance_delta,
            ));
        }

        // Funds were burnt (not minted), only log an error.
        tracing::error!(
            actual = actual_balance_delta,
            expected = self.expected_balance_delta,
            "unexpected balance delta (funds burnt)"
        );
        Ok(())
    }

    /// Total user transactions processed.
    pub fn user_txs_processed(&self) -> usize {
        self.user_txs_processed
    }

    /// Current ArbOS version.
    pub fn arbos_version(&self) -> u64 {
        self.arbos_version
    }
}
