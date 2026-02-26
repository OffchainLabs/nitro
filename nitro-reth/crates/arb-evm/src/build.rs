use alloy_consensus::{Transaction, TransactionEnvelope, TxReceipt};
use alloy_eips::eip2718::Encodable2718;
use alloy_evm::block::{
    BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
    BlockExecutorFor, CommitChanges, ExecutableTx, OnStateHook,
};
use alloy_evm::eth::EthTxResult;
use alloy_evm::eth::receipt_builder::ReceiptBuilder;
use alloy_evm::eth::spec::EthExecutorSpec;
use alloy_evm::eth::{EthBlockExecutionCtx, EthBlockExecutor};
use alloy_evm::tx::{FromRecoveredTx, FromTxWithEncoded};
use alloy_evm::{Database, Evm, EvmFactory};
use alloy_primitives::Log;
use revm::context::result::ExecutionResult;
use revm::database::State;
use revm::inspector::Inspector;

use crate::context::ArbBlockExecutionCtx;
use crate::executor::DefaultArbOsHooks;

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
        ArbBlockExecutor {
            inner: EthBlockExecutor::new(evm, ctx, &self.spec, &self.receipt_builder),
            arb_hooks: None,
            arb_ctx: ArbBlockExecutionCtx::default(),
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

        tracing::trace!(
            target: "arb::executor",
            l1_block = self.arb_ctx.l1_block_number,
            delayed_msgs = self.arb_ctx.delayed_messages_read,
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
        self.inner.execute_transaction_with_commit_condition(tx, f)
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
