use alloc::sync::Arc;
use alloc::vec::Vec;

use alloy_consensus::{Block, BlockBody, Header, TxReceipt, EMPTY_OMMER_ROOT_HASH, proofs};
use alloy_evm::block::{BlockExecutionError, BlockExecutionResult, BlockExecutorFactory};
use alloy_evm::eth::EthBlockExecutionCtx;
use alloy_primitives::{B64, B256, U256};
use reth_evm::execute::{BlockAssembler, BlockAssemblerInput};
use reth_primitives_traits::{Receipt, SignedTransaction, logs_bloom};
use revm::context::Block as RevmBlock;

use arbos::header::{ArbHeaderInfo, derive_arb_header_info};

/// Arbitrum block assembler.
///
/// Constructs block headers with Arbitrum-specific fields:
/// - `extra_data`: send root in first 32 bytes
/// - `mix_hash`: encodes (send_count, l1_block_number, arbos_version)
/// - `nonce`: delayed_messages_read
/// - `difficulty`: always 1
#[derive(Debug, Clone)]
pub struct ArbBlockAssembler<ChainSpec> {
    chain_spec: Arc<ChainSpec>,
}

impl<ChainSpec> ArbBlockAssembler<ChainSpec> {
    pub fn new(chain_spec: Arc<ChainSpec>) -> Self {
        Self { chain_spec }
    }
}

impl<F, ChainSpec> BlockAssembler<F> for ArbBlockAssembler<ChainSpec>
where
    F: for<'a> BlockExecutorFactory<
        ExecutionCtx<'a> = EthBlockExecutionCtx<'a>,
        Transaction: SignedTransaction,
        Receipt: Receipt,
    >,
    ChainSpec: Send + Sync + Unpin + 'static,
{
    type Block = Block<F::Transaction>;

    fn assemble_block(
        &self,
        input: BlockAssemblerInput<'_, '_, F>,
    ) -> Result<Self::Block, BlockExecutionError> {
        let BlockAssemblerInput {
            evm_env,
            execution_ctx: ctx,
            parent: _,
            transactions,
            output: BlockExecutionResult { receipts, gas_used, .. },
            bundle_state,
            state_provider,
            state_root,
            ..
        } = input;

        let timestamp = evm_env.block_env.timestamp().saturating_to();

        let transactions_root = proofs::calculate_transaction_root(&transactions);
        let receipts_root = proofs::calculate_receipt_root(
            &receipts.iter().map(|r| r.with_bloom_ref()).collect::<Vec<_>>(),
        );
        let logs_bloom = logs_bloom(receipts.iter().flat_map(|r| r.logs()));

        // Derive send root, send count, l1 block number, and arbos version
        // from the post-execution state.
        let arb_info = derive_header_info_from_state(state_provider, bundle_state);

        let mix_hash = arb_info
            .as_ref()
            .map(|info| info.compute_mix_hash())
            .unwrap_or_else(|| evm_env.block_env.prevrandao().unwrap_or_default());

        let extra_data = arb_info
            .as_ref()
            .map(|info| {
                let mut data = info.send_root.to_vec();
                data.resize(32, 0);
                data.into()
            })
            .unwrap_or_else(|| ctx.extra_data.clone());

        // Decode delayed_messages_read from bytes 32-39 of the execution context's extra_data.
        let extra_bytes = ctx.extra_data.as_ref();
        let delayed_messages_read = if extra_bytes.len() >= 40 {
            let mut buf = [0u8; 8];
            buf.copy_from_slice(&extra_bytes[32..40]);
            u64::from_be_bytes(buf)
        } else {
            0
        };

        let header = Header {
            parent_hash: ctx.parent_hash,
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: evm_env.block_env.beneficiary(),
            state_root,
            transactions_root,
            receipts_root,
            withdrawals_root: None,
            logs_bloom,
            timestamp,
            mix_hash,
            nonce: B64::from(delayed_messages_read.to_be_bytes()),
            base_fee_per_gas: Some(evm_env.block_env.basefee()),
            number: evm_env.block_env.number().saturating_to(),
            gas_limit: evm_env.block_env.gas_limit(),
            difficulty: U256::from(1),
            gas_used: *gas_used,
            extra_data,
            parent_beacon_block_root: ctx.parent_beacon_block_root,
            blob_gas_used: Some(0),
            excess_blob_gas: Some(0),
            requests_hash: None,
        };

        Ok(Block {
            header,
            body: BlockBody {
                transactions,
                ommers: Default::default(),
                withdrawals: None,
            },
        })
    }
}

/// Derive ArbHeaderInfo by reading ArbOS state from the post-execution state.
///
/// Combines bundle_state (pending changes) with state_provider (committed state)
/// to read the Merkle accumulator's send root/count and L1 block number.
fn derive_header_info_from_state(
    state_provider: &dyn reth_storage_api::StateProvider,
    bundle_state: &revm_database::BundleState,
) -> Option<ArbHeaderInfo> {
    let read_slot = |addr: alloy_primitives::Address, slot: B256| -> Option<U256> {
        // Check bundle state first (post-execution changes).
        if let Some(account) = bundle_state.state.get(&addr) {
            let slot_u256 = U256::from_be_bytes(slot.0);
            if let Some(storage_slot) = account.storage.get(&slot_u256) {
                return Some(storage_slot.present_value);
            }
        }
        // Fall back to the committed state provider.
        state_provider.storage(addr, slot.into()).ok().flatten()
    };

    derive_arb_header_info(&read_slot)
}
