use core::cell::Cell;

use alloy_consensus::{Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::eip2718::Encodable2718;
use alloy_eips::eip2718::Typed2718;
use alloy_evm::block::{
    BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
    BlockExecutorFor, CommitChanges, ExecutableTx, OnStateHook,
};
use alloy_evm::RecoveredTx;
use alloy_evm::eth::EthTxResult;
use alloy_evm::eth::receipt_builder::ReceiptBuilder;
use alloy_evm::eth::spec::EthExecutorSpec;
use alloy_evm::eth::{EthBlockExecutionCtx, EthBlockExecutor};
use alloy_evm::tx::{FromRecoveredTx, FromTxWithEncoded};
use alloy_evm::{Database, Evm, EvmFactory};
use alloy_primitives::{Address, Log, U256};
use arbos::arbos_state::ArbosState;
use arbos::burn::SystemBurner;
use arb_primitives::multigas::MultiGas;
use arbos::l1_pricing;
use arbos::tx_processor::{EndTxFeeDistribution, compute_poster_gas};
use revm::context::result::ExecutionResult;
use revm::database::State;
use revm::inspector::Inspector;
use revm_database::DatabaseCommitExt;

use crate::context::ArbBlockExecutionCtx;
use crate::executor::DefaultArbOsHooks;
use crate::hooks::EndTxContext;

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
}

impl<R, Spec, EvmF> BlockExecutorFactory for ArbBlockExecutorFactory<R, Spec, EvmF>
where
    R: ReceiptBuilder<
            Transaction: Transaction + Encodable2718,
            Receipt: TxReceipt<Log = Log>,
        > + 'static,
    Spec: EthExecutorSpec + Clone + 'static,
    EvmF: EvmFactory<Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction>>,
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
        // Capture header-derived fields from the Eth context before passing
        // it to the inner executor. The rest is populated from state in
        // apply_pre_execution_changes.
        let arb_ctx = ArbBlockExecutionCtx {
            parent_hash: ctx.parent_hash,
            parent_beacon_block_root: ctx.parent_beacon_block_root,
            extra_data: ctx.extra_data.to_vec(),
            ..Default::default()
        };
        ArbBlockExecutor {
            inner: EthBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder),
            arb_hooks: None,
            arb_ctx,
        }
    }
}

/// Arbitrum block executor wrapping `EthBlockExecutor`.
///
/// Adds ArbOS-specific pre/post execution logic:
/// - Loads ArbOS state at block start (version, fee accounts)
/// - Calls ArbOS hooks for gas charging on each transaction
/// - Skips block rewards (Arbitrum has no mining rewards)
pub struct ArbBlockExecutor<'a, Evm, Spec, R: ReceiptBuilder> {
    /// Inner Ethereum block executor.
    pub inner: EthBlockExecutor<'a, Evm, Spec, R>,
    /// ArbOS hooks for per-transaction processing.
    pub arb_hooks: Option<DefaultArbOsHooks>,
    /// Arbitrum-specific block context.
    pub arb_ctx: ArbBlockExecutionCtx,
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
}

impl<'db, DB, E, Spec, R> BlockExecutor for ArbBlockExecutor<'_, E, Spec, R>
where
    DB: Database + 'db,
    E: Evm<
        DB = &'db mut State<DB>,
        Tx: FromRecoveredTx<R::Transaction> + FromTxWithEncoded<R::Transaction>,
    >,
    Spec: EthExecutorSpec,
    R: ReceiptBuilder<
        Transaction: Transaction + Encodable2718,
        Receipt: TxReceipt<Log = Log>,
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
        // These are fields encoded in mix_hash and other header fields that
        // aren't available through the standard EthBlockExecutionCtx.
        {
            let block = self.inner.evm().block();
            let timestamp = revm::context::Block::timestamp(block).to::<u64>();
            if self.arb_ctx.block_timestamp == 0 {
                self.arb_ctx.block_timestamp = timestamp;
            }
            if let Some(prevrandao) = revm::context::Block::prevrandao(block) {
                if self.arb_ctx.l1_block_number == 0 {
                    self.arb_ctx.l1_block_number =
                        crate::config::l1_block_number_from_mix_hash(&prevrandao);
                }
            }
        }

        // Load ArbOS state parameters from the EVM database.
        // This populates the execution context with state-derived fields
        // and creates the ArbOS hooks for per-transaction processing.
        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
        let state_ptr: *mut State<DB> = db as *mut State<DB>;

        if let Ok(mut arb_state) =
            ArbosState::open(state_ptr, SystemBurner::new(None, false))
        {
            // Rotate multi-gas fees: copy next-block fees to current-block.
            // This must happen before reading the base fee or executing transactions.
            let _ = arb_state.l2_pricing_state.commit_multi_gas_fees();

            let arbos_version = arb_state.arbos_version;
            let time_passed = self.arb_ctx.time_passed;
            let block_timestamp = self.arb_ctx.block_timestamp;

            // --- Start-block state updates ---
            // Record L1 block hashes if L1 block number advanced.
            let l1_block_number = self.arb_ctx.l1_block_number;
            if let Ok(old_l1_block) = arb_state.blockhashes.l1_block_number() {
                if l1_block_number > old_l1_block {
                    let _ = arb_state.blockhashes.record_new_l1_block(
                        l1_block_number - 1,
                        self.arb_ctx.parent_hash,
                        arbos_version,
                    );
                }
            }

            // Reap up to 2 expired retryables.
            let noop_transfer = &mut |_from: Address, _to: Address, _value: U256| -> Result<(), ()> {
                // TODO: implement ETH transfer via State<DB> for retryable reaping
                Ok(())
            };
            let _ = arb_state.retryable_state.try_to_reap_one_retryable(
                block_timestamp,
                &mut *noop_transfer,
            );
            let _ = arb_state.retryable_state.try_to_reap_one_retryable(
                block_timestamp,
                &mut *noop_transfer,
            );

            // Update L2 pricing model (drain backlogs, recalculate base fee).
            let _ = arb_state
                .l2_pricing_state
                .update_pricing_model(time_passed, arbos_version);

            // Check for scheduled ArbOS upgrade.
            let _ = arb_state.upgrade_arbos_version_if_necessary(block_timestamp);

            // Re-read state after updates (version may have changed).
            let arbos_version = arb_state.arbos_version();

            let network_fee_account = arb_state
                .network_fee_account
                .get()
                .unwrap_or(Address::ZERO);
            let infra_fee_account = arb_state
                .infra_fee_account
                .get()
                .unwrap_or(Address::ZERO);
            let brotli_compression_level = arb_state
                .brotli_compression_level
                .get()
                .unwrap_or(0);

            let l1_price_per_unit = arb_state
                .l1_pricing_state
                .price_per_unit()
                .unwrap_or(U256::ZERO);
            let min_base_fee = arb_state
                .l2_pricing_state
                .min_base_fee_wei()
                .unwrap_or(U256::ZERO);
            let per_block_gas_limit = arb_state
                .l2_pricing_state
                .per_block_gas_limit()
                .unwrap_or(0);
            let per_tx_gas_limit = arb_state
                .l2_pricing_state
                .per_tx_gas_limit()
                .unwrap_or(0);
            // Base fee from L2 pricing state (after pricing model update).
            let base_fee = arb_state
                .l2_pricing_state
                .base_fee_wei()
                .unwrap_or(U256::ZERO);
            let l1_base_fee = self.arb_ctx.l1_base_fee;

            // Populate state-derived fields in the execution context.
            self.arb_ctx.arbos_version = arbos_version;
            self.arb_ctx.network_fee_account = network_fee_account;
            self.arb_ctx.infra_fee_account = infra_fee_account;
            self.arb_ctx.brotli_compression_level = brotli_compression_level;
            self.arb_ctx.l1_price_per_unit = l1_price_per_unit;
            self.arb_ctx.min_base_fee = min_base_fee;
            self.arb_ctx.basefee = base_fee;

            // Create ArbOS hooks with the loaded state parameters.
            let coinbase = revm::context::Block::beneficiary(self.inner.evm().block());
            self.arb_hooks = Some(DefaultArbOsHooks::new(
                coinbase,
                arbos_version,
                network_fee_account,
                infra_fee_account,
                min_base_fee,
                per_block_gas_limit,
                per_tx_gas_limit,
                false,
                l1_base_fee,
            ));
        }

        tracing::trace!(
            target: "arb::executor",
            l1_block = self.arb_ctx.l1_block_number,
            delayed_msgs = self.arb_ctx.delayed_messages_read,
            chain_id = self.arb_ctx.chain_id,
            basefee = %self.arb_ctx.basefee,
            arbos_version = self.arb_ctx.arbos_version,
            has_hooks = self.arb_hooks.is_some(),
            "starting arbitrum block execution"
        );

        Ok(())
    }

    fn execute_transaction_without_commit(
        &mut self,
        tx: impl ExecutableTx<Self>,
    ) -> Result<Self::Result, BlockExecutionError> {
        self.inner.execute_transaction_without_commit(tx)
    }

    fn commit_transaction(&mut self, output: Self::Result) -> Result<u64, BlockExecutionError> {
        self.inner.commit_transaction(output)
    }

    fn execute_transaction_with_commit_condition(
        &mut self,
        tx: impl ExecutableTx<Self>,
        f: impl FnOnce(&ExecutionResult<<Self::Evm as Evm>::HaltReason>) -> CommitChanges,
    ) -> Result<Option<u64>, BlockExecutionError> {
        // Decompose the transaction to extract sender and type before execution.
        let (tx_env, recovered) = tx.into_parts();
        let sender = *recovered.signer();
        let tx_type_raw = recovered.tx().ty();

        // Pre-compute poster costs for correct fee distribution.
        // Encode the tx to compute brotli-compressed L1 data cost.
        let tx_bytes = recovered.tx().encoded_2718();
        let coinbase = self
            .arb_hooks
            .as_ref()
            .map(|h| h.coinbase)
            .unwrap_or(Address::ZERO);
        let (poster_cost, calldata_units) = l1_pricing::compute_poster_cost_standalone(
            &tx_bytes,
            coinbase,
            self.arb_ctx.l1_price_per_unit,
            self.arb_ctx.brotli_compression_level,
        );

        // Set poster_gas and poster_fee on the TxProcessor so fee distribution
        // correctly splits between poster and compute costs.
        if let Some(hooks) = self.arb_hooks.as_mut() {
            let base_fee = self.arb_ctx.basefee;
            hooks.tx_proc.poster_gas =
                compute_poster_gas(poster_cost, base_fee, false, self.arb_ctx.min_base_fee);
            hooks.tx_proc.poster_fee =
                base_fee.saturating_mul(U256::from(hooks.tx_proc.poster_gas));
        }

        // Capture execution result info for post-tx hooks.
        let captured_gas_used = Cell::new(0u64);
        let captured_success = Cell::new(false);

        let result = self.inner.execute_transaction_with_commit_condition(
            (tx_env, recovered),
            |exec_result| {
                let (used, success) = match exec_result {
                    ExecutionResult::Success { gas_used, .. } => (*gas_used, true),
                    ExecutionResult::Revert { gas_used, .. } => (*gas_used, false),
                    ExecutionResult::Halt { gas_used, .. } => (*gas_used, false),
                };
                captured_gas_used.set(used);
                captured_success.set(success);
                f(exec_result)
            },
        )?;

        // Post-execution: compute fee distribution.
        let fee_dist = if let Some(committed_gas) = result {
            let base_fee = self.arb_ctx.basefee;
            let gas_used = captured_gas_used.get();
            let arb_tx_type = arb_primitives::tx_types::ArbTxType::from_u8(tx_type_raw)
                .unwrap_or(arb_primitives::tx_types::ArbTxType::ArbitrumLegacyTx);

            self.arb_hooks.as_ref().map(|hooks| {
                hooks.compute_end_tx_fees(&EndTxContext {
                    sender,
                    gas_left: committed_gas.saturating_sub(gas_used),
                    gas_used,
                    gas_price: base_fee,
                    base_fee,
                    tx_type: arb_tx_type,
                    success: captured_success.get(),
                    refund_to: sender,
                })
            })
        } else {
            None
        };

        // Apply fee distribution and update pricing state.
        if let Some(ref dist) = fee_dist {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            apply_fee_distribution(db, dist, None);

            // Update gas backlog and L1 pricing state.
            let state_ptr: *mut State<DB> = db as *mut State<DB>;
            if let Ok(arb_state) =
                ArbosState::open(state_ptr, SystemBurner::new(None, false))
            {
                let _ = arb_state.l2_pricing_state.grow_backlog(
                    dist.compute_gas_for_backlog,
                    MultiGas::default(),
                );
                if !dist.l1_fees_to_add.is_zero() {
                    let _ = arb_state
                        .l1_pricing_state
                        .add_to_l1_fees_available(dist.l1_fees_to_add);
                }
                let _ = arb_state
                    .l1_pricing_state
                    .add_to_units_since_update(calldata_units);
            }
        }

        Ok(result)
    }

    fn finish(
        self,
    ) -> Result<(Self::Evm, BlockExecutionResult<R::Receipt>), BlockExecutionError> {
        self.inner.finish()
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
// Fee distribution helpers
// ---------------------------------------------------------------------------

/// Mint balance to an address in the EVM state.
///
/// This is Arbitrum's mechanism for crediting fee accounts without
/// a corresponding debit (ETH is minted into the L2).
fn mint_balance<DB: Database>(state: &mut State<DB>, address: Address, amount: U256) {
    if amount.is_zero() || address == Address::ZERO {
        return;
    }
    let _ = state.load_cache_account(address);
    let amount_u128: u128 = amount.try_into().unwrap_or(u128::MAX);
    let _ = state.increment_balances(core::iter::once((address, amount_u128)));
}

/// Apply a computed fee distribution to the EVM state.
///
/// Mints ETH to the network fee account, infrastructure fee account,
/// and poster fee destination (L1 pricer funds pool). Also updates
/// L1 fees available in L1 pricing state when applicable.
fn apply_fee_distribution<DB: Database>(
    state: &mut State<DB>,
    dist: &EndTxFeeDistribution,
    l1_pricing: Option<&l1_pricing::L1PricingState<DB>>,
) {
    mint_balance(state, dist.network_fee_account, dist.network_fee_amount);
    mint_balance(state, dist.infra_fee_account, dist.infra_fee_amount);
    mint_balance(state, dist.poster_fee_destination, dist.poster_fee_amount);

    // Update L1 fees available for the pricing model.
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
