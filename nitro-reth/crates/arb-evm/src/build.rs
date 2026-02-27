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
use reth_evm::TransactionEnv;
use arbos::arbos_state::ArbosState;
use arbos::burn::SystemBurner;
use arbos::internal_tx::{self, InternalTxContext};
use arb_primitives::multigas::MultiGas;
use arb_primitives::tx_types::ArbTxType;
use arb_chainspec;
use arbos::l1_pricing;
use arbos::tx_processor::{EndTxFeeDistribution, compute_poster_gas};
use arbos::util::tx_type_has_poster_costs;
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
    EvmF: EvmFactory<
        Tx: FromRecoveredTx<R::Transaction>
            + FromTxWithEncoded<R::Transaction>
            + TransactionEnv,
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

    /// Read state parameters from ArbOS state into the execution context
    /// and create/update the hooks.
    fn load_state_params<D: Database>(
        &mut self,
        arb_state: &ArbosState<D, impl arbos::burn::Burner>,
    ) {
        let arbos_version = arb_state.arbos_version();
        self.arb_ctx.arbos_version = arbos_version;

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

        self.arb_hooks = Some(DefaultArbOsHooks::new(
            self.arb_ctx.coinbase,
            arbos_version,
            self.arb_ctx.network_fee_account,
            self.arb_ctx.infra_fee_account,
            self.arb_ctx.min_base_fee,
            per_block_gas_limit,
            per_tx_gas_limit,
            false,
            self.arb_ctx.l1_base_fee,
        ));
    }
}

impl<'db, DB, E, Spec, R> BlockExecutor for ArbBlockExecutor<'_, E, Spec, R>
where
    DB: Database + 'db,
    E: Evm<
        DB = &'db mut State<DB>,
        Tx: FromRecoveredTx<R::Transaction>
            + FromTxWithEncoded<R::Transaction>
            + TransactionEnv,
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
        {
            let block = self.inner.evm().block();
            let timestamp = revm::context::Block::timestamp(block).to::<u64>();
            if self.arb_ctx.block_timestamp == 0 {
                self.arb_ctx.block_timestamp = timestamp;
            }
            self.arb_ctx.coinbase = revm::context::Block::beneficiary(block);
            if let Some(prevrandao) = revm::context::Block::prevrandao(block) {
                if self.arb_ctx.l1_block_number == 0 {
                    self.arb_ctx.l1_block_number =
                        crate::config::l1_block_number_from_mix_hash(&prevrandao);
                }
            }
        }

        // Load ArbOS state parameters from the EVM database.
        // Block-start operations (pricing model update, retryable reaping, etc.)
        // are triggered by the startBlock internal tx, NOT here.
        // We only commit multi-gas fees and read state parameters.
        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
        let state_ptr: *mut State<DB> = db as *mut State<DB>;

        if let Ok(arb_state) =
            ArbosState::open(state_ptr, SystemBurner::new(None, false))
        {
            // Rotate multi-gas fees: copy next-block fees to current-block.
            let _ = arb_state.l2_pricing_state.commit_multi_gas_fees();

            // Read state parameters for the execution context and hooks.
            self.load_state_params(&arb_state);
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
        // Decompose the transaction to extract sender, type, and gas limit.
        let (tx_env, recovered) = tx.into_parts();
        let sender = *recovered.signer();
        let tx_type_raw = recovered.tx().ty();
        let tx_gas_limit = recovered.tx().gas_limit();

        // Classify the transaction type.
        let arb_tx_type = ArbTxType::from_u8(tx_type_raw).ok();
        let is_arb_internal = arb_tx_type == Some(ArbTxType::ArbitrumInternalTx);
        let is_arb_deposit = arb_tx_type == Some(ArbTxType::ArbitrumDepositTx);
        let has_poster_costs = tx_type_has_poster_costs(tx_type_raw);

        // Reset per-tx processor state. Go creates a fresh TxProcessor per tx;
        // we reset the mutable fields that accumulate across transactions.
        if let Some(hooks) = self.arb_hooks.as_mut() {
            hooks.tx_proc.poster_fee = U256::ZERO;
            hooks.tx_proc.poster_gas = 0;
            hooks.tx_proc.compute_hold_gas = 0;
            hooks.tx_proc.current_retryable = None;
            hooks.tx_proc.current_refund_to = None;
            hooks.tx_proc.scheduled_txs.clear();
        }

        // --- Pre-execution: apply special tx type state changes ---

        // Internal txs: startBlock triggers block-start operations (pricing model
        // update, retryable reaping, upgrade checks); batch posting reports
        // update L1 pricing from batch poster spending.
        if is_arb_internal {
            let tx_data = recovered.tx().input().to_vec();
            if tx_data.len() >= 4 {
                let selector: [u8; 4] = tx_data[0..4].try_into().unwrap();
                let is_start_block = selector == internal_tx::INTERNAL_TX_START_BLOCK_METHOD_ID;

                // Extract l1_base_fee and time_passed from startBlock data
                // before dispatching, so we can update context afterwards.
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
                    let mut noop = |_: Address, _: Address, _: U256| Ok(());
                    if let Err(e) = internal_tx::apply_internal_tx_update(
                        &tx_data,
                        &mut arb_state,
                        &ctx,
                        &mut noop,
                    ) {
                        tracing::warn!(
                            target: "arb::executor",
                            error = %e,
                            "internal tx processing failed"
                        );
                    }

                    // After startBlock, re-read updated state params since the
                    // pricing model update may have changed base fees.
                    if is_start_block {
                        self.load_state_params(&arb_state);
                    }
                }
            }
        }

        // Deposit txs mint ETH to the sender before EVM execution.
        // The EVM then transfers value from sender to recipient naturally.
        if is_arb_deposit {
            let value = recovered.tx().value();
            if !value.is_zero() {
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                mint_balance(db, sender, value);
            }
        }

        // --- Poster cost and gas limiting (user txs only) ---

        let mut poster_gas = 0u64;
        let mut compute_hold_gas = 0u64;
        let calldata_units = if has_poster_costs {
            let tx_bytes = recovered.tx().encoded_2718();
            let (poster_cost, units) = l1_pricing::compute_poster_cost_standalone(
                &tx_bytes,
                self.arb_ctx.coinbase,
                self.arb_ctx.l1_price_per_unit,
                self.arb_ctx.brotli_compression_level,
            );

            if let Some(hooks) = self.arb_hooks.as_mut() {
                let base_fee = self.arb_ctx.basefee;
                hooks.tx_proc.poster_gas =
                    compute_poster_gas(poster_cost, base_fee, false, self.arb_ctx.min_base_fee);
                hooks.tx_proc.poster_fee =
                    base_fee.saturating_mul(U256::from(hooks.tx_proc.poster_gas));
                poster_gas = hooks.tx_proc.poster_gas;

                // Compute gas held to cap computation per tx.
                // Estimate intrinsic gas for the hold calculation.
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

            units
        } else {
            0
        };

        // --- Adjust gas limit for EVM execution ---
        //
        // Reduce the gas the EVM sees by poster_gas (L1 data cost expressed
        // as L2 gas) and compute_hold_gas (excess gas held to enforce the
        // per-block/per-tx gas limit). Both are accounted for in fee
        // distribution after execution.
        let mut tx_env = tx_env;
        let gas_deduction = poster_gas.saturating_add(compute_hold_gas);
        if gas_deduction > 0 {
            let current = revm::context_interface::Transaction::gas_limit(&tx_env);
            tx_env.set_gas_limit(current.saturating_sub(gas_deduction));
        }

        // --- Execute via inner EVM executor ---

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

        // --- Post-execution: fee distribution (user txs only) ---
        //
        // The EVM's gas_used doesn't include poster_gas (we subtracted it
        // from the gas limit). Add poster_gas back for correct total gas_used.
        // compute_hold_gas is NOT consumed — it's returned to the user.

        if has_poster_costs {
            let fee_dist = if result.is_some() {
                let base_fee = self.arb_ctx.basefee;
                let evm_gas_used = captured_gas_used.get();
                let gas_used = evm_gas_used.saturating_add(poster_gas);
                let gas_left = tx_gas_limit.saturating_sub(gas_used);

                self.arb_hooks.as_ref().map(|hooks| {
                    hooks.compute_end_tx_fees(&EndTxContext {
                        sender,
                        gas_left,
                        gas_used,
                        gas_price: base_fee,
                        base_fee,
                        tx_type: arb_tx_type
                            .unwrap_or(ArbTxType::ArbitrumLegacyTx),
                        success: captured_success.get(),
                        refund_to: sender,
                    })
                })
            } else {
                None
            };

            if let Some(ref dist) = fee_dist {
                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                apply_fee_distribution(db, dist, None);

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

/// Estimate intrinsic gas for a transaction.
///
/// Used to compute compute_hold_gas before passing the tx to the EVM.
/// Matches the standard Ethereum intrinsic gas calculation.
fn estimate_intrinsic_gas(tx: &impl Transaction) -> u64 {
    const TX_GAS: u64 = 21_000;
    const TX_CREATE_GAS: u64 = 32_000;
    const TX_DATA_ZERO_GAS: u64 = 4;
    const TX_DATA_NON_ZERO_GAS: u64 = 16;

    let mut gas = TX_GAS;
    if tx.to().is_none() {
        gas += TX_CREATE_GAS;
    }
    let data = tx.input();
    let data_gas: u64 = data
        .iter()
        .map(|&b| if b == 0 { TX_DATA_ZERO_GAS } else { TX_DATA_NON_ZERO_GAS })
        .sum();
    gas.saturating_add(data_gas)
}
