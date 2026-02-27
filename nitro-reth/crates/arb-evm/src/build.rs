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
use revm_database::{DatabaseCommit, DatabaseCommitExt};

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
            Transaction: Transaction + Encodable2718 + ArbTransactionExt,
            Receipt: TxReceipt<Log = Log>,
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
        // Decode delayed_messages_read from bytes 32-39 of extra_data if present.
        let extra_bytes = ctx.extra_data.as_ref();
        let delayed_messages_read = if extra_bytes.len() >= 40 {
            let mut buf = [0u8; 8];
            buf.copy_from_slice(&extra_bytes[32..40]);
            u64::from_be_bytes(buf)
        } else {
            0
        };
        let arb_ctx = ArbBlockExecutionCtx {
            parent_hash: ctx.parent_hash,
            parent_beacon_block_root: ctx.parent_beacon_block_root,
            extra_data: extra_bytes[..core::cmp::min(extra_bytes.len(), 32)].to_vec(),
            delayed_messages_read,
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
    calldata_units: u64,
    /// Gas that reth's EVM actually charged the sender for (gas_used before
    /// poster/compute adjustments). Zero for paths that bypass reth's EVM
    /// (internal tx, deposit, submit retryable, pre-recorded revert, filtered tx).
    /// Used to compute how much additional balance to debit from the sender.
    evm_gas_used: u64,
    /// Multi-dimensional gas charged during gas charging (L1 calldata component).
    charged_multi_gas: MultiGas,
    /// True when the tx's effective gas price is non-zero. Go skips backlog
    /// updates when gas price is zero (test/estimation scenarios).
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

    /// Deduct TX_GAS from block gas budget for a failed/invalid transaction.
    /// Call this when a transaction fails execution so the block budget
    /// stays in sync (matching Go's behavior of charging TX_GAS for invalid txs).
    pub fn deduct_failed_tx_gas(&mut self) {
        const TX_GAS: u64 = 21_000;
        self.block_gas_left = self.block_gas_left.saturating_sub(TX_GAS);
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

        let mut hooks = DefaultArbOsHooks::new(
            self.arb_ctx.coinbase,
            arbos_version,
            self.arb_ctx.network_fee_account,
            self.arb_ctx.infra_fee_account,
            self.arb_ctx.min_base_fee,
            per_block_gas_limit,
            per_tx_gas_limit,
            false,
            self.arb_ctx.l1_base_fee,
        );
        // Populate L1 block number cache from header-derived context.
        if self.arb_ctx.l1_block_number > 0 {
            hooks.tx_proc.set_l1_block_number(self.arb_ctx.l1_block_number);
        }
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
    /// Returns a synthetic execution result (endTxNow=true in Go terms).
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
                calldata_units: 0,
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

        // 3. Transfer submission fee to network fee account.
        if !fees.submission_fee.is_zero() {
            transfer_balance(db, sender, self.arb_ctx.network_fee_account, fees.submission_fee);
        }

        // 4. Refund excess submission fee.
        if !fees.submission_fee_refund.is_zero() {
            transfer_balance(db, sender, info.fee_refund_addr, fees.submission_fee_refund);
        }

        // 5. Move call value into escrow. If sender has insufficient funds
        //    (e.g. deposit didn't cover retry_value after fee deductions),
        //    refund the submission fee and end the transaction.
        if !info.retry_value.is_zero() {
            if !try_transfer_balance(db, sender, fees.escrow, info.retry_value) {
                // Refund submission fee from network account back to sender.
                transfer_balance(
                    db, self.arb_ctx.network_fee_account, sender, fees.submission_fee,
                );
                // Refund withheld portion of submission fee to fee refund address.
                transfer_balance(
                    db, sender, info.fee_refund_addr, fees.withheld_submission_fee,
                );

                self.pending_tx = Some(PendingArbTx {
                    sender,
                    tx_gas_limit: user_gas,
                    arb_tx_type: Some(ArbTxType::ArbitrumSubmitRetryableTx),
                    has_poster_costs: false,
                    poster_gas: 0,
                    evm_gas_used: 0,
                    calldata_units: 0,
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
        }

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
            // Pay infra fee.
            if !fees.infra_cost.is_zero() {
                transfer_balance(db, sender, self.arb_ctx.infra_fee_account, fees.infra_cost);
            }
            // Pay network fee.
            if !fees.network_cost.is_zero() {
                transfer_balance(db, sender, self.arb_ctx.network_fee_account, fees.network_cost);
            }
            // Gas price refund.
            if !fees.gas_price_refund.is_zero() {
                transfer_balance(db, sender, info.fee_refund_addr, fees.gas_price_refund);
            }

            // For filtered retryables, skip auto-redeem scheduling.
            if !is_filtered {
                // Schedule auto-redeem: use make_tx to construct from stored
                // retryable fields (matching Go's MakeTx pattern), then
                // increment num_tries.
                let state_ptr2: *mut State<DB> = db as *mut State<DB>;
                if let Ok(arb_state) = ArbosState::open(state_ptr2, SystemBurner::new(None, false)) {
                    if let Ok(Some(retryable)) = arb_state.retryable_state.open_retryable(
                        ticket_id,
                        u64::MAX, // pass max time so it's always considered valid
                    ) {
                        let _ = retryable.increment_num_tries();

                        if let Ok(retry_tx) = retryable.make_tx(
                            U256::from(self.arb_ctx.chain_id),
                            0, // nonce = 0 for first auto-redeem
                            effective_base_fee,
                            user_gas,
                            ticket_id,
                            info.fee_refund_addr,
                            fees.available_refund,
                            fees.submission_fee,
                        ) {
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
                                hooks.tx_proc.scheduled_txs.push(encoded);
                            }
                        }
                    }
                }
            }
        } else if !fees.gas_cost_refund.is_zero() {
            // Can't pay for gas: refund gas cost from deposit.
            transfer_balance(db, sender, info.fee_refund_addr, fees.gas_cost_refund);
        }

        // Store pending state for commit_transaction.
        self.pending_tx = Some(PendingArbTx {
            sender,
            tx_gas_limit: user_gas,
            arb_tx_type: Some(ArbTxType::ArbitrumSubmitRetryableTx),
            has_poster_costs: false, // No poster costs for submit retryable
            poster_gas: 0,
            evm_gas_used: 0,
            calldata_units: 0,
            charged_multi_gas: if fees.can_pay_for_gas {
                MultiGas::l2_calldata_gas(user_gas)
            } else {
                MultiGas::default()
            },
            gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
            retry_context: None,
        });

        // Construct synthetic execution result. Filtered retryables always
        // return a failure receipt (Go sets filteredErr). Non-filtered txs
        // succeed even when can't pay for gas (retryable was created).
        let gas_used = if fees.can_pay_for_gas { user_gas } else { 0 };
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
            self.arb_ctx.basefee = U256::from(revm::context::Block::basefee(block));
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
        let db: &mut State<DB> = self.inner.evm_mut().db_mut();
        let state_ptr: *mut State<DB> = db as *mut State<DB>;

        if let Ok(arb_state) =
            ArbosState::open(state_ptr, SystemBurner::new(None, false))
        {
            // Rotate multi-gas fees: copy next-block fees to current-block.
            let _ = arb_state.l2_pricing_state.commit_multi_gas_fees();

            // Read state parameters for the execution context and hooks.
            self.load_state_params(&arb_state);

            // Initialize block gas rate limiting.
            self.block_gas_left = arb_state
                .l2_pricing_state
                .per_block_gas_limit()
                .unwrap_or(0);
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
                    let mut do_transfer = |from: Address, to: Address, amount: U256| {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        unsafe { transfer_balance(&mut *state_ptr, from, to, amount) };
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
                calldata_units: 0,
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

            self.pending_tx = Some(PendingArbTx {
                sender,
                tx_gas_limit: 0,
                arb_tx_type: Some(ArbTxType::ArbitrumDepositTx),
                has_poster_costs: false,
                poster_gas: 0,
                evm_gas_used: 0,
                calldata_units: 0,
                charged_multi_gas: MultiGas::default(),
                gas_price_positive: self.arb_ctx.basefee > U256::ZERO,
                retry_context: None,
            });

            // Filtered deposits: Go returns ErrFilteredTx (non-nil error) which
            // produces a failed receipt (status=0). The state changes (mint +
            // redirected transfer) are still committed.
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
                            if !value.is_zero()
                                && !try_transfer_balance(db, escrow, sender, value)
                            {
                                // Escrow has insufficient funds — abort the retry tx.
                                let tx_type = recovered.tx().tx_type();
                                self.pending_tx = Some(PendingArbTx {
                                    sender,
                                    tx_gas_limit: 0,
                                    arb_tx_type: Some(ArbTxType::ArbitrumRetryTx),
                                    has_poster_costs: false,
                                    poster_gas: 0,
                                    evm_gas_used: 0,
                                    calldata_units: 0,
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
                                calldata_units: 0,
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
                                calldata_units: 0,
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

        // --- Poster cost and gas limiting (user txs only) ---

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

        // ArbOS < 50: reject user txs whose compute gas exceeds block gas left,
        // but always allow the first user tx through (matching Go's userTxsProcessed > 0).
        // ArbOS >= 50 uses per-tx gas limit clamping (compute_hold_gas) instead.
        if is_user_tx
            && self.arb_ctx.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_50
            && self.user_txs_processed > 0
        {
            let compute_gas = tx_gas_limit.saturating_sub(poster_gas);
            if compute_gas > self.block_gas_left {
                return Err(BlockExecutionError::msg("block gas limit reached"));
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
                            calldata_units,
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
                            calldata_units,
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

        // Drop the priority fee tip: cap gas price to the base fee.
        // In Arbitrum, fees go to network/infra accounts via EndTxHook, not to coinbase.
        // Without this, revm's reward_beneficiary sends the tip to coinbase.
        let should_drop_tip = self.arb_hooks.as_ref()
            .map(|h| h.drop_tip())
            .unwrap_or(false);
        if should_drop_tip {
            let base_fee: u128 = self.arb_ctx.basefee.try_into().unwrap_or(u128::MAX);
            let current_price = revm::context_interface::Transaction::gas_price(&tx_env);
            if current_price > base_fee {
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

        // Fix nonce for retry txs: the encoded tx nonce is the retryable's
        // numTries sequence number, not the sender's account nonce. Go's
        // skipNonceChecks() returns true for ArbitrumRetryTx. Override the
        // tx_env nonce to match the sender's current state nonce so revm's
        // nonce validation passes. revm will then increment it (matching Go).
        if is_retry_tx {
            let db: &mut State<DB> = self.inner.evm_mut().db_mut();
            let sender_nonce = db.load_cache_account(sender)
                .map(|a| a.account_info().map(|i| i.nonce).unwrap_or(0))
                .unwrap_or(0);
            tx_env.set_nonce(sender_nonce);
        }

        let mut output = self.inner.execute_transaction_without_commit((tx_env, recovered))?;

        // Capture gas_used as reported by reth's EVM (before our adjustments).
        // This represents the gas cost reth already deducted from the sender.
        let evm_gas_used = output.result.result.gas_used();

        // Adjust gas_used to include poster_gas only.
        // poster_gas was deducted from gas_limit before EVM execution so reth's
        // reported gas_used doesn't include it. Adding it back produces correct
        // receipt gas_used. compute_hold_gas is NOT added: Go returns it via
        // calcHeldGasRefund() before computing final gasUsed, and Go's
        // NonRefundableGas() excludes it from the refund denominator.
        if poster_gas > 0 {
            adjust_result_gas_used(&mut output.result.result, poster_gas);
        }

        // Scan execution logs for RedeemScheduled events (manual redeem path).
        // The ArbRetryableTx.Redeem precompile emits this event; we discover it
        // here and schedule the retry tx, matching Go's ScheduledTxes() mechanism.
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
        // Build multi-gas: L1 calldata gas was charged for poster costs.
        let charged_multi_gas = MultiGas::l1_calldata_gas(poster_gas);

        self.pending_tx = Some(PendingArbTx {
            sender,
            tx_gas_limit,
            arb_tx_type,
            has_poster_costs,
            poster_gas,
            evm_gas_used,
            calldata_units,
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

        // Inner executor builds receipt with the adjusted gas_used and commits state.
        let gas_used = self.inner.commit_transaction(output)?;

        // Track poster gas for this receipt (parallel to receipts vector).
        let poster_gas_for_receipt = pending.as_ref().map_or(0, |p| p.poster_gas);
        self.gas_used_for_l1.push(poster_gas_for_receipt);

        // --- Post-execution: fee distribution ---
        if let Some(pending) = pending {
            let is_retry = pending.retry_context.is_some();

            // Charge the sender for gas costs that reth's internal buyGas
            // didn't cover. For normal EVM txs, this equals poster_gas
            // (deducted from gas_limit before reth sees it; compute_hold_gas
            // is also deducted but not charged — Go returns it via
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
            }

            if let Some(retry_ctx) = pending.retry_context {
                // RetryTx end-of-tx: handle gas refunds, retryable cleanup.
                let gas_left = pending.tx_gas_limit.saturating_sub(gas_used_total);

                let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                let state_ptr: *mut State<DB> = db as *mut State<DB>;

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
                            // SAFETY: closures execute sequentially within end_tx_retryable.
                            unsafe { burn_balance(&mut *state_ptr, addr, amount) };
                        },
                        |from, to, amount| {
                            // SAFETY: closures execute sequentially within end_tx_retryable.
                            unsafe { transfer_balance(&mut *state_ptr, from, to, amount) };
                            Ok(())
                        },
                    )
                });

                if let Some(ref result) = result {
                    if result.should_delete_retryable {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        if let Ok(arb_state) =
                            ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        {
                            let _ = arb_state.retryable_state.delete_retryable(
                                retry_ctx.ticket_id,
                                |from, to, amount| {
                                    // SAFETY: called sequentially within delete_retryable.
                                    unsafe { transfer_balance(&mut *state_ptr, from, to, amount) };
                                    Ok(())
                                },
                                |addr| {
                                    // SAFETY: called sequentially within delete_retryable.
                                    unsafe { get_balance(&mut *state_ptr, addr) }
                                },
                            );
                        }
                    } else if result.should_return_value_to_escrow
                        && !retry_ctx.call_value.is_zero()
                    {
                        // Failed retry: return call value to escrow.
                        unsafe {
                            transfer_balance(
                                &mut *state_ptr,
                                pending.sender,
                                result.escrow_address,
                                retry_ctx.call_value,
                            );
                        }
                    }

                    // Grow gas backlog with the actual multi-gas used.
                    // Go skips this when gas price is zero (test scenarios).
                    if pending.gas_price_positive {
                        // SAFETY: state_ptr is valid for the lifetime of this block.
                        if let Ok(arb_state) =
                            ArbosState::open(state_ptr, SystemBurner::new(None, false))
                        {
                            let _ = arb_state.l2_pricing_state.grow_backlog(
                                result.compute_gas_for_backlog,
                                pending.charged_multi_gas,
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
                    let db: &mut State<DB> = self.inner.evm_mut().db_mut();
                    apply_fee_distribution(db, dist, None);

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
                        // Go skips backlog update when gas price is zero.
                        if pending.gas_price_positive {
                            let _ = arb_state.l2_pricing_state.grow_backlog(
                                dist.compute_gas_for_backlog,
                                used_multi_gas,
                            );
                        }
                        if !dist.l1_fees_to_add.is_zero() {
                            let _ = arb_state
                                .l1_pricing_state
                                .add_to_l1_fees_available(dist.l1_fees_to_add);
                        }
                        let _ = arb_state
                            .l1_pricing_state
                            .add_to_units_since_update(pending.calldata_units);
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

        Ok(gas_used)
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
// Helpers
// ---------------------------------------------------------------------------

/// Adjust gas_used in an `ExecutionResult` by adding extra gas.
///
/// Used to account for poster gas (L1 data cost) which is deducted before
/// EVM execution but must be reflected in the receipt's gas_used.
fn adjust_result_gas_used<H>(result: &mut ExecutionResult<H>, extra_gas: u64) {
    match result {
        ExecutionResult::Success { gas_used, .. } => *gas_used += extra_gas,
        ExecutionResult::Revert { gas_used, .. } => *gas_used += extra_gas,
        ExecutionResult::Halt { gas_used, .. } => *gas_used += extra_gas,
    }
}

/// Mint balance to an address in the EVM state.
fn mint_balance<DB: Database>(state: &mut State<DB>, address: Address, amount: U256) {
    if amount.is_zero() || address == Address::ZERO {
        return;
    }
    let _ = state.load_cache_account(address);
    let amount_u128: u128 = amount.try_into().unwrap_or(u128::MAX);
    let _ = state.increment_balances(core::iter::once((address, amount_u128)));
}

/// Burn (deduct) balance from an address in the EVM state.
fn burn_balance<DB: Database>(state: &mut State<DB>, address: Address, amount: U256) {
    if amount.is_zero() {
        return;
    }
    if let Ok(Some(mut info)) = revm::Database::basic(state, address) {
        info.balance = info.balance.saturating_sub(amount);
        let mut account = revm_state::Account::from(info);
        account.mark_touch();
        state.commit_iter(&mut core::iter::once((address, account)));
    }
}

/// Increment the nonce of an account in the EVM state.
fn increment_nonce<DB: Database>(state: &mut State<DB>, address: Address) {
    if let Ok(Some(mut info)) = revm::Database::basic(state, address) {
        info.nonce += 1;
        let mut account = revm_state::Account::from(info);
        account.mark_touch();
        state.commit_iter(&mut core::iter::once((address, account)));
    }
}

/// Read the balance of an account in the EVM state.
fn get_balance<DB: Database>(state: &mut State<DB>, address: Address) -> U256 {
    match revm::Database::basic(state, address) {
        Ok(Some(info)) => info.balance,
        _ => U256::ZERO,
    }
}

/// Transfer balance between two addresses in the EVM state.
fn transfer_balance<DB: Database>(
    state: &mut State<DB>,
    from: Address,
    to: Address,
    amount: U256,
) {
    if amount.is_zero() || from == to {
        return;
    }
    burn_balance(state, from, amount);
    mint_balance(state, to, amount);
}

/// Transfer balance with balance check. Returns false if sender has
/// insufficient funds (no state changes in that case).
fn try_transfer_balance<DB: Database>(
    state: &mut State<DB>,
    from: Address,
    to: Address,
    amount: U256,
) -> bool {
    if amount.is_zero() || from == to {
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

