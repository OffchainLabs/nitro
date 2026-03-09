use alloy_consensus::{Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::eip2718::Encodable2718;
use alloy_eips::eip2718::Typed2718;
use alloy_evm::block::{
    BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
    BlockExecutorFor, ExecutableTx, OnStateHook,
};
use alloy_evm::RecoveredTx;
use alloy_evm::eth::EthTxResult;
use alloy_evm::eth::receipt_builder::ReceiptBuilder;
use alloy_evm::eth::spec::EthExecutorSpec;
use alloy_evm::eth::{EthBlockExecutionCtx, EthBlockExecutor};
use alloy_evm::tx::{FromRecoveredTx, FromTxWithEncoded};
use alloy_evm::{Database, Evm, EvmFactory};
use alloy_primitives::{Address, B256, Log, TxKind, U256, keccak256};
use arb_chainspec;
use arbos::arbos_state::ArbosState;
use arbos::burn::SystemBurner;
use arbos::internal_tx::{self, InternalTxContext};
use arbos::l1_pricing;
use arbos::retryables;
use arbos::tx_processor::{
    EndTxFeeDistribution, EndTxRetryableParams, SubmitRetryableParams,
    compute_poster_gas, compute_submit_retryable_fees,
};
use arbos::util::tx_type_has_poster_costs;
use arb_primitives::multigas::MultiGas;
use arb_primitives::signed_tx::ArbTransactionExt;
use arb_primitives::tx_types::ArbTxType;
use reth_evm::TransactionEnv;
use revm::context::result::ExecutionResult;
use revm::context::TxEnv;
use revm::database::State;
use revm::inspector::Inspector;

use crate::context::ArbBlockExecutionCtx;
use crate::executor::DefaultArbOsHooks;
use crate::hooks::{ArbOsHooks, EndTxContext};

/// Extension trait for transaction environments that support gas price mutation.
///
/// Arbitrum needs to cap the gas price to the base fee when dropping tips,
/// which requires mutating fields not exposed by the standard `TransactionEnv` trait.
pub trait ArbTransactionEnv: TransactionEnv {
    /// Set the effective gas price (max_fee_per_gas for EIP-1559, gas_price for legacy).
    fn set_gas_price(&mut self, gas_price: u128);
    /// Set the max priority fee per gas (tip cap).
    fn set_gas_priority_fee(&mut self, fee: Option<u128>);
}

impl ArbTransactionEnv for TxEnv {
    fn set_gas_price(&mut self, gas_price: u128) {
        self.gas_price = gas_price;
    }
    fn set_gas_priority_fee(&mut self, fee: Option<u128>) {
        self.gas_priority_fee = fee;
    }
}

/// Extension trait for draining scheduled transactions from the executor.
///
/// After executing a SubmitRetryable or a manual Redeem precompile call,
/// auto-redeem retry transactions may be queued. The block producer must
/// drain and re-inject them in the same block.
pub trait ArbScheduledTxDrain {
    /// Drain any scheduled transactions (e.g. auto-redeem retry txs) produced
    /// by the most recently committed transaction.
    fn drain_scheduled_txs(&mut self) -> Vec<Vec<u8>>;
}

impl<'a, Evm, Spec, R: ReceiptBuilder> ArbScheduledTxDrain for ArbBlockExecutor<'a, Evm, Spec, R> {
    fn drain_scheduled_txs(&mut self) -> Vec<Vec<u8>> {
        self.arb_hooks
            .as_mut()
            .map(|hooks| std::mem::take(&mut hooks.tx_proc.scheduled_txs))
            .unwrap_or_default()
    }
}

/// Arbitrum block executor factory.
///
/// Wraps an `EthBlockExecutor` with ArbOS-specific hooks for gas charging,
/// fee distribution, and L1 data pricing.
#[derive(Debug, Clone)]
pub struct ArbBlockExecutorFactory<R, Spec, EvmF> {
    receipt_builder: R,
    spec: Spec,
    evm_factory: EvmF,
}

impl<R, Spec, EvmF> ArbBlockExecutorFactory<R, Spec, EvmF> {
    pub fn new(receipt_builder: R, spec: Spec, evm_factory: EvmF) -> Self {
        Self { receipt_builder, spec, evm_factory }
    }

    /// Create an executor with the concrete `ArbBlockExecutor` return type.
    ///
    /// Unlike the trait method which returns an opaque type, this provides
    /// access to Arbitrum-specific methods like `drain_scheduled_txs`.
    pub fn create_arb_executor<'a, DB, I>(
        &'a self,
        evm: EvmF::Evm<&'a mut State<DB>, I>,
        ctx: EthBlockExecutionCtx<'a>,
        chain_id: u64,
    ) -> ArbBlockExecutor<'a, EvmF::Evm<&'a mut State<DB>, I>, &'a Spec, &'a R>
    where
        DB: Database + 'a,
        R: ReceiptBuilder,
        Spec: EthExecutorSpec + Clone,
        I: Inspector<EvmF::Context<&'a mut State<DB>>> + 'a,
        EvmF: EvmFactory,
    {
        let extra_bytes = ctx.extra_data.as_ref();
        let (delayed_messages_read, l2_block_number) = decode_extra_fields(extra_bytes);
        let arb_ctx = ArbBlockExecutionCtx {
            parent_hash: ctx.parent_hash,
            parent_beacon_block_root: ctx.parent_beacon_block_root,
            extra_data: extra_bytes[..core::cmp::min(extra_bytes.len(), 32)].to_vec(),
            delayed_messages_read,
            l2_block_number,
            chain_id,
            ..Default::default()
        };
        ArbBlockExecutor {
            inner: EthBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder),
            arb_hooks: None,
            arb_ctx,
            pending_tx: None,
            block_gas_left: 0,
            user_txs_processed: 0,
            gas_used_for_l1: Vec::new(),
            multi_gas_used: Vec::new(),
            expected_balance_delta: 0,
            zombie_accounts: std::collections::HashSet::new(),
            finalise_deleted: std::collections::HashSet::new(),
            touched_accounts: std::collections::HashSet::new(),
        }
    }
}

impl<R, Spec, EvmF> BlockExecutorFactory for ArbBlockExecutorFactory<R, Spec, EvmF>
where
    R: ReceiptBuilder<
            Transaction: Transaction + Encodable2718 + ArbTransactionExt,
            Receipt: TxReceipt<Log = Log> + arb_primitives::SetArbReceiptFields,
        > + 'static,
    Spec: EthExecutorSpec + Clone + 'static,
    EvmF: EvmFactory<
        Tx: FromRecoveredTx<R::Transaction>
            + FromTxWithEncoded<R::Transaction>
            + ArbTransactionEnv,
    >,
    Self: 'static,
{
    type EvmFactory = EvmF;
    type ExecutionCtx<'a> = EthBlockExecutionCtx<'a>;
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: EvmF::Evm<&'a mut State<DB>, I>,
        ctx: Self::ExecutionCtx<'a>,
    ) -> impl BlockExecutorFor<'a, Self, DB, I>
    where
        DB: Database + 'a,
        I: Inspector<EvmF::Context<&'a mut State<DB>>> + 'a,
    {
        let extra_bytes = ctx.extra_data.as_ref();
        let (delayed_messages_read, l2_block_number) = decode_extra_fields(extra_bytes);
        let arb_ctx = ArbBlockExecutionCtx {
            parent_hash: ctx.parent_hash,
            parent_beacon_block_root: ctx.parent_beacon_block_root,
            extra_data: extra_bytes[..core::cmp::min(extra_bytes.len(), 32)].to_vec(),
            delayed_messages_read,
            l2_block_number,
            ..Default::default()
        };
        ArbBlockExecutor {
            inner: EthBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder),
            arb_hooks: None,
            arb_ctx,
            pending_tx: None,
            block_gas_left: 0, // Set from state in apply_pre_execution_changes
            user_txs_processed: 0,
            gas_used_for_l1: Vec::new(),
            multi_gas_used: Vec::new(),
            expected_balance_delta: 0,
            zombie_accounts: std::collections::HashSet::new(),
            finalise_deleted: std::collections::HashSet::new(),
            touched_accounts: std::collections::HashSet::new(),
        }
    }
}

// ---------------------------------------------------------------------------
// Per-transaction state carried between execute and commit
// ---------------------------------------------------------------------------

/// Captured per-transaction state for fee distribution in `commit_transaction`.
struct PendingArbTx {
    sender: Address,
    tx_gas_limit: u64,
    arb_tx_type: Option<ArbTxType>,
    has_poster_costs: bool,
    poster_gas: u64,
    /// Gas that reth's EVM actually charged the sender for (gas_used before
    /// poster/compute adjustments). Zero for paths that bypass reth's EVM
    /// (internal tx, deposit, submit retryable, pre-recorded revert, filtered tx).
    /// Used to compute how much additional balance to debit from the sender.
    evm_gas_used: u64,
    /// Multi-dimensional gas charged during gas charging (L1 calldata component).
    charged_multi_gas: MultiGas,
    /// True when the tx's effective gas price is non-zero. Backlog updates
    /// are skipped when gas price is zero (test/estimation scenarios).
    gas_price_positive: bool,
    /// Retry tx context for end-tx retryable processing.
    retry_context: Option<PendingRetryContext>,
}

/// Context for a retry tx that needs end-tx processing after EVM execution.
struct PendingRetryContext {
    ticket_id: alloy_primitives::B256,
    refund_to: Address,
    #[allow(dead_code)]
    gas_fee_cap: U256,
    max_refund: U256,
    submission_fee_refund: U256,
    /// Call value transferred from escrow; returned to escrow on failure.
    call_value: U256,
}

/// Arbitrum block executor wrapping `EthBlockExecutor`.
///
/// Adds ArbOS-specific pre/post execution logic:
/// - Loads ArbOS state at block start (version, fee accounts)
/// - Adjusts gas accounting for L1 poster costs
/// - Distributes fees to network/infra/poster accounts after each tx
/// - Tracks block gas consumption for rate limiting
pub struct ArbBlockExecutor<'a, Evm, Spec, R: ReceiptBuilder> {
    /// Inner Ethereum block executor.
    pub inner: EthBlockExecutor<'a, Evm, Spec, R>,
    /// ArbOS hooks for per-transaction processing.
    pub arb_hooks: Option<DefaultArbOsHooks>,
    /// Arbitrum-specific block context.
    pub arb_ctx: ArbBlockExecutionCtx,
    /// Per-tx state between execute and commit.
    pending_tx: Option<PendingArbTx>,
    /// Remaining block gas for rate limiting.
    /// Starts at per_block_gas_limit and decreases with each tx's compute gas.
    pub block_gas_left: u64,
    /// Number of user transactions successfully committed.
    /// Used for ArbOS < 50 block gas check (first user tx may exceed limit).
    user_txs_processed: u64,
    /// Per-receipt poster gas (L1 gas component), parallel to the receipts vector.
    /// Used to populate `gasUsedForL1` in RPC receipt responses.
    pub gas_used_for_l1: Vec<u64>,
    /// Per-receipt multi-dimensional gas, parallel to the receipts vector.
    pub multi_gas_used: Vec<MultiGas>,
    /// Expected balance delta from deposits (positive) and L2→L1 withdrawals (negative).
    /// Used for post-block safety verification.
    expected_balance_delta: i128,
    /// Zombie accounts: empty accounts preserved from EIP-161 deletion because
    /// they were touched by a zero-value transfer on pre-Stylus ArbOS.
    zombie_accounts: std::collections::HashSet<Address>,
    /// Accounts removed by per-tx Finalise (EIP-161). Tracked so the producer
    /// can mark them for trie deletion if they existed pre-block.
    finalise_deleted: std::collections::HashSet<Address>,
    /// Accounts modified in the current tx (bypass ops + EVM state).
    /// Per-tx Finalise only processes these, matching Go's journal.dirties.
    touched_accounts: std::collections::HashSet<Address>,
}

impl<'a, Evm, Spec, R: ReceiptBuilder> ArbBlockExecutor<'a, Evm, Spec, R> {
    /// Set the ArbOS hooks for this block execution.
    pub fn with_hooks(mut self, hooks: DefaultArbOsHooks) -> Self {
        self.arb_hooks = Some(hooks);
        self
    }

    /// Set the Arbitrum execution context.
    pub fn with_arb_ctx(mut self, ctx: ArbBlockExecutionCtx) -> Self {
        self.arb_ctx = ctx;
        self
    }

    /// Returns the set of zombie account addresses.
    ///
    /// Zombie accounts are empty accounts that should be preserved in the
    /// state trie (not deleted by EIP-161) because they were re-created by
    /// a zero-value transfer on pre-Stylus ArbOS.
    pub fn zombie_accounts(&self) -> std::collections::HashSet<Address> {
        self.zombie_accounts.clone()
    }

    /// Returns accounts deleted by per-tx Finalise (EIP-161).
    /// These may need trie deletion if they existed pre-block.
    pub fn finalise_deleted(&self) -> &std::collections::HashSet<Address> {
        &self.finalise_deleted
    }

    /// Deduct TX_GAS from block gas budget for a failed/invalid transaction.
    /// Call this when a user transaction fails execution so the block budget
    /// and user-tx counter stay in sync (TX_GAS is charged for invalid txs
    /// and userTxsProcessed is incremented).
    pub fn deduct_failed_tx_gas(&mut self, is_user_tx: bool) {
        const TX_GAS: u64 = 21_000;
        self.block_gas_left = self.block_gas_left.saturating_sub(TX_GAS);
        if is_user_tx {
            self.user_txs_processed += 1;
        }
    }

    /// Drain any scheduled transactions (e.g. auto-redeem retry txs) produced
    /// by the most recently committed transaction. The caller should decode and
    /// re-inject these as new transactions in the same block.
    pub fn drain_scheduled_txs(&mut self) -> Vec<Vec<u8>> {
        self.arb_hooks
            .as_mut()
            .map(|hooks| std::mem::take(&mut hooks.tx_proc.scheduled_txs))
            .unwrap_or_default()
    }

    /// Read state parameters from ArbOS state into the execution context
    /// and create/update the hooks.
    fn load_state_params<D: Database>(
        &mut self,
        arb_state: &ArbosState<D, impl arbos::burn::Burner>,
    ) {
        let arbos_version = arb_state.arbos_version();
        self.arb_ctx.arbos_version = arbos_version;
        // Set thread-locals for precompile access.
        arb_precompiles::set_arbos_version(arbos_version);
        arb_precompiles::set_block_timestamp(self.arb_ctx.block_timestamp);
        arb_precompiles::set_current_l2_block(self.arb_ctx.l2_block_number);
        arb_precompiles::set_cached_l1_block_number(
            self.arb_ctx.l2_block_number,
            self.arb_ctx.l1_block_number,
        );

        if let Ok(addr) = arb_state.network_fee_account() {
            self.arb_ctx.network_fee_account = addr;
        }
        if let Ok(addr) = arb_state.infra_fee_account() {
            self.arb_ctx.infra_fee_account = addr;
        }
        if let Ok(level) = arb_state.brotli_compression_level() {
            self.arb_ctx.brotli_compression_level = level;
        }
        if let Ok(price) = arb_state.l1_pricing_state.price_per_unit() {
            self.arb_ctx.l1_price_per_unit = price;
        }
        if let Ok(min_fee) = arb_state.l2_pricing_state.min_base_fee_wei() {
            self.arb_ctx.min_base_fee = min_fee;
        }

        let per_block_gas_limit = arb_state
            .l2_pricing_state
            .per_block_gas_limit()
            .unwrap_or(0);
        let per_tx_gas_limit = arb_state
            .l2_pricing_state
            .per_tx_gas_limit()
            .unwrap_or(0);

        // Read calldata pricing increase feature flag (ArbOS >= 40).
        let calldata_pricing_increase_enabled =
            arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_40
                && arb_state
                    .features
                    .is_increased_calldata_price_enabled()
                    .unwrap_or(false);

        let hooks = DefaultArbOsHooks::new(
            self.arb_ctx.coinbase,
            arbos_version,
            self.arb_ctx.network_fee_account,
            self.arb_ctx.infra_fee_account,
            self.arb_ctx.min_base_fee,
            per_block_gas_limit,
            per_tx_gas_limit,
            false,
            self.arb_ctx.l1_base_fee,
            calldata_pricing_increase_enabled,
        );
        self.arb_hooks = Some(hooks);
    }
}

impl<'db, DB, E, Spec, R> ArbBlockExecutor<'_, E, Spec, R>
where
    DB: Database + 'db,
    E: Evm<
        DB = &'db mut State<DB>,
        Tx: FromRecoveredTx<R::Transaction>
            + FromTxWithEncoded<R::Transaction>
            + ArbTransactionEnv,
    >,
    Spec: EthExecutorSpec,
    R: ReceiptBuilder<
        Transaction: Transaction + Encodable2718 + ArbTransactionExt,
        Receipt: TxReceipt<Log = Log>,
    >,
    R::Transaction: TransactionEnvelope,
{
    /// Handle SubmitRetryableTx: no EVM execution, all state changes done directly.
    ///
    /// Returns a synthetic execution result (endTxNow=true).
    fn execute_submit_retryable(
        &mut self,
        ticket_id: alloy_primitives::B256,
        tx_type: <R::Transaction as TransactionEnvelope>::TxType,
        mut info: arb_primitives::SubmitRetryableInfo,
    ) -> Result<EthTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>, BlockExecutionError> {
        let sender = info.from;

        // Check if this submit retryable is in the on-chain filter.
        // If filtered, redirect fee_refund_addr and beneficiary to the
        // filtered funds recipient. The retryable is still created but
        // auto-redeem scheduling is skipped.
        let is_filtered = {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let state_ptr: *mut State<DB> = db as *mut State<DB>;
            if let Ok(arb_state) = ArbosState::open(state_ptr, SystemBurner::new(None, false)) {
                if arb_state.filtered_transactions.is_filtered_free(ticket_id) {
                    if let Ok(recipient) = arb_state.filtered_funds_recipient_or_default() {
                        info.fee_refund_addr = recipient;
                        info.beneficiary = recipient;
                    }
                    true
                } else {
                    false
                }
            } else {
                false
            }
        };

        // Compute fees (read block info before mutably borrowing db).
        let block = self.inner.evm().block();
        let current_time = revm::context::Block::timestamp(block).to::<u64>();
        let effective_base_fee = self.arb_ctx.basefee;

        let db: &mut State<DB> = self.inner.evm_mut().db_mut();

        // Mint deposit value to sender.
        mint_balance(db, sender, info.deposit_value);
        self.touched_accounts.insert(sender);

        // Track retryable deposit for balance delta verification.
        let dep_i128: i128 = info.deposit_value.try_into().unwrap_or(i128::MAX);
        self.expected_balance_delta = self.expected_balance_delta.saturating_add(dep_i128);

        // Get sender balance after minting.
        let _ = db.load_cache_account(sender);
        let balance_after_mint = db
            .cache
            .accounts
            .get(&sender)
            .and_then(|a| a.account.as_ref())
            .map(|a| a.info.balance)
            .unwrap_or(U256::ZERO);

        let params = SubmitRetryableParams {
            ticket_id,
            deposit_value: info.deposit_value,
            retry_value: info.retry_value,
            gas_fee_cap: info.gas_fee_cap,
            gas: info.gas,
            max_submission_fee: info.max_submission_fee,
            retry_data_len: info.retry_data.len(),
            l1_base_fee: info.l1_base_fee,
            effective_base_fee,
            current_time,
            balance_after_mint,
            infra_fee_account: self.arb_ctx.infra_fee_account,
            min_base_fee: self.arb_ctx.min_base_fee,
            arbos_version: self.arb_ctx.arbos_version,
        };

        let fees = compute_submit_retryable_fees(&params);

        tracing::debug!(
            target: "arb::executor",
            can_pay = fees.can_pay_for_gas,
            has_error = fees.error.is_some(),
            submission_fee = %fees.submission_fee,
            "submit retryable fee computation"
        );

        let user_gas = info.gas;

        // Fee validation errors end the transaction immediately with zero gas.
        // The deposit was already minted (separate ArbitrumDepositTx), and no
        // further transfers should occur.
        if let Some(ref err) = fees.error {
            tracing::warn!(
                target: "arb::executor",
                ticket_id = %ticket_id,
                error = %err,
                "submit retryable fee validation failed"
            );

            self.pending_tx = Some(PendingArbTx {
                sender,
                tx_gas_limit: user_gas,
                arb_tx_type: Some(ArbTxType::ArbitrumSubmitRetryableTx),
                has_poster_costs: false,
                poster_gas: 0,
                evm_gas_used: 0,

                charged_multi_gas: MultiGas::default(),
                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                retry_context: None,
            });

            return Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result: ExecutionResult::Revert {
                        gas_used: 0,
                        output: alloy_primitives::Bytes::new(),
                    },
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            });
        }

        let db: &mut State<DB> = self.inner.evm_mut().db_mut();

        tracing::debug!(
            target: "arb::executor",
            %ticket_id,
            %sender,
            fee_refund = %info.fee_refund_addr,
            deposit = %info.deposit_value,
            retry_value = %info.retry_value,
            submission_fee = %fees.submission_fee,
            escrow = %fees.escrow,
            can_pay = fees.can_pay_for_gas,
            "SubmitRetryable fee breakdown"
        );

        // 3. Transfer submission fee to network fee account.
        if !fees.submission_fee.is_zero() {
            transfer_balance(db, sender, self.arb_ctx.network_fee_account, fees.submission_fee);
            self.touched_accounts.insert(sender);
            self.touched_accounts.insert(self.arb_ctx.network_fee_account);
        }

        // 4. Refund excess submission fee.
        transfer_balance(db, sender, info.fee_refund_addr, fees.submission_fee_refund);
        self.touched_accounts.insert(sender);
        self.touched_accounts.insert(info.fee_refund_addr);

        // 5. Move call value into escrow. If sender has insufficient funds
        //    (e.g. deposit didn't cover retry_value after fee deductions),
        //    refund the submission fee and end the transaction.
        if !try_transfer_balance(db, sender, fees.escrow, info.retry_value) {
            self.touched_accounts.insert(sender);
            self.touched_accounts.insert(fees.escrow);
            // Refund submission fee from network account back to sender.
            transfer_balance(
                db, self.arb_ctx.network_fee_account, sender, fees.submission_fee,
            );
            self.touched_accounts.insert(self.arb_ctx.network_fee_account);
            // Refund withheld portion of submission fee to fee refund address.
            transfer_balance(
                db, sender, info.fee_refund_addr, fees.withheld_submission_fee,
            );
            self.touched_accounts.insert(info.fee_refund_addr);

            self.pending_tx = Some(PendingArbTx {
                sender,
                tx_gas_limit: user_gas,
                arb_tx_type: Some(ArbTxType::ArbitrumSubmitRetryableTx),
                has_poster_costs: false,
                poster_gas: 0,
                evm_gas_used: 0,

                charged_multi_gas: MultiGas::default(),
                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                retry_context: None,
            });

            return Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result: ExecutionResult::Revert {
                        gas_used: 0,
                        output: alloy_primitives::Bytes::new(),
                    },
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            });
        }
        self.touched_accounts.insert(sender);
        self.touched_accounts.insert(fees.escrow);

        // 6. Create retryable ticket.
        let state_ptr: *mut State<DB> = db as *mut State<DB>;
        if let Ok(arb_state) = ArbosState::open(state_ptr, SystemBurner::new(None, false)) {
            let _ = arb_state.retryable_state.create_retryable(
                ticket_id,
                fees.timeout,
                sender,
                info.retry_to,
                info.retry_value,
                info.beneficiary,
                &info.retry_data,
            );
        }

        // Emit TicketCreated event (always, after retryable creation).
        let mut receipt_logs: Vec<Log> = Vec::new();
        receipt_logs.push(Log {
            address: arb_precompiles::ARBRETRYABLETX_ADDRESS,
            data: alloy_primitives::LogData::new_unchecked(
                vec![
                    arb_precompiles::ticket_created_topic(),
                    ticket_id,
                ],
                alloy_primitives::Bytes::new(),
            ),
        });

        let db: &mut State<DB> = self.inner.evm_mut().db_mut();

        // 7. Handle gas fees if user can pay.
        if fees.can_pay_for_gas {
            // Pay infra fee (skip when infra_fee_account is zero, matching Go).
            if self.arb_ctx.infra_fee_account != Address::ZERO {
                transfer_balance(db, sender, self.arb_ctx.infra_fee_account, fees.infra_cost);
                self.touched_accounts.insert(sender);
                self.touched_accounts.insert(self.arb_ctx.infra_fee_account);
            }
            // Pay network fee.
            if !fees.network_cost.is_zero() {
                transfer_balance(db, sender, self.arb_ctx.network_fee_account, fees.network_cost);
                self.touched_accounts.insert(sender);
                self.touched_accounts.insert(self.arb_ctx.network_fee_account);
            }
            // Gas price refund.
            transfer_balance(db, sender, info.fee_refund_addr, fees.gas_price_refund);
            self.touched_accounts.insert(sender);
            self.touched_accounts.insert(info.fee_refund_addr);

            // For filtered retryables, skip auto-redeem scheduling.
            tracing::debug!(
                target: "arb::executor",
                filtered = is_filtered,
                "auto-redeem: checking is_filtered"
            );
            if !is_filtered {
                // Schedule auto-redeem: use make_tx to construct from stored
                // retryable fields (via MakeTx), then
                // increment num_tries.
                let state_ptr2: *mut State<DB> = db as *mut State<DB>;
                match ArbosState::open(state_ptr2, SystemBurner::new(None, false)) {
                    Ok(arb_state) => {
                    match arb_state.retryable_state.open_retryable(
                        ticket_id,
                        0, // pass 0 so any non-zero timeout is valid
                    ) {
                        Ok(Some(retryable)) => {
                        let _ = retryable.increment_num_tries();

                        match retryable.make_tx(
                            U256::from(self.arb_ctx.chain_id),
                            0, // nonce = 0 for first auto-redeem
                            effective_base_fee,
                            user_gas,
                            ticket_id,
                            info.fee_refund_addr,
                            fees.available_refund,
                            fees.submission_fee,
                        ) {
                            Ok(retry_tx) => {
                            // Compute retry tx hash for the event.
                            let retry_tx_hash = {
                                let mut enc = Vec::new();
                                enc.push(ArbTxType::ArbitrumRetryTx.as_u8());
                                alloy_rlp::Encodable::encode(&retry_tx, &mut enc);
                                keccak256(&enc)
                            };

                            // Emit RedeemScheduled event.
                            let mut event_data = Vec::with_capacity(128);
                            event_data.extend_from_slice(&B256::left_padding_from(&user_gas.to_be_bytes()).0);
                            event_data.extend_from_slice(&B256::left_padding_from(info.fee_refund_addr.as_slice()).0);
                            event_data.extend_from_slice(&fees.available_refund.to_be_bytes::<32>());
                            event_data.extend_from_slice(&fees.submission_fee.to_be_bytes::<32>());

                            receipt_logs.push(Log {
                                address: arb_precompiles::ARBRETRYABLETX_ADDRESS,
                                data: alloy_primitives::LogData::new_unchecked(
                                    vec![
                                        arb_precompiles::redeem_scheduled_topic(),
                                        ticket_id,
                                        retry_tx_hash,
                                        B256::left_padding_from(&0u64.to_be_bytes()),
                                    ],
                                    event_data.into(),
                                ),
                            });

                            if let Some(hooks) = self.arb_hooks.as_mut() {
                                let mut encoded = Vec::new();
                                encoded.push(ArbTxType::ArbitrumRetryTx.as_u8());
                                alloy_rlp::Encodable::encode(&retry_tx, &mut encoded);
                                tracing::debug!(
                                    target: "arb::executor",
                                    encoded_len = encoded.len(),
                                    "Scheduling auto-redeem retry tx"
                                );
                                hooks.tx_proc.scheduled_txs.push(encoded);
                            } else {
                                tracing::warn!(
                                    target: "arb::executor",
                                    "Cannot schedule auto-redeem: arb_hooks is None"
                                );
                            }
                            }
                            Err(_) => {
                                tracing::warn!(
                                    target: "arb::executor",
                                    "Auto-redeem make_tx failed"
                                );
                            }
                        }
                        }
                        Ok(None) => {
                            tracing::warn!(
                                target: "arb::executor",
                                %ticket_id,
                                "open_retryable returned None after create"
                            );
                        }
                        Err(_) => {
                            tracing::warn!(
                                target: "arb::executor",
                                "open_retryable failed"
                            );
                        }
                    }
                    }
                    Err(_) => {
                        tracing::warn!(
                            target: "arb::executor",
                            "ArbosState::open failed for auto-redeem"
                        );
                    }
                }
            }
        } else if !fees.gas_cost_refund.is_zero() {
            // Can't pay for gas: refund gas cost from deposit.
            transfer_balance(db, sender, info.fee_refund_addr, fees.gas_cost_refund);
            self.touched_accounts.insert(sender);
            self.touched_accounts.insert(info.fee_refund_addr);
        }

        // Store pending state for commit_transaction.
        // evm_gas_used must equal gas_used when can_pay_for_gas because the gas
        // fees were already transferred inside execute_submit_retryable. Setting
        // evm_gas_used = gas_used prevents the sender_extra_gas burn in
        // commit_transaction from double-charging the sender.
        let gas_used = if fees.can_pay_for_gas { user_gas } else { 0 };
        self.pending_tx = Some(PendingArbTx {
            sender,
            tx_gas_limit: user_gas,
            arb_tx_type: Some(ArbTxType::ArbitrumSubmitRetryableTx),
            has_poster_costs: false, // No poster costs for submit retryable
            poster_gas: 0,
            evm_gas_used: gas_used,
            charged_multi_gas: if fees.can_pay_for_gas {
                MultiGas::l2_calldata_gas(user_gas)
            } else {
                MultiGas::default()
            },
            gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
            retry_context: None,
        });

        // Construct synthetic execution result. Filtered retryables always
        // return a failure receipt (filteredErr). Non-filtered txs
        // succeed even when can't pay for gas (retryable was created).
        let ticket_bytes = alloy_primitives::Bytes::copy_from_slice(ticket_id.as_slice());

        if is_filtered {
            Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result: ExecutionResult::Revert {
                        gas_used,
                        output: ticket_bytes,
                    },
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            })
        } else {
            Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result: ExecutionResult::Success {
                        reason: revm::context::result::SuccessReason::Return,
                        gas_used,
                        gas_refunded: 0,
                        output: revm::context::result::Output::Call(ticket_bytes),
                        logs: receipt_logs,
                    },
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            })
        }
    }
}

impl<'db, DB, E, Spec, R> BlockExecutor for ArbBlockExecutor<'_, E, Spec, R>
where
    DB: Database + 'db,
    E: Evm<
        DB = &'db mut State<DB>,
        Tx: FromRecoveredTx<R::Transaction>
            + FromTxWithEncoded<R::Transaction>
            + ArbTransactionEnv,
    >,
    Spec: EthExecutorSpec,
    R: ReceiptBuilder<
        Transaction: Transaction + Encodable2718 + ArbTransactionExt,
        Receipt: TxReceipt<Log = Log> + arb_primitives::SetArbReceiptFields,
    >,
    R::Transaction: TransactionEnvelope,
{
    type Transaction = R::Transaction;
    type Receipt = R::Receipt;
    type Evm = E;
    type Result = EthTxResult<E::HaltReason, <R::Transaction as TransactionEnvelope>::TxType>;

    fn apply_pre_execution_changes(&mut self) -> Result<(), BlockExecutionError> {
        self.inner.apply_pre_execution_changes()?;

        // Populate header-derived fields from the EVM block/cfg environment.
        {
            let block = self.inner.evm().block();
            let timestamp = revm::context::Block::timestamp(block).to::<u64>();
            if self.arb_ctx.block_timestamp == 0 {
                self.arb_ctx.block_timestamp = timestamp;
            }
            self.arb_ctx.coinbase = revm::context::Block::beneficiary(block);
            self.arb_ctx.basefee = U256::from(revm::context::Block::basefee(block));
            if let Some(prevrandao) = revm::context::Block::prevrandao(block) {
                if self.arb_ctx.l1_block_number == 0 {
                    self.arb_ctx.l1_block_number =
                        crate::config::l1_block_number_from_mix_hash(&prevrandao);
                }
            }
        }

        // Ensure L2 block number is set for precompile access.
        // block_env.number holds L1 block number; L2 comes from the sealed header
        // (set via arb_context_for_block or with_arb_ctx). If still 0, we're in a
        // path where it wasn't explicitly set — this shouldn't happen in production.
        if self.arb_ctx.l2_block_number > 0 {
            arb_precompiles::set_current_l2_block(self.arb_ctx.l2_block_number);
            arb_precompiles::set_cached_l1_block_number(
                self.arb_ctx.l2_block_number,
                self.arb_ctx.l1_block_number,
            );
        }

        // Load ArbOS state parameters from the EVM database.
        // Block-start operations (pricing model update, retryable reaping, etc.)
        // are triggered by the startBlock internal tx, NOT here.
        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
        let state_ptr: *mut State<DB> = db as *mut State<DB>;

        if let Ok(arb_state) =
            ArbosState::open(state_ptr, SystemBurner::new(None, false))
        {
            // Rotate multi-gas fees: copy next-block fees to current-block.
            let _ = arb_state.l2_pricing_state.commit_multi_gas_fees();

            // Read baseFee from L2PricingState BEFORE StartBlock runs.
            // This value was written by the previous block's StartBlock.
            if let Ok(base_fee) = arb_state.l2_pricing_state.base_fee_wei() {
                self.arb_ctx.basefee = base_fee;
            }

            // Read state parameters for the execution context and hooks.
            self.load_state_params(&arb_state);

            // Initialize block gas rate limiting.
            self.block_gas_left = arb_state
                .l2_pricing_state
                .per_block_gas_limit()
                .unwrap_or(0);

            // Pre-populate the State's block_hashes cache with L1 block hashes
            // from ArbOS state. Arbitrum overrides the BLOCKHASH opcode to return
            // L1 block hashes (not L2). Since block_env.number is already set to
            // the L1 block number, revm's range check uses L1 numbers — we just
            // need to ensure the hash lookup returns L1 hashes.
            if let Ok(l1_block_number) = arb_state.blockhashes.l1_block_number() {
                let lower = l1_block_number.saturating_sub(256);
                // SAFETY: state_ptr is valid for the lifetime of this block.
                let state_ref = unsafe { &mut *state_ptr };
                for n in lower..l1_block_number {
                    if let Ok(Some(hash)) = arb_state.blockhashes.block_hash(n) {
                        state_ref.block_hashes.insert(n, hash);
                    }
                }
            }
        }

        tracing::trace!(
            target: "arb::executor",
            l1_block = self.arb_ctx.l1_block_number,
            delayed_msgs = self.arb_ctx.delayed_messages_read,
            chain_id = self.arb_ctx.chain_id,
            basefee = %self.arb_ctx.basefee,
            arbos_version = self.arb_ctx.arbos_version,
            has_hooks = self.arb_hooks.is_some(),
            "starting block execution"
        );

        Ok(())
    }

    fn execute_transaction_without_commit(
        &mut self,
        tx: impl ExecutableTx<Self>,
    ) -> Result<Self::Result, BlockExecutionError> {
        // Decompose the transaction to extract sender, type, and gas limit.
        let (tx_env, recovered) = tx.into_parts();
        let sender = *recovered.signer();
        let tx_type_raw = recovered.tx().ty();
        let tx_gas_limit = recovered.tx().gas_limit();
        let envelope_tx_type = recovered.tx().tx_type();

        // Classify the transaction type.
        let arb_tx_type = ArbTxType::from_u8(tx_type_raw).ok();
        let is_arb_internal = arb_tx_type == Some(ArbTxType::ArbitrumInternalTx);
        let is_arb_deposit = arb_tx_type == Some(ArbTxType::ArbitrumDepositTx);
        let is_submit_retryable = arb_tx_type == Some(ArbTxType::ArbitrumSubmitRetryableTx);
        let is_retry_tx = arb_tx_type == Some(ArbTxType::ArbitrumRetryTx);
        let is_contract_tx = arb_tx_type == Some(ArbTxType::ArbitrumContractTx);
        let has_poster_costs = tx_type_has_poster_costs(tx_type_raw);

        // Block gas rate limit: reject user txs when block gas budget is
        // exhausted. Internal, deposit, and submit retryable txs always proceed
        // (they are block-critical or come from the delayed inbox).
        let is_user_tx = !is_arb_internal && !is_arb_deposit
            && !is_submit_retryable && !is_retry_tx;
        const TX_GAS_MIN: u64 = 21_000;
        if is_user_tx && self.block_gas_left < TX_GAS_MIN {
            return Err(BlockExecutionError::msg("block gas limit reached"));
        }

        // Reset per-tx processor state.
        crate::evm::reset_stylus_pages();
        if let Some(hooks) = self.arb_hooks.as_mut() {
            hooks.tx_proc.poster_fee = U256::ZERO;
            hooks.tx_proc.poster_gas = 0;
            hooks.tx_proc.compute_hold_gas = 0;
            hooks.tx_proc.current_retryable = None;
            hooks.tx_proc.current_refund_to = None;
            hooks.tx_proc.scheduled_txs.clear();
        }

        // --- Pre-execution: apply special tx type state changes ---

        // Internal txs: verify sender, apply state update, end immediately.
        if is_arb_internal {
            use arbos::tx_processor::ARBOS_ADDRESS;

            if sender != ARBOS_ADDRESS {
                return Err(BlockExecutionError::msg(
                    "internal tx not from ArbOS address",
                ));
            }

            let tx_data = recovered.tx().input().to_vec();
            let tx_type = recovered.tx().tx_type();
            let mut tx_err = None;

            if tx_data.len() >= 4 {
                let selector: [u8; 4] = tx_data[0..4].try_into().unwrap();
                let is_start_block = selector == internal_tx::INTERNAL_TX_START_BLOCK_METHOD_ID;

                if is_start_block {
                    if let Ok(start_data) = internal_tx::decode_start_block_data(&tx_data) {
                        self.arb_ctx.l1_base_fee = start_data.l1_base_fee;
                        self.arb_ctx.time_passed = start_data.time_passed;
                    }
                }

                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                if let Ok(mut arb_state) =
                    ArbosState::open(state_ptr, SystemBurner::new(None, false))
                {
                    let block = self.inner.evm().block();
                    let current_time =
                        revm::context::Block::timestamp(block).to::<u64>();
                    let ctx = InternalTxContext {
                        block_number: revm::context::Block::number(block).to::<u64>(),
                        current_time,
                        prev_hash: self.arb_ctx.parent_hash,
                    };

                    // EIP-2935: Store parent block hash for ArbOS >= 40.
                    if is_start_block
                        && arb_state.arbos_version()
                            >= arb_chainspec::arbos_version::ARBOS_VERSION_40
                    {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        process_parent_block_hash(
                            unsafe { &mut *state_ptr },
                            ctx.block_number,
                            ctx.prev_hash,
                        );
                    }

                    let touched_ptr = &mut self.touched_accounts
                        as *mut std::collections::HashSet<Address>;
                    let zombie_ptr = &mut self.zombie_accounts
                        as *mut std::collections::HashSet<Address>;
                    let finalise_ptr = &self.finalise_deleted
                        as *const std::collections::HashSet<Address>;
                    let arbos_ver = self.arb_ctx.arbos_version;
                    let mut do_transfer = |from: Address, to: Address, amount: U256| {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        unsafe {
                            if amount.is_zero()
                                && arbos_ver < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
                            {
                                create_zombie_if_deleted(
                                    &mut *state_ptr, from, &*finalise_ptr,
                                    &mut *zombie_ptr, &mut *touched_ptr,
                                );
                            }
                            transfer_balance(&mut *state_ptr, from, to, amount);
                            if !amount.is_zero() {
                                (*zombie_ptr).remove(&from);
                            }
                            (*zombie_ptr).remove(&to);
                            (*touched_ptr).insert(from);
                            (*touched_ptr).insert(to);
                        }
                        Ok(())
                    };
                    let mut do_balance = |addr: Address| -> U256 {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        unsafe { get_balance(&mut *state_ptr, addr) }
                    };
                    if let Err(e) = internal_tx::apply_internal_tx_update(
                        &tx_data,
                        &mut arb_state,
                        &ctx,
                        &mut do_transfer,
                        &mut do_balance,
                    ) {
                        tracing::warn!(
                            target: "arb::executor",
                            error = %e,
                            "internal tx processing failed"
                        );
                        tx_err = Some(e);
                    }

                    if is_start_block {
                        self.load_state_params(&arb_state);

                        // Refresh L1 block hashes cache after StartBlock.
                        // StartBlock calls record_new_l1_block which may add
                        // gap hashes that weren't in the initial pre-population.
                        if let Ok(l1_block_number) =
                            arb_state.blockhashes.l1_block_number()
                        {
                            let lower = l1_block_number.saturating_sub(256);
                            let state_ref = unsafe { &mut *state_ptr };
                            for n in lower..l1_block_number {
                                if let Ok(Some(hash)) =
                                    arb_state.blockhashes.block_hash(n)
                                {
                                    state_ref.block_hashes.insert(n, hash);
                                }
                            }
                        }
                    }
                }
            }

            // Internal txs end immediately — no EVM execution.
            self.pending_tx = Some(PendingArbTx {
                sender,
                tx_gas_limit: 0,
                arb_tx_type: Some(ArbTxType::ArbitrumInternalTx),
                has_poster_costs: false,
                poster_gas: 0,
                evm_gas_used: 0,

                charged_multi_gas: MultiGas::default(),
                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                retry_context: None,
            });

            // Internal tx errors are fatal — abort block production.
            if let Some(err) = tx_err {
                return Err(BlockExecutionError::msg(
                    format!("failed to apply internal transaction: {err}"),
                ));
            }

            return Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result: ExecutionResult::Success {
                        reason: revm::context::result::SuccessReason::Return,
                        gas_used: 0,
                        gas_refunded: 0,
                        output: revm::context::result::Output::Call(
                            alloy_primitives::Bytes::new(),
                        ),
                        logs: Vec::new(),
                    },
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            });
        }

        // Deposit txs: mint to sender, transfer to recipient, end immediately.
        // No EVM execution — the value transfer is the entire transaction.
        if is_arb_deposit {
            let value = recovered.tx().value();
            let mut to = match recovered.tx().kind() {
                TxKind::Call(addr) => addr,
                TxKind::Create => {
                    return Err(BlockExecutionError::msg("deposit tx has no To address"));
                }
            };
            let tx_type = recovered.tx().tx_type();
            let tx_hash = recovered.tx().trie_hash();

            // Check if this deposit is in the on-chain filter.
            // Deposits return endTxNow=true so RevertedTxHook is never reached;
            // we must check here instead.
            let mut is_filtered = false;
            {
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                if let Ok(arb_state) =
                    ArbosState::open(state_ptr, SystemBurner::new(None, false))
                {
                    if arb_state.filtered_transactions.is_filtered_free(tx_hash) {
                        if let Ok(recipient) =
                            arb_state.filtered_funds_recipient_or_default()
                        {
                            to = recipient;
                        }
                        is_filtered = true;
                    }
                }
            }

            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            // Mint deposit value to sender, then transfer to recipient.
            mint_balance(db, sender, value);
            transfer_balance(db, sender, to, value);
            self.touched_accounts.insert(sender);
            self.touched_accounts.insert(to);

            // Track deposit for balance delta verification.
            let value_i128: i128 = value.try_into().unwrap_or(i128::MAX);
            self.expected_balance_delta = self.expected_balance_delta.saturating_add(value_i128);

            self.pending_tx = Some(PendingArbTx {
                sender,
                tx_gas_limit: 0,
                arb_tx_type: Some(ArbTxType::ArbitrumDepositTx),
                has_poster_costs: false,
                poster_gas: 0,
                evm_gas_used: 0,

                charged_multi_gas: MultiGas::default(),
                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                retry_context: None,
            });

            // Filtered deposits produce a failed receipt (status=0) via
            // ErrFilteredTx. The state changes (mint + redirected transfer)
            // are still committed.
            let result = if is_filtered {
                ExecutionResult::Revert {
                    gas_used: 0,
                    output: alloy_primitives::Bytes::from("filtered transaction"),
                }
            } else {
                ExecutionResult::Success {
                    reason: revm::context::result::SuccessReason::Return,
                    gas_used: 0,
                    gas_refunded: 0,
                    output: revm::context::result::Output::Call(
                        alloy_primitives::Bytes::new(),
                    ),
                    logs: Vec::new(),
                }
            };

            return Ok(EthTxResult {
                result: revm::context::result::ResultAndState {
                    result,
                    state: Default::default(),
                },
                blob_gas_used: 0,
                tx_type,
            });
        }

        // --- SubmitRetryable: skip EVM, handle fees/escrow/ticket creation ---
        if is_submit_retryable {
            if let Some(info) = recovered.tx().submit_retryable_info() {
                let ticket_id = recovered.tx().trie_hash();
                let tx_type = recovered.tx().tx_type();
                return self.execute_submit_retryable(ticket_id, tx_type, info);
            }
        }

        // --- RetryTx pre-processing: escrow transfer and prepaid gas ---
        let mut retry_context = None;
        if is_retry_tx {
            if let Some(info) = recovered.tx().retry_tx_info() {
                let block = self.inner.evm().block();
                let current_time = revm::context::Block::timestamp(block).to::<u64>();
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;

                // Open the retryable ticket.
                if let Ok(arb_state) =
                    ArbosState::open(state_ptr, SystemBurner::new(None, false))
                {
                    let retryable = arb_state
                        .retryable_state
                        .open_retryable(info.ticket_id, current_time);

                    match retryable {
                        Ok(Some(_)) => {
                            // Transfer call value from escrow to sender.
                            let escrow = retryables::retryable_escrow_address(info.ticket_id);
                            let value = recovered.tx().value();

                            // Go's TransferBalance calls CreateZombieIfDeleted(from)
                            // when amount == 0 on pre-Stylus ArbOS.
                            if value.is_zero()
                                && self.arb_ctx.arbos_version
                                    < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
                            {
                                create_zombie_if_deleted(
                                    db, escrow, &self.finalise_deleted,
                                    &mut self.zombie_accounts,
                                    &mut self.touched_accounts,
                                );
                            }

                            if !try_transfer_balance(db, escrow, sender, value) {
                                // Escrow has insufficient funds — abort the retry tx.
                                let tx_type = recovered.tx().tx_type();
                                self.pending_tx = Some(PendingArbTx {
                                    sender,
                                    tx_gas_limit: 0,
                                    arb_tx_type: Some(ArbTxType::ArbitrumRetryTx),
                                    has_poster_costs: false,
                                    poster_gas: 0,
                                    evm_gas_used: 0,
                    
                                    charged_multi_gas: MultiGas::default(),
                                    gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                                    retry_context: None,
                                });
                                return Ok(EthTxResult {
                                    result: revm::context::result::ResultAndState {
                                        result: ExecutionResult::Revert {
                                            gas_used: 0,
                                            output: alloy_primitives::Bytes::new(),
                                        },
                                        state: Default::default(),
                                    },
                                    blob_gas_used: 0,
                                    tx_type,
                                });
                            }

                            // Track escrow transfer addresses.
                            if !value.is_zero() {
                                self.zombie_accounts.remove(&escrow);
                            }
                            self.zombie_accounts.remove(&sender);
                            self.touched_accounts.insert(escrow);
                            self.touched_accounts.insert(sender);

                            // Mint prepaid gas to sender.
                            let prepaid = self.arb_ctx.basefee
                                .saturating_mul(U256::from(tx_gas_limit));
                            mint_balance(db, sender, prepaid);

                            // Set retry context for end-tx processing.
                            if let Some(hooks) = self.arb_hooks.as_mut() {
                                hooks.tx_proc.prepare_retry_tx(
                                    info.ticket_id,
                                    info.refund_to,
                                );
                            }

                            retry_context = Some(PendingRetryContext {
                                ticket_id: info.ticket_id,
                                refund_to: info.refund_to,
                                gas_fee_cap: info.gas_fee_cap,
                                max_refund: info.max_refund,
                                submission_fee_refund: info.submission_fee_refund,
                                call_value: recovered.tx().value(),
                            });
                        }
                        Ok(None) => {
                            // Retryable expired or not found — endTxNow=true.
                            let tx_type = recovered.tx().tx_type();
                            self.pending_tx = Some(PendingArbTx {
                                sender,
                                tx_gas_limit: 0,
                                arb_tx_type: Some(ArbTxType::ArbitrumRetryTx),
                                has_poster_costs: false,
                                poster_gas: 0,
                                evm_gas_used: 0,
                
                                charged_multi_gas: MultiGas::default(),
                                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                                retry_context: None,
                            });
                            let err_msg = format!(
                                "retryable ticket {} not found",
                                info.ticket_id,
                            );
                            return Ok(EthTxResult {
                                result: revm::context::result::ResultAndState {
                                    result: ExecutionResult::Revert {
                                        gas_used: 0,
                                        output: alloy_primitives::Bytes::from(
                                            err_msg.into_bytes(),
                                        ),
                                    },
                                    state: Default::default(),
                                },
                                blob_gas_used: 0,
                                tx_type,
                            });
                        }
                        Err(_) => {
                            // State error opening retryable — endTxNow=true.
                            let tx_type = recovered.tx().tx_type();
                            self.pending_tx = Some(PendingArbTx {
                                sender,
                                tx_gas_limit: 0,
                                arb_tx_type: Some(ArbTxType::ArbitrumRetryTx),
                                has_poster_costs: false,
                                poster_gas: 0,
                                evm_gas_used: 0,
                
                                charged_multi_gas: MultiGas::default(),
                                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                                retry_context: None,
                            });
                            return Ok(EthTxResult {
                                result: revm::context::result::ResultAndState {
                                    result: ExecutionResult::Revert {
                                        gas_used: 0,
                                        output: alloy_primitives::Bytes::from(
                                            format!(
                                                "error opening retryable {}",
                                                info.ticket_id,
                                            )
                                            .into_bytes(),
                                        ),
                                    },
                                    state: Default::default(),
                                },
                                blob_gas_used: 0,
                                tx_type,
                            });
                        }
                    }
                }
            }
        }

        // --- Poster cost and gas limiting ---

        let mut poster_gas = 0u64;
        let mut compute_hold_gas = 0u64;
        let calldata_units = if has_poster_costs {
            let tx_bytes = recovered.tx().encoded_2718();
            let (_poster_cost, units) = l1_pricing::compute_poster_cost_standalone(
                &tx_bytes,
                self.arb_ctx.coinbase,
                self.arb_ctx.l1_price_per_unit,
                self.arb_ctx.brotli_compression_level,
            );

            if let Some(hooks) = self.arb_hooks.as_mut() {
                let base_fee = self.arb_ctx.basefee;
                hooks.tx_proc.poster_gas =
                    compute_poster_gas(_poster_cost, base_fee, false, self.arb_ctx.min_base_fee);
                hooks.tx_proc.poster_fee =
                    base_fee.saturating_mul(U256::from(hooks.tx_proc.poster_gas));
                poster_gas = hooks.tx_proc.poster_gas;


            }

            units
        } else {
            0
        };

        // Compute hold gas: clamp gas available for EVM execution to the
        // per-block (< v50) or per-tx (>= v50) gas limit. Applies to ALL
        // non-endTxNow txs (including retry txs with poster_gas=0), as the
        // GasChargingHook runs for every tx that enters the EVM.
        if let Some(hooks) = self.arb_hooks.as_mut() {
            if !hooks.is_eth_call {
                let intrinsic_estimate = estimate_intrinsic_gas(recovered.tx());
                let gas_after_intrinsic =
                    tx_gas_limit.saturating_sub(intrinsic_estimate);
                let gas_after_poster =
                    gas_after_intrinsic.saturating_sub(poster_gas);

                let max_compute = if hooks.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_50 {
                    hooks.per_block_gas_limit
                } else {
                    hooks.per_tx_gas_limit.saturating_sub(intrinsic_estimate)
                };

                if max_compute > 0 && gas_after_poster > max_compute {
                    compute_hold_gas = gas_after_poster - max_compute;
                    hooks.tx_proc.compute_hold_gas = compute_hold_gas;
                }
            }
        }

        // ArbOS < 50: reject user txs whose compute gas exceeds block gas left,
        // but always allow the first user tx through (userTxsProcessed > 0).
        // ArbOS >= 50 uses per-tx gas limit clamping (compute_hold_gas) instead.
        // computeGas is clamped to at least TxGas before this check.
        if is_user_tx
            && self.arb_ctx.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_50
            && self.user_txs_processed > 0
        {
            const TX_GAS: u64 = 21_000;
            let compute_gas = tx_gas_limit.saturating_sub(poster_gas).max(TX_GAS);
            if compute_gas > self.block_gas_left {
                return Err(BlockExecutionError::msg("block gas limit reached"));
            }
        }

        // Add calldata units to L1 pricing state BEFORE EVM execution
        // (before EVM execution, during gas charging).
        if calldata_units > 0 {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let state_ptr: *mut State<DB> = db as *mut State<DB>;
            if let Ok(arb_state) = ArbosState::open(state_ptr, SystemBurner::new(None, false)) {
                let _ = arb_state
                    .l1_pricing_state
                    .add_to_units_since_update(calldata_units);
            }
        }

        // Reduce the gas the EVM sees by poster_gas and compute_hold_gas.
        let mut tx_env = tx_env;
        let gas_deduction = poster_gas.saturating_add(compute_hold_gas);
        if gas_deduction > 0 {
            let current = revm::context_interface::Transaction::gas_limit(&tx_env);
            tx_env.set_gas_limit(current.saturating_sub(gas_deduction));
        }

        // --- RevertedTxHook: check for pre-recorded reverted or filtered txs ---
        // Called after gas charging but before EVM execution.
        {
            use arbos::tx_processor::RevertedTxAction;

            // Get tx hash for filtered check.
            let tx_hash = recovered.tx().trie_hash();

            // Check if tx is in the filtered transactions list.
            let is_filtered = {
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                ArbosState::open(state_ptr, SystemBurner::new(None, false))
                    .ok()
                    .map(|s| s.filtered_transactions.is_filtered_free(tx_hash))
                    .unwrap_or(false)
            };

            if let Some(hooks) = self.arb_hooks.as_ref() {
                let action = hooks.tx_proc.reverted_tx_hook(
                    Some(tx_hash),
                    None, // pre_recorded_gas: sequencer-specific, not used in state machine
                    is_filtered,
                );

                match action {
                    RevertedTxAction::PreRecordedRevert { gas_to_consume } => {
                        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                        increment_nonce(db, sender);
                        self.touched_accounts.insert(sender);
                        let gas_used = poster_gas + gas_to_consume;
                        let charged_multi_gas = MultiGas::l1_calldata_gas(poster_gas)
                            .saturating_add(MultiGas::computation_gas(gas_to_consume));
                        self.pending_tx = Some(PendingArbTx {
                            sender,
                            tx_gas_limit,
                            arb_tx_type,
                            has_poster_costs,
                            poster_gas,
                            evm_gas_used: 0,
                            charged_multi_gas,
                            gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                            retry_context,
                        });
                        return Ok(EthTxResult {
                            result: revm::context::result::ResultAndState {
                                result: ExecutionResult::Revert {
                                    gas_used,
                                    output: alloy_primitives::Bytes::new(),
                                },
                                state: Default::default(),
                            },
                            blob_gas_used: 0,
                            tx_type: envelope_tx_type,
                        });
                    }
                    RevertedTxAction::FilteredTx => {
                        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                        increment_nonce(db, sender);
                        self.touched_accounts.insert(sender);
                        // Consume all remaining gas.
                        let gas_remaining = tx_gas_limit
                            .saturating_sub(poster_gas)
                            .saturating_sub(compute_hold_gas);
                        let gas_used = tx_gas_limit;
                        let charged_multi_gas = MultiGas::l1_calldata_gas(poster_gas)
                            .saturating_add(MultiGas::computation_gas(gas_remaining));
                        self.pending_tx = Some(PendingArbTx {
                            sender,
                            tx_gas_limit,
                            arb_tx_type,
                            has_poster_costs,
                            poster_gas,
                            evm_gas_used: 0,
                            charged_multi_gas,
                            gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                            retry_context,
                        });
                        return Ok(EthTxResult {
                            result: revm::context::result::ResultAndState {
                                result: ExecutionResult::Revert {
                                    gas_used,
                                    output: alloy_primitives::Bytes::from(
                                        "filtered transaction".as_bytes(),
                                    ),
                                },
                                state: Default::default(),
                            },
                            blob_gas_used: 0,
                            tx_type: envelope_tx_type,
                        });
                    }
                    RevertedTxAction::None => {}
                }
            }
        }

        // --- Execute via inner EVM executor ---

        // Save the original gas price before tip drop for upfront balance check.
        // The balance check uses GasFeeCap (full gas price), not the
        // effective gas price after tip drop.
        let upfront_gas_price: u128 = revm::context_interface::Transaction::gas_price(&tx_env);

        // Drop the priority fee tip: cap gas price to the base fee.
        // In Arbitrum, fees go to network/infra accounts via EndTxHook, not to coinbase.
        // Without this, revm's reward_beneficiary sends the tip to coinbase.
        let should_drop_tip = self.arb_hooks.as_ref()
            .map(|h| h.drop_tip())
            .unwrap_or(false);
        if should_drop_tip {
            let base_fee: u128 = self.arb_ctx.basefee.try_into().unwrap_or(u128::MAX);
            if upfront_gas_price > base_fee {
                tx_env.set_gas_price(base_fee);
                tx_env.set_gas_priority_fee(Some(0));
            }
        }

        // Write the poster fee to a scratch storage slot so the ArbGasInfo
        // precompile can read it via GetCurrentTxL1Fees during EVM execution.
        {
            use arb_precompiles::storage_slot::current_tx_poster_fee_slot;
            let poster_fee_val = self.arb_hooks.as_ref()
                .map(|h| h.tx_proc.poster_fee)
                .unwrap_or(U256::ZERO);
            arb_storage::write_arbos_storage(
                self.inner.evm_mut().db_mut(),
                current_tx_poster_fee_slot(),
                poster_fee_val,
            );
        }

        // Write the current retryable ticket ID to a scratch slot so the
        // Redeem precompile can reject self-modification during retry execution.
        {
            use arb_precompiles::storage_slot::{current_redeemer_slot, current_retryable_slot};
            let retryable_id = retry_context
                .as_ref()
                .map(|ctx| U256::from_be_bytes(ctx.ticket_id.0))
                .unwrap_or(U256::ZERO);
            arb_storage::write_arbos_storage(
                self.inner.evm_mut().db_mut(),
                current_retryable_slot(),
                retryable_id,
            );
            // Write the current redeemer (refund_to) so GetCurrentRedeemer can read it.
            let redeemer = retry_context
                .as_ref()
                .map(|ctx| U256::from_be_bytes(B256::left_padding_from(ctx.refund_to.as_slice()).0))
                .unwrap_or(U256::ZERO);
            arb_storage::write_arbos_storage(
                self.inner.evm_mut().db_mut(),
                current_redeemer_slot(),
                redeemer,
            );
        }

        // Helper: roll back pre-execution state writes when a tx is rejected.
        // Clears scratch slots and undoes calldata units addition.
        let rollback_pre_exec_state = |this: &mut Self, units: u64| {
            use arb_precompiles::storage_slot::{
                current_redeemer_slot, current_retryable_slot,
                current_tx_poster_fee_slot,
            };
            let db: &mut State<DB> = this.inner.evm_mut().db_mut();
            arb_storage::write_arbos_storage(db, current_tx_poster_fee_slot(), U256::ZERO);
            arb_storage::write_arbos_storage(db, current_retryable_slot(), U256::ZERO);
            arb_storage::write_arbos_storage(db, current_redeemer_slot(), U256::ZERO);
            if units > 0 {
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                if let Ok(arb_state) = ArbosState::open(
                    state_ptr,
                    SystemBurner::new(None, false),
                ) {
                    let _ = arb_state
                        .l1_pricing_state
                        .subtract_from_units_since_update(units);
                }
            }
        };

        // Manual balance and nonce validation for user txs. Revm's checks are
        // disabled globally in arb_cfg_env (internal/deposit/retryable txs need
        // to bypass them). User txs from the delayed inbox may have insufficient
        // funds or wrong nonces and must be rejected here.
        if is_user_tx {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let account = db.load_cache_account(sender).ok().and_then(|a| a.account_info());
            let sender_balance = account.as_ref().map(|a| a.balance).unwrap_or(U256::ZERO);
            let sender_nonce = account.as_ref().map(|a| a.nonce).unwrap_or(0);

            // Nonce check: tx nonce must match sender's current nonce.
            let tx_nonce = revm::context_interface::Transaction::nonce(&tx_env);
            if tx_nonce != sender_nonce {
                rollback_pre_exec_state(self, calldata_units);
                return Err(BlockExecutionError::msg(format!(
                    "nonce mismatch: address {sender} tx nonce {tx_nonce} != state nonce {sender_nonce}"
                )));
            }

            // Balance check: sender must cover gas * upfront_gas_price + value.
            // Uses the original gas price (before tip drop) since the canonical
            // state transition uses GasFeeCap for the balance check.
            let gas_cost = U256::from(tx_gas_limit) * U256::from(upfront_gas_price);
            let tx_value = revm::context_interface::Transaction::value(&tx_env);
            let total_cost = gas_cost.saturating_add(tx_value);
            if sender_balance < total_cost {
                rollback_pre_exec_state(self, calldata_units);
                return Err(BlockExecutionError::msg(format!(
                    "insufficient funds: address {sender} have {sender_balance} want {total_cost}"
                )));
            }
        }

        // Fix nonce for retry and contract txs: skipNonceChecks() skips
        // the preCheck nonce validation but the nonce is still incremented in
        // TransitionDb for non-CREATE calls. Override the tx_env nonce to
        // match the sender's current state nonce so revm increments from the
        // right value.
        if is_retry_tx || is_contract_tx {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let sender_nonce = db.load_cache_account(sender)
                .map(|a| a.account_info().map(|i| i.nonce).unwrap_or(0))
                .unwrap_or(0);
            tx_env.set_nonce(sender_nonce);
        }

        let mut output = match self.inner.execute_transaction_without_commit((tx_env, recovered)) {
            Ok(o) => o,
            Err(e) => {
                rollback_pre_exec_state(self, calldata_units);
                return Err(e);
            }
        };

        // Capture gas_used as reported by reth's EVM (before our adjustments).
        // This represents the gas cost reth already deducted from the sender.
        let evm_gas_used = output.result.result.gas_used();

        // Adjust gas_used to include poster_gas only.
        // poster_gas was deducted from gas_limit before EVM execution so reth's
        // reported gas_used doesn't include it. Adding it back produces correct
        // receipt gas_used. compute_hold_gas is NOT added: it is returned via
        // calcHeldGasRefund() before computing final gasUsed, and
        // NonRefundableGas() excludes it from the refund denominator.
        if poster_gas > 0 {
            adjust_result_gas_used(&mut output.result.result, poster_gas);
        }

        // Scan execution logs for RedeemScheduled events (manual redeem path).
        // The ArbRetryableTx.Redeem precompile emits this event; we discover it
        // here and schedule the retry tx via the ScheduledTxes() mechanism.
        if let ExecutionResult::Success { ref logs, .. } = output.result.result {
            let redeem_topic = arb_precompiles::redeem_scheduled_topic();
            let precompile_addr = arb_precompiles::ARBRETRYABLETX_ADDRESS;

            for log in logs {
                if log.address != precompile_addr { continue; }
                if log.topics().is_empty() || log.topics()[0] != redeem_topic { continue; }
                if log.topics().len() < 4 || log.data.data.len() < 128 { continue; }

                let ticket_id = log.topics()[1];
                let seq_num_bytes = log.topics()[3];
                let nonce = u64::from_be_bytes(seq_num_bytes.0[24..32].try_into().unwrap_or([0u8; 8]));
                let data = &log.data.data;
                let donated_gas = U256::from_be_slice(&data[0..32]).to::<u64>();
                let gas_donor = Address::from_slice(&data[44..64]);
                let max_refund = U256::from_be_slice(&data[64..96]);
                let submission_fee_refund = U256::from_be_slice(&data[96..128]);

                // Open the retryable and construct the retry tx.
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                let current_time = {
                    let block = self.inner.evm().block();
                    revm::context::Block::timestamp(block).to::<u64>()
                };
                if let Ok(arb_state) = ArbosState::open(state_ptr, SystemBurner::new(None, false)) {
                    if let Ok(Some(retryable)) = arb_state.retryable_state.open_retryable(
                        ticket_id,
                        current_time,
                    ) {
                        if let Ok(retry_tx) = retryable.make_tx(
                            U256::from(self.arb_ctx.chain_id),
                            nonce,
                            self.arb_ctx.basefee,
                            donated_gas,
                            ticket_id,
                            gas_donor,
                            max_refund,
                            submission_fee_refund,
                        ) {
                            if let Some(hooks) = self.arb_hooks.as_mut() {
                                let mut encoded = Vec::new();
                                encoded.push(ArbTxType::ArbitrumRetryTx.as_u8());
                                alloy_rlp::Encodable::encode(&retry_tx, &mut encoded);
                                hooks.tx_proc.scheduled_txs.push(encoded);
                            }
                        }
                    }

                    // Shrink the backlog by the donated gas amount.
                    let _ = arb_state.l2_pricing_state.shrink_backlog(
                        donated_gas,
                        MultiGas::default(),
                    );
                }
            }
        }

        // Store per-tx state for fee distribution in commit_transaction.
        // Build multi-gas: L1 calldata from poster costs + EVM execution as computation.
        // Note: revm doesn't track per-resource gas dimensions, so all EVM gas
        // (intrinsic + opcode execution) is classified as computation gas.
        let charged_multi_gas = MultiGas::l1_calldata_gas(poster_gas)
            .saturating_add(MultiGas::computation_gas(evm_gas_used));

        self.pending_tx = Some(PendingArbTx {
            sender,
            tx_gas_limit,
            arb_tx_type,
            has_poster_costs,
            poster_gas,
            evm_gas_used,
            charged_multi_gas,
            gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
            retry_context,
        });

        Ok(output)
    }

    fn commit_transaction(&mut self, output: Self::Result) -> Result<u64, BlockExecutionError> {
        // Extract info needed for fee distribution before the output is consumed.
        let pending = self.pending_tx.take();
        let gas_used_total = output.result.result.gas_used();
        let success = matches!(&output.result.result, ExecutionResult::Success { .. });

        // Scan receipt logs for L2→L1 withdrawal events and burn value from ArbSys.
        // Value transferred to the ArbSys address during a withdrawEth call
        // is burned (subtracted from ArbSys balance) after the tx commits.
        let mut withdrawal_value = U256::ZERO;
        if let ExecutionResult::Success { ref logs, .. } = output.result.result {
            let arbsys_addr = arb_precompiles::ARBSYS_ADDRESS;
            let l2_to_l1_tx_topic = keccak256(
                b"L2ToL1Tx(address,address,uint256,uint256,uint256,uint256,uint256,uint256,bytes)",
            );
            for log in logs {
                if log.address == arbsys_addr
                    && !log.data.topics().is_empty()
                    && log.data.topics()[0] == l2_to_l1_tx_topic
                {
                    // L2ToL1Tx data layout: ABI-encoded [caller, arb_block, eth_block, timestamp, callvalue, data]
                    // callvalue is at offset 4*32 = 128 bytes.
                    if log.data.data.len() >= 160 {
                        let callvalue = U256::from_be_slice(&log.data.data[128..160]);
                        withdrawal_value = withdrawal_value.saturating_add(callvalue);
                        let val_i128: i128 = callvalue.try_into().unwrap_or(i128::MAX);
                        self.expected_balance_delta =
                            self.expected_balance_delta.saturating_sub(val_i128);
                    }
                }
            }
        }

        // Capture EVM-modified addresses for dirty tracking before commit consumes output.
        for addr in output.result.state.keys() {
            self.touched_accounts.insert(*addr);
        }

        // Inner executor builds receipt with the adjusted gas_used and commits state.
        let gas_used = self.inner.commit_transaction(output)?;

        // Burn ETH from ArbSys address for L2→L1 withdrawals.
        if !withdrawal_value.is_zero() {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            burn_balance(db, arb_precompiles::ARBSYS_ADDRESS, withdrawal_value);
            self.touched_accounts.insert(arb_precompiles::ARBSYS_ADDRESS);
        }

        // Track poster gas and multi-gas for this receipt (parallel to receipts vector).
        let poster_gas_for_receipt = pending.as_ref().map_or(0, |p| p.poster_gas);
        self.gas_used_for_l1.push(poster_gas_for_receipt);
        let multi_gas_for_receipt = pending.as_ref().map_or(MultiGas::zero(), |p| p.charged_multi_gas);
        self.multi_gas_used.push(multi_gas_for_receipt);

        // --- Post-execution: fee distribution ---
        if let Some(pending) = pending {
            let is_retry = pending.retry_context.is_some();

            // Safety check: gas refund should never exceed gas limit.
            debug_assert!(
                gas_used_total <= pending.tx_gas_limit,
                "gas_used ({gas_used_total}) exceeds gas_limit ({})",
                pending.tx_gas_limit
            );

            // Charge the sender for gas costs that reth's internal buyGas
            // didn't cover. For normal EVM txs, this equals poster_gas
            // (deducted from gas_limit before reth sees it; compute_hold_gas
            // is also deducted but not charged — it is returned via
            // calcHeldGasRefund before final gasUsed). For early-return paths
            // (pre-recorded revert, filtered tx), evm_gas_used is 0 and the
            // sender must pay the full gas_used.
            let sender_extra_gas = gas_used_total
                .saturating_sub(pending.evm_gas_used);
            if sender_extra_gas > 0 {
                let extra_cost = self.arb_ctx.basefee
                    .saturating_mul(U256::from(sender_extra_gas));
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                burn_balance(db, pending.sender, extra_cost);
                self.touched_accounts.insert(pending.sender);
            }

            tracing::warn!(
                target: "arb::backlog",
                has_poster_costs = pending.has_poster_costs,
                retry_context = pending.retry_context.is_some(),
                gas_price_positive = pending.gas_price_positive,
                gas_used = gas_used_total,
                poster_gas = pending.poster_gas,
                arb_tx_type = ?pending.arb_tx_type,
                "commit_transaction branching"
            );

            if let Some(retry_ctx) = pending.retry_context {
                // RetryTx end-of-tx: handle gas refunds, retryable cleanup.
                let gas_left = pending.tx_gas_limit.saturating_sub(gas_used_total);

                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;
                let touched_ptr = &mut self.touched_accounts
                    as *mut std::collections::HashSet<Address>;
                let zombie_ptr = &mut self.zombie_accounts
                    as *mut std::collections::HashSet<Address>;
                let finalise_ptr = &self.finalise_deleted
                    as *const std::collections::HashSet<Address>;
                let arbos_ver = self.arb_ctx.arbos_version;

                // Compute multi-dimensional cost for refund (ArbOS v60+).
                let multi_dimensional_cost = if self.arb_ctx.arbos_version
                    >= arb_chainspec::arbos_version::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS
                {
                    ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        .ok()
                        .and_then(|s| {
                            s.l2_pricing_state
                                .multi_dimensional_price_for_refund(pending.charged_multi_gas)
                                .ok()
                        })
                } else {
                    None
                };

                let result = self.arb_hooks.as_ref().map(|hooks| {
                    hooks.tx_proc.end_tx_retryable(
                        &EndTxRetryableParams {
                            gas_left,
                            gas_used: gas_used_total,
                            effective_base_fee: self.arb_ctx.basefee,
                            from: pending.sender,
                            refund_to: retry_ctx.refund_to,
                            max_refund: retry_ctx.max_refund,
                            submission_fee_refund: retry_ctx.submission_fee_refund,
                            ticket_id: retry_ctx.ticket_id,
                            value: U256::ZERO, // Already transferred in pre-exec
                            success,
                            network_fee_account: self.arb_ctx.network_fee_account,
                            infra_fee_account: self.arb_ctx.infra_fee_account,
                            min_base_fee: self.arb_ctx.min_base_fee,
                            arbos_version: self.arb_ctx.arbos_version,
                            multi_dimensional_cost,
                            block_base_fee: self.arb_ctx.basefee,
                        },
                        |addr, amount| {
                            unsafe {
                                burn_balance(&mut *state_ptr, addr, amount);
                                (*touched_ptr).insert(addr);
                            }
                        },
                        |from, to, amount| {
                            unsafe {
                                if amount.is_zero()
                                    && arbos_ver < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
                                {
                                    create_zombie_if_deleted(
                                        &mut *state_ptr, from, &*finalise_ptr,
                                        &mut *zombie_ptr, &mut *touched_ptr,
                                    );
                                }
                                transfer_balance(&mut *state_ptr, from, to, amount);
                                // Go's SubBalance(from, nonzero) creates a non-zombie
                                // balanceChange entry, breaking zombie protection.
                                if !amount.is_zero() {
                                    (*zombie_ptr).remove(&from);
                                }
                                // Go's AddBalance(to, _) dirts `to`, breaking zombie.
                                (*zombie_ptr).remove(&to);
                                (*touched_ptr).insert(from);
                                (*touched_ptr).insert(to);
                            }
                            Ok(())
                        },
                    )
                });

                if let Some(ref result) = result {
                    tracing::debug!(
                        target: "arb::executor",
                        ticket_id = %retry_ctx.ticket_id,
                        should_delete = result.should_delete_retryable,
                        "Retry EndTxHook result"
                    );
                    if result.should_delete_retryable {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        if let Ok(arb_state) =
                            ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        {
                            let _ = arb_state.retryable_state.delete_retryable(
                                retry_ctx.ticket_id,
                                |from, to, amount| {
                                    unsafe {
                                        if amount.is_zero()
                                            && arbos_ver < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
                                        {
                                            create_zombie_if_deleted(
                                                &mut *state_ptr, from, &*finalise_ptr,
                                                &mut *zombie_ptr, &mut *touched_ptr,
                                            );
                                        }
                                        transfer_balance(&mut *state_ptr, from, to, amount);
                                        if !amount.is_zero() {
                                            (*zombie_ptr).remove(&from);
                                        }
                                        (*zombie_ptr).remove(&to);
                                        (*touched_ptr).insert(from);
                                        (*touched_ptr).insert(to);
                                    }
                                    Ok(())
                                },
                                |addr| {
                                    unsafe { get_balance(&mut *state_ptr, addr) }
                                },
                            );
                        }
                    } else if result.should_return_value_to_escrow {
                        // Failed retry: return call value to escrow.
                        unsafe {
                            if retry_ctx.call_value.is_zero()
                                && arbos_ver < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
                            {
                                create_zombie_if_deleted(
                                    &mut *state_ptr, pending.sender, &*finalise_ptr,
                                    &mut *zombie_ptr, &mut *touched_ptr,
                                );
                            }
                            transfer_balance(
                                &mut *state_ptr,
                                pending.sender,
                                result.escrow_address,
                                retry_ctx.call_value,
                            );
                            // Go's SubBalance(sender, nonzero) breaks zombie on sender.
                            if !retry_ctx.call_value.is_zero() {
                                (*zombie_ptr).remove(&pending.sender);
                            }
                            // Go's AddBalance(escrow, _) breaks zombie on escrow.
                            (*zombie_ptr).remove(&result.escrow_address);
                            (*touched_ptr).insert(pending.sender);
                            (*touched_ptr).insert(result.escrow_address);
                        }
                    }

                    // Grow gas backlog unconditionally for retryable txs.
                    // Unlike normal txs, backlog growth is unconditional here.
                    {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        if let Ok(arb_state) =
                            ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        {
                            let backlog_before = arb_state.l2_pricing_state.gas_backlog().unwrap_or(u64::MAX);
                            let grow_result = arb_state.l2_pricing_state.grow_backlog(
                                result.compute_gas_for_backlog,
                                pending.charged_multi_gas,
                            );
                            let backlog_after = arb_state.l2_pricing_state.gas_backlog().unwrap_or(u64::MAX);
                            tracing::warn!(
                                target: "arb::backlog",
                                gas_for_backlog = result.compute_gas_for_backlog,
                                backlog_before,
                                backlog_after,
                                grow_ok = grow_result.is_ok(),
                                "RetryTx GrowBacklog"
                            );
                        } else {
                            tracing::error!(
                                target: "arb::backlog",
                                "RetryTx GrowBacklog: ArbosState::open FAILED"
                            );
                        }
                    }
                }
            } else if pending.has_poster_costs {
                // Normal tx: fee distribution.
                let gas_left = pending.tx_gas_limit.saturating_sub(gas_used_total);

                let fee_dist = self.arb_hooks.as_ref().map(|hooks| {
                    hooks.compute_end_tx_fees(&EndTxContext {
                        sender: pending.sender,
                        gas_left,
                        gas_used: gas_used_total,
                        gas_price: self.arb_ctx.basefee,
                        base_fee: self.arb_ctx.basefee,
                        tx_type: pending.arb_tx_type
                            .unwrap_or(ArbTxType::ArbitrumLegacyTx),
                        success,
                        refund_to: pending.sender,
                    })
                });

                if let Some(ref dist) = fee_dist {
                    tracing::warn!(
                        target: "arb::backlog",
                        compute_gas_for_backlog = dist.compute_gas_for_backlog,
                        poster_fee = ?dist.poster_fee_amount,
                        "NormalTx fee distribution"
                    );
                    let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                    apply_fee_distribution(db, dist, None);
                    self.touched_accounts.insert(dist.network_fee_account);
                    self.touched_accounts.insert(dist.infra_fee_account);
                    self.touched_accounts.insert(dist.poster_fee_destination);

                    // Multi-dimensional gas refund: if the multi-gas cost is less
                    // than the single-gas cost, refund the difference to the sender.
                    if self.arb_ctx.arbos_version
                        >= arb_chainspec::arbos_version::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS
                    {
                        let total_cost = self.arb_ctx.basefee
                            .saturating_mul(U256::from(gas_used_total));
                        let state_ptr: *mut State<DB> = db as *mut State<DB>;
                        if let Ok(arb_state) =
                            ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        {
                            if let Ok(multi_cost) = arb_state
                                .l2_pricing_state
                                .multi_dimensional_price_for_refund(pending.charged_multi_gas)
                            {
                                if total_cost > multi_cost {
                                    let refund_amount = total_cost.saturating_sub(multi_cost);
                                    transfer_balance(
                                        db,
                                        dist.network_fee_account,
                                        pending.sender,
                                        refund_amount,
                                    );
                                    self.touched_accounts.insert(dist.network_fee_account);
                                    self.touched_accounts.insert(pending.sender);
                                }
                            }
                        }
                    }

                    // Remove poster gas from the L1Calldata dimension: the
                    // poster gas was added during gas charging, but for backlog
                    // growth we only want compute gas in the multi-gas.
                    let used_multi_gas = pending.charged_multi_gas
                        .saturating_sub(MultiGas::l1_calldata_gas(pending.poster_gas));

                    let state_ptr: *mut State<DB> = db as *mut State<DB>;
                    if let Ok(arb_state) =
                        ArbosState::open(state_ptr, SystemBurner::new(None, false))
                    {
                        // Backlog update is skipped when gas price is zero.
                        if pending.gas_price_positive {
                            let backlog_before = arb_state.l2_pricing_state.gas_backlog().unwrap_or(u64::MAX);
                            let grow_result = arb_state.l2_pricing_state.grow_backlog(
                                dist.compute_gas_for_backlog,
                                used_multi_gas,
                            );
                            let backlog_after = arb_state.l2_pricing_state.gas_backlog().unwrap_or(u64::MAX);
                            tracing::warn!(
                                target: "arb::backlog",
                                gas_for_backlog = dist.compute_gas_for_backlog,
                                backlog_before,
                                backlog_after,
                                grow_ok = grow_result.is_ok(),
                                "NormalTx GrowBacklog"
                            );
                        } else {
                            tracing::error!(
                                target: "arb::backlog",
                                "NormalTx: gas_price_positive is FALSE, skipping grow_backlog"
                            );
                        }
                        if !dist.l1_fees_to_add.is_zero() {
                            let _ = arb_state
                                .l1_pricing_state
                                .add_to_l1_fees_available(dist.l1_fees_to_add);
                        }
                    } else {
                        tracing::error!(
                            target: "arb::backlog",
                            "NormalTx: ArbosState::open FAILED for grow_backlog"
                        );
                    }
                }
            }

            // FixRedeemGas (ArbOS >= 11): subtract gas allocated to scheduled
            // retry txs from this tx's gas_used for block rate limiting, since
            // that gas will be accounted for when the retry tx itself executes.
            let mut adjusted_gas_used = gas_used_total;
            if self.arb_ctx.arbos_version
                >= arb_chainspec::arbos_version::ARBOS_VERSION_FIX_REDEEM_GAS
            {
                if let Some(hooks) = self.arb_hooks.as_ref() {
                    for scheduled in &hooks.tx_proc.scheduled_txs {
                        if let Some(retry_gas) = decode_retry_tx_gas(scheduled) {
                            adjusted_gas_used =
                                adjusted_gas_used.saturating_sub(retry_gas);
                        }
                    }
                }
            }

            // Block gas rate limiting: deduct compute gas from block budget.
            const TX_GAS: u64 = 21_000;
            let data_gas = pending.poster_gas;
            let compute_used = if adjusted_gas_used < data_gas {
                TX_GAS
            } else {
                let compute = adjusted_gas_used - data_gas;
                if compute < TX_GAS { TX_GAS } else { compute }
            };
            self.block_gas_left = self.block_gas_left.saturating_sub(compute_used);

            // Track user txs for the ArbOS < 50 first-tx bypass.
            let is_user_tx = !matches!(
                pending.arb_tx_type,
                Some(ArbTxType::ArbitrumInternalTx)
                    | Some(ArbTxType::ArbitrumDepositTx)
                    | Some(ArbTxType::ArbitrumSubmitRetryableTx)
                    | Some(ArbTxType::ArbitrumRetryTx)
            );
            if is_user_tx {
                self.user_txs_processed += 1;
            }

            let _ = is_retry; // suppress unused warning
        }

        // Clear per-tx scratch slots so they don't affect the state root.
        // Canonically these are in-memory fields on TxProcessor, not in storage.
        // We write them to storage so precompiles can read them via sload, but
        // must clear them after each tx to avoid polluting the state trie.
        {
            use arb_precompiles::storage_slot::{
                current_redeemer_slot, current_retryable_slot, current_tx_poster_fee_slot,
            };
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            arb_storage::write_arbos_storage(db, current_tx_poster_fee_slot(), U256::ZERO);
            arb_storage::write_arbos_storage(db, current_retryable_slot(), U256::ZERO);
            arb_storage::write_arbos_storage(db, current_redeemer_slot(), U256::ZERO);
        }

        // Per-tx Finalise: delete empty accounts from cache.
        // Only iterates touched accounts (matching Go's journal.dirties).
        // Accounts merely loaded (e.g. balance check) are not considered.
        {
            let keccak_empty = alloy_primitives::B256::from(alloy_primitives::keccak256(&[]));
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let to_remove: Vec<Address> = self.touched_accounts.drain()
                .filter(|addr| {
                    // Zombie accounts must be preserved even if empty.
                    if self.zombie_accounts.contains(addr) {
                        return false;
                    }
                    if let Some(cached) = db.cache.accounts.get(addr) {
                        if let Some(ref acct) = cached.account {
                            let is_empty = acct.info.nonce == 0
                                && acct.info.balance.is_zero()
                                && acct.info.code_hash == keccak_empty;
                            return is_empty;
                        }
                    }
                    false
                })
                .collect();

            for addr in &to_remove {
                db.cache.accounts.remove(addr);
            }
            self.finalise_deleted.extend(to_remove);
        }

        Ok(gas_used)
    }

    fn finish(
        self,
    ) -> Result<(Self::Evm, BlockExecutionResult<R::Receipt>), BlockExecutionError> {
        // Log if expected balance delta is non-zero (deposits/withdrawals occurred).
        if self.expected_balance_delta != 0 {
            tracing::trace!(
                target: "arb::executor",
                delta = self.expected_balance_delta,
                "expected balance delta from deposits/withdrawals"
            );
        }
        // Skip inner.finish() to avoid Ethereum block rewards.
        // Arbitrum has no block rewards (no PoW/PoS mining).
        // Directly extract the EVM and receipts instead.
        let mut result = BlockExecutionResult {
            receipts: self.inner.receipts,
            requests: Default::default(),
            gas_used: self.inner.gas_used,
            blob_gas_used: self.inner.blob_gas_used,
        };
        // Set Arbitrum-specific fields on each receipt from tracking vectors.
        for (i, receipt) in result.receipts.iter_mut().enumerate() {
            if let Some(&l1_gas) = self.gas_used_for_l1.get(i) {
                arb_primitives::SetArbReceiptFields::set_gas_used_for_l1(receipt, l1_gas);
            }
            if let Some(&multi_gas) = self.multi_gas_used.get(i) {
                arb_primitives::SetArbReceiptFields::set_multi_gas_used(receipt, multi_gas);
            }
        }
        Ok((self.inner.evm, result))
    }

    fn set_state_hook(&mut self, hook: Option<Box<dyn OnStateHook>>) {
        self.inner.set_state_hook(hook);
    }

    fn evm_mut(&mut self) -> &mut Self::Evm {
        self.inner.evm_mut()
    }

    fn evm(&self) -> &Self::Evm {
        self.inner.evm()
    }

    fn receipts(&self) -> &[Self::Receipt] {
        self.inner.receipts()
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Adjust gas_used in an `ExecutionResult` by adding extra gas.
///
/// Used to account for poster gas (L1 data cost) which is deducted before
/// EVM execution but must be reflected in the receipt's gas_used.
fn adjust_result_gas_used<H>(result: &mut ExecutionResult<H>, extra_gas: u64) {
    match result {
        ExecutionResult::Success { gas_used, .. } => *gas_used = gas_used.saturating_add(extra_gas),
        ExecutionResult::Revert { gas_used, .. } => *gas_used = gas_used.saturating_add(extra_gas),
        ExecutionResult::Halt { gas_used, .. } => *gas_used = gas_used.saturating_add(extra_gas),
    }
}

/// Mint balance to an address in the EVM state.
/// Modifies cache directly; net effect captured by augment_bundle_from_cache.
fn mint_balance<DB: Database>(state: &mut State<DB>, address: Address, amount: U256) {
    let _ = state.load_cache_account(address);
    if let Some(cache_acct) = state.cache.accounts.get_mut(&address) {
        if let Some(ref mut acct) = cache_acct.account {
            acct.info.balance = acct.info.balance.saturating_add(amount);
        } else {
            cache_acct.account = Some(revm_database::states::plain_account::PlainAccount {
                info: revm_state::AccountInfo {
                    balance: amount,
                    ..Default::default()
                },
                storage: Default::default(),
            });
        }
    }
}

/// Burn balance from an address in the EVM state.
/// Modifies cache directly; net effect captured by augment_bundle_from_cache.
fn burn_balance<DB: Database>(state: &mut State<DB>, address: Address, amount: U256) {
    let _ = state.load_cache_account(address);
    if let Some(cache_acct) = state.cache.accounts.get_mut(&address) {
        if let Some(ref mut acct) = cache_acct.account {
            acct.info.balance = acct.info.balance.saturating_sub(amount);
        }
    }
}

/// Increment the nonce of an account in the EVM state.
///
/// Directly modifies the cache without creating transitions.
fn increment_nonce<DB: Database>(state: &mut State<DB>, address: Address) {
    let _ = state.load_cache_account(address);
    if let Some(cache_acct) = state.cache.accounts.get_mut(&address) {
        if let Some(ref mut acct) = cache_acct.account {
            acct.info.nonce += 1;
        }
    }
}

/// Read the balance of an account in the EVM state.
fn get_balance<DB: Database>(state: &mut State<DB>, address: Address) -> U256 {
    match revm::Database::basic(state, address) {
        Ok(Some(info)) => info.balance,
        _ => U256::ZERO,
    }
}

/// Transfer balance between two addresses. Atomic: skipped on insufficient funds.
fn transfer_balance<DB: Database>(
    state: &mut State<DB>,
    from: Address,
    to: Address,
    amount: U256,
) {
    if amount.is_zero() {
        ensure_account_exists(state, from);
        ensure_account_exists(state, to);
        return;
    }
    // No from == to early return — Go always does SubBalance + AddBalance
    // independently even when from == to. This ensures the account gets
    // dirtied in the state trie consistently.
    let balance = get_balance(state, from);
    if balance < amount {
        tracing::warn!(
            target: "arb::executor",
            %from, %to, %amount, %balance,
            "transfer_balance: insufficient funds, skipping"
        );
        return;
    }
    burn_balance(state, from, amount);
    mint_balance(state, to, amount);
}

/// Ensure an account exists in the state cache, creating an empty one if needed.
/// Matches Go's `getOrNewStateObject` — guarantees the account is dirty in cache.
fn ensure_account_exists<DB: Database>(state: &mut State<DB>, addr: Address) {
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        if cached.account.is_none() {
            cached.account = Some(revm_database::states::plain_account::PlainAccount {
                info: revm_state::AccountInfo::default(),
                storage: Default::default(),
            });
            cached.status = revm_database::AccountStatus::InMemoryChange;
        }
    }
}


/// Re-create an empty account that was deleted by per-tx Finalise.
/// Matches Go's `CreateZombieIfDeleted`: if `addr` was removed by Finalise
/// (present in `finalise_deleted`) and no longer in cache, create a zombie.
/// Go calls this for `from` in TransferBalance when amount == 0 and
/// ArbOS version < Stylus.
fn create_zombie_if_deleted<DB: Database>(
    state: &mut State<DB>,
    addr: Address,
    finalise_deleted: &std::collections::HashSet<Address>,
    zombie_accounts: &mut std::collections::HashSet<Address>,
    touched_accounts: &mut std::collections::HashSet<Address>,
) {
    let _ = state.load_cache_account(addr);
    let account_missing = state.cache.accounts.get(&addr)
        .map_or(true, |c| c.account.is_none());
    if account_missing && finalise_deleted.contains(&addr) {
        if let Some(cached) = state.cache.accounts.get_mut(&addr) {
            cached.account = Some(revm_database::states::plain_account::PlainAccount {
                info: revm_state::AccountInfo::default(),
                storage: Default::default(),
            });
            cached.status = revm_database::AccountStatus::InMemoryChange;
        }
        zombie_accounts.insert(addr);
        touched_accounts.insert(addr);
    }
}

/// Transfer balance with balance check. Returns false if sender has
/// insufficient funds (no state changes in that case).
fn try_transfer_balance<DB: Database>(
    state: &mut State<DB>,
    from: Address,
    to: Address,
    amount: U256,
) -> bool {
    if amount.is_zero() {
        ensure_account_exists(state, from);
        ensure_account_exists(state, to);
        return true;
    }
    if get_balance(state, from) < amount {
        return false;
    }
    burn_balance(state, from, amount);
    mint_balance(state, to, amount);
    true
}

/// Apply a computed fee distribution to the EVM state.
fn apply_fee_distribution<DB: Database>(
    state: &mut State<DB>,
    dist: &EndTxFeeDistribution,
    l1_pricing: Option<&l1_pricing::L1PricingState<DB>>,
) {
    mint_balance(state, dist.network_fee_account, dist.network_fee_amount);
    mint_balance(state, dist.infra_fee_account, dist.infra_fee_amount);
    mint_balance(state, dist.poster_fee_destination, dist.poster_fee_amount);

    if !dist.l1_fees_to_add.is_zero() {
        if let Some(l1_state) = l1_pricing {
            let _ = l1_state.add_to_l1_fees_available(dist.l1_fees_to_add);
        }
    }

    tracing::trace!(
        target: "arb::executor",
        network_fee = %dist.network_fee_amount,
        infra_fee = %dist.infra_fee_amount,
        poster_fee = %dist.poster_fee_amount,
        poster_dest = %dist.poster_fee_destination,
        l1_fees_added = %dist.l1_fees_to_add,
        backlog_gas = dist.compute_gas_for_backlog,
        "applied fee distribution"
    );
}

/// Estimate intrinsic gas for a transaction.
///
/// Matches geth's `IntrinsicGas()`: base 21000 + calldata cost + create cost +
/// access list cost + EIP-3860 initcode cost.
fn estimate_intrinsic_gas(tx: &impl Transaction) -> u64 {
    const TX_GAS: u64 = 21_000;
    const TX_CREATE_GAS: u64 = 32_000;
    const TX_DATA_ZERO_GAS: u64 = 4;
    const TX_DATA_NON_ZERO_GAS: u64 = 16;
    const TX_ACCESS_LIST_ADDRESS_GAS: u64 = 2400;
    const TX_ACCESS_LIST_STORAGE_KEY_GAS: u64 = 1900;
    const INIT_CODE_WORD_GAS: u64 = 2;

    let is_create = tx.to().is_none();

    let mut gas = TX_GAS;
    if is_create {
        gas += TX_CREATE_GAS;
    }

    let data = tx.input();

    // Calldata cost.
    let data_gas: u64 = data
        .iter()
        .map(|&b| if b == 0 { TX_DATA_ZERO_GAS } else { TX_DATA_NON_ZERO_GAS })
        .sum();
    gas = gas.saturating_add(data_gas);

    // EIP-2930: access list cost.
    if let Some(access_list) = tx.access_list() {
        for item in access_list.iter() {
            gas = gas.saturating_add(TX_ACCESS_LIST_ADDRESS_GAS);
            gas = gas.saturating_add(
                (item.storage_keys.len() as u64).saturating_mul(TX_ACCESS_LIST_STORAGE_KEY_GAS),
            );
        }
    }

    // EIP-3860: initcode word cost for CREATE txs (Shanghai+).
    if is_create && !data.is_empty() {
        let words = (data.len() as u64 + 31) / 32;
        gas = gas.saturating_add(words.saturating_mul(INIT_CODE_WORD_GAS));
    }

    gas
}

/// Decode delayed_messages_read (bytes 32-39) and L2 block number (bytes 40-47)
/// from the extra_data field passed through EthBlockExecutionCtx.
fn decode_extra_fields(extra_bytes: &[u8]) -> (u64, u64) {
    let delayed = if extra_bytes.len() >= 40 {
        let mut buf = [0u8; 8];
        buf.copy_from_slice(&extra_bytes[32..40]);
        u64::from_be_bytes(buf)
    } else {
        0
    };
    let l2_block = if extra_bytes.len() >= 48 {
        let mut buf = [0u8; 8];
        buf.copy_from_slice(&extra_bytes[40..48]);
        u64::from_be_bytes(buf)
    } else {
        0
    };
    (delayed, l2_block)
}

/// EIP-2935: Store the parent block hash in the history storage contract.
///
/// For Arbitrum, uses L2 block numbers and a buffer size of 393168 blocks.
fn process_parent_block_hash<DB: Database>(
    state: &mut State<DB>,
    l2_block_number: u64,
    prev_hash: B256,
) {
    use arb_primitives::arbos_versions::HISTORY_STORAGE_ADDRESS;

    /// Arbitrum EIP-2935 buffer size (matching the Arbitrum history storage contract).
    const HISTORY_SERVE_WINDOW: u64 = 393168;

    if l2_block_number == 0 {
        return;
    }

    let slot = U256::from((l2_block_number - 1) % HISTORY_SERVE_WINDOW);
    let value = U256::from_be_slice(prev_hash.as_slice());

    arb_storage::write_storage_at(state, HISTORY_STORAGE_ADDRESS, slot, value);
}

/// Extract the gas field from a scheduled retry tx's encoded bytes.
///
/// The encoded format is `[type_byte][RLP(ArbRetryTx)]`.
fn decode_retry_tx_gas(encoded: &[u8]) -> Option<u64> {
    if encoded.is_empty() {
        return None;
    }
    if encoded[0] != ArbTxType::ArbitrumRetryTx.as_u8() {
        tracing::warn!(
            target: "arb::executor",
            tx_type = encoded[0],
            "unexpected scheduled tx type"
        );
        return None;
    }
    let rlp_data = &encoded[1..];
    let retry = <arb_alloy_consensus::tx::ArbRetryTx as alloy_rlp::Decodable>::decode(
        &mut &rlp_data[..],
    )
    .ok()?;
    Some(retry.gas)
}

