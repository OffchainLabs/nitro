//! Block producer implementation.
//!
//! Produces blocks from L1 incoming messages by parsing transactions,
//! executing them against the current state, and persisting the results.

use std::sync::Arc;

use alloy_consensus::transaction::{Recovered, SignerRecoverable};
use alloy_consensus::{Block, BlockBody, BlockHeader, Header, TxReceipt, proofs, EMPTY_OMMER_ROOT_HASH};
use alloy_eips::eip2718::Decodable2718;
use alloy_evm::block::{BlockExecutor, BlockExecutorFactory};
use alloy_evm::EvmFactory;
use alloy_primitives::{Address, Bytes, B64, B256, U256};
use alloy_rpc_types_eth::BlockNumberOrTag;
use parking_lot::Mutex;
use reth_chainspec::ChainSpec;
use reth_evm::ConfigureEvm;
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider, StateProviderFactory};
use reth_primitives_traits::{SealedHeader, logs_bloom};
use reth_revm::database::StateProviderDatabase;
use reth_storage_api::StateProvider;
use reth_trie_common::HashedPostState;
use revm::database::{BundleState, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use tracing::{debug, info, warn};

use arb_evm::config::{ArbEvmConfig, arbos_version_from_mix_hash};
use arb_primitives::signed_tx::ArbTransactionSigned;
use arb_primitives::tx_types::ArbInternalTx;
use arb_rpc::block_producer::{BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock};
use arbos::arbos_types::{L1_MESSAGE_TYPE_INITIALIZE, parse_init_message};
use arbos::header::{ArbHeaderInfo, derive_arb_header_info};
use arbos::internal_tx;
use arbos::parse_l2::{ParsedTransaction, parse_l2_transactions, parsed_tx_to_signed};

use crate::genesis;

/// Type-erased block persister.
///
/// Wraps the concrete persistence operations so the block producer
/// does not need to carry `DatabaseProviderFactory` and `CanonChainTracker`
/// trait bounds, which cannot be threaded through reth's node builder
/// without modifying upstream traits.
pub(crate) struct ErasedPersister {
    /// Persist a sealed block with execution output to the database.
    persist_fn: Box<
        dyn Fn(
                &reth_primitives_traits::SealedBlock<Block<ArbTransactionSigned>>,
                Vec<arb_primitives::ArbReceipt>,
                BundleState,
            ) -> Result<(), BlockProducerError>
            + Send
            + Sync,
    >,
}

impl ErasedPersister {
    fn persist(
        &self,
        sealed: &reth_primitives_traits::SealedBlock<Block<ArbTransactionSigned>>,
        receipts: Vec<arb_primitives::ArbReceipt>,
        bundle_state: BundleState,
    ) -> Result<(), BlockProducerError> {
        (self.persist_fn)(sealed, receipts, bundle_state)
    }
}

/// Concrete block producer backed by reth's database.
pub struct ArbBlockProducer<Provider> {
    provider: Provider,
    chain_spec: Arc<ChainSpec>,
    evm_config: ArbEvmConfig,
    persister: ErasedPersister,
    /// Mutex to serialize block production.
    produce_lock: Mutex<()>,
    /// Whether ArbOS genesis initialization has been done.
    genesis_initialized: Mutex<bool>,
}

impl<Provider> ArbBlockProducer<Provider> {
    /// Create a new block producer.
    ///
    /// The `persist_fn` closure handles writing blocks and state to the database.
    /// It captures the concrete provider type so the producer itself
    /// does not need `DatabaseProviderFactory` bounds.
    pub fn new(
        provider: Provider,
        chain_spec: Arc<ChainSpec>,
        evm_config: ArbEvmConfig,
        persist_fn: impl Fn(
                &reth_primitives_traits::SealedBlock<Block<ArbTransactionSigned>>,
                Vec<arb_primitives::ArbReceipt>,
                BundleState,
            ) -> Result<(), BlockProducerError>
            + Send
            + Sync
            + 'static,
    ) -> Self {
        Self {
            provider,
            chain_spec,
            evm_config,
            persister: ErasedPersister {
                persist_fn: Box::new(persist_fn),
            },
            produce_lock: Mutex::new(()),
            genesis_initialized: Mutex::new(false),
        }
    }
}

impl<Provider> ArbBlockProducer<Provider>
where
    Provider: BlockNumReader
        + BlockReaderIdExt
        + HeaderProvider<Header = Header>
        + StateProviderFactory
        + Send
        + Sync
        + 'static,
{
    /// Get the current head block number.
    fn head_block_number(&self) -> Result<u64, BlockProducerError> {
        self.provider
            .last_block_number()
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))
    }

    /// Get the parent sealed header for block production.
    fn parent_header(
        &self,
        head_num: u64,
    ) -> Result<SealedHeader<Header>, BlockProducerError> {
        self.provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(head_num))
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?
            .ok_or_else(|| {
                BlockProducerError::StateAccess(format!("Parent block {head_num} not found"))
            })
    }

    /// Handle the Initialize message (Kind=11).
    ///
    /// This initializes ArbOS state in the database and produces the genesis block.
    fn handle_initialize_message(
        &self,
        input: &BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        let init_msg = parse_init_message(&input.l2_msg)
            .map_err(|e| BlockProducerError::Parse(format!("init message: {e}")))?;

        let chain_id = self.chain_spec.chain().id();

        info!(
            target: "block_producer",
            chain_id,
            init_chain_id = %init_msg.chain_id,
            initial_l1_base_fee = %init_msg.initial_l1_base_fee,
            "Processing Initialize message"
        );

        // Open state at genesis (latest = genesis state from alloc).
        let state_provider = self
            .provider
            .latest()
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?;

        let mut db = StateBuilder::new()
            .with_database(StateProviderDatabase::new(state_provider.as_ref()))
            .with_bundle_update()
            .build();

        // Initialize ArbOS state.
        genesis::initialize_arbos_state(
            &mut db,
            &init_msg,
            chain_id,
            genesis::INITIAL_ARBOS_VERSION,
            genesis::DEFAULT_CHAIN_OWNER,
        )
        .map_err(|e| BlockProducerError::Execution(e))?;

        // Merge state changes and produce the genesis block.
        db.merge_transitions(BundleRetention::Reverts);
        let bundle = db.take_bundle();

        // Compute state root over the genesis alloc + ArbOS init changes.
        let hashed_state =
            HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(bundle.state());
        let state_root = state_provider
            .state_root(hashed_state)
            .map_err(|e| BlockProducerError::Execution(format!("state root: {e}")))?;

        // Read the send root from the ArbOS state via the bundle.
        let arb_info = derive_header_info_from_state(state_provider.as_ref(), &bundle);

        let mix_hash = arb_info
            .as_ref()
            .map(|info| info.compute_mix_hash())
            .unwrap_or_default();

        let extra_data: Bytes = arb_info
            .as_ref()
            .map(|info| {
                let mut data = info.send_root.to_vec();
                data.resize(32, 0);
                data.into()
            })
            .unwrap_or_default();

        let send_root = arb_info
            .as_ref()
            .map(|info| info.send_root)
            .unwrap_or(B256::ZERO);

        // Build genesis block header.
        let head_num = self.head_block_number()?;
        let parent_header = self.parent_header(head_num)?;
        let l2_block_number = head_num + 1;

        let header = Header {
            parent_hash: parent_header.hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: input.sender,
            state_root,
            transactions_root: proofs::calculate_transaction_root::<ArbTransactionSigned>(&[]),
            receipts_root: proofs::calculate_receipt_root::<arb_primitives::ArbReceipt>(&[]),
            withdrawals_root: None,
            logs_bloom: Default::default(),
            timestamp: input.l1_timestamp.max(parent_header.timestamp()),
            mix_hash,
            nonce: B64::from(input.delayed_messages_read.to_be_bytes()),
            base_fee_per_gas: parent_header.base_fee_per_gas(),
            number: l2_block_number,
            gas_limit: parent_header.gas_limit(),
            difficulty: U256::from(1),
            gas_used: 0,
            extra_data,
            parent_beacon_block_root: None,
            blob_gas_used: None,
            excess_blob_gas: None,
            requests_hash: None,
        };

        let block = Block::<ArbTransactionSigned> {
            header: header.clone(),
            body: BlockBody {
                transactions: vec![],
                ommers: Default::default(),
                withdrawals: None,
            },
        };

        let sealed = reth_primitives_traits::SealedBlock::seal_slow(block);
        let block_hash = sealed.hash();

        // Persist block and state changes.
        self.persister.persist(&sealed, vec![], bundle)?;

        info!(
            target: "block_producer",
            block_num = l2_block_number,
            ?block_hash,
            ?send_root,
            ?state_root,
            "Produced genesis init block"
        );

        *self.genesis_initialized.lock() = true;

        Ok(ProducedBlock {
            block_hash,
            send_root,
        })
    }

    /// Produce a block with full transaction execution.
    fn produce_block_with_execution(
        &self,
        input: &BlockProductionInput,
        parsed_txs: Vec<ParsedTransaction>,
    ) -> Result<ProducedBlock, BlockProducerError> {
        let head_num = self.head_block_number()?;
        let parent_header = self.parent_header(head_num)?;
        let l2_block_number = head_num + 1;

        let timestamp = input.l1_timestamp.max(parent_header.timestamp());
        let time_passed = timestamp.saturating_sub(parent_header.timestamp());

        let parent_mix_hash = parent_header.mix_hash().unwrap_or_default();
        let parent_arbos_version = arbos_version_from_mix_hash(&parent_mix_hash);

        // Build the EVM environment for this block.
        let l1_block_number = input.l1_block_number;
        let arbos_version = parent_arbos_version; // May upgrade during StartBlock

        // Construct a provisional mix_hash for the EVM environment.
        let send_count = {
            let mut buf = [0u8; 8];
            buf.copy_from_slice(&parent_mix_hash.0[0..8]);
            u64::from_be_bytes(buf)
        };
        let provisional_mix_hash = compute_mix_hash(send_count, l1_block_number, arbos_version);

        // Build a provisional header for the EVM config.
        let provisional_header = Header {
            parent_hash: parent_header.hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: input.sender,
            state_root: B256::ZERO, // placeholder
            transactions_root: B256::ZERO,
            receipts_root: B256::ZERO,
            withdrawals_root: None,
            logs_bloom: Default::default(),
            timestamp,
            mix_hash: provisional_mix_hash,
            nonce: B64::from(input.delayed_messages_read.to_be_bytes()),
            base_fee_per_gas: parent_header.base_fee_per_gas(),
            number: l2_block_number,
            gas_limit: parent_header.gas_limit(),
            difficulty: U256::from(1),
            gas_used: 0,
            extra_data: Default::default(),
            parent_beacon_block_root: None,
            blob_gas_used: None,
            excess_blob_gas: None,
            requests_hash: None,
        };

        let evm_env = self
            .evm_config
            .evm_env(&provisional_header)
            .map_err(|_| BlockProducerError::Execution("evm_env construction failed".into()))?;

        // Open state at parent block.
        let state_provider = self
            .provider
            .latest()
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?;

        let mut db = StateBuilder::new()
            .with_database(StateProviderDatabase::new(state_provider.as_ref()))
            .with_bundle_update()
            .build();

        let chain_id = self.chain_spec.chain().id();

        // Build execution context: extra_data carries send_root + delayed_messages_read.
        let parent_extra = parent_header.extra_data().to_vec();
        let mut exec_extra = parent_extra.clone();
        // Append delayed_messages_read as bytes 32..39.
        exec_extra.resize(32, 0);
        exec_extra.extend_from_slice(&input.delayed_messages_read.to_be_bytes());

        let exec_ctx = alloy_evm::eth::EthBlockExecutionCtx {
            tx_count_hint: Some(parsed_txs.len() + 2), // +2 for internal txs
            parent_hash: parent_header.hash(),
            parent_beacon_block_root: None,
            ommers: &[],
            withdrawals: None,
            extra_data: exec_extra.into(),
        };

        // Create the block executor via the factory.
        let evm = self
            .evm_config
            .block_executor_factory()
            .evm_factory()
            .create_evm(&mut db, evm_env.clone());
        let mut executor = self
            .evm_config
            .block_executor_factory()
            .create_executor(evm, exec_ctx);

        // Apply pre-execution changes (loads ArbOS state, fee accounts, block hashes).
        executor
            .apply_pre_execution_changes()
            .map_err(|e| BlockProducerError::Execution(format!("pre-exec: {e}")))?;

        let mut all_txs: Vec<ArbTransactionSigned> = Vec::new();

        // 1. Generate and execute the StartBlock internal tx (always first).
        let l1_base_fee = input.l1_base_fee.unwrap_or(U256::ZERO);
        let start_block_data = internal_tx::encode_start_block(
            l1_base_fee,
            l1_block_number,
            l2_block_number,
            time_passed,
        );

        let start_block_tx = create_internal_tx(chain_id, &start_block_data);
        execute_and_commit_tx(&mut executor, &start_block_tx, "StartBlock")?;
        all_txs.push(start_block_tx);

        // 2. Generate batch posting report if batch data stats are present.
        if let Some((length, non_zeros)) = input.batch_data_stats {
            let batch_report_data = internal_tx::encode_batch_posting_report_v2(
                input.l1_timestamp,
                input.sender,
                0, // batch_number: not critical for state machine
                length,
                non_zeros,
                0, // extra_gas
                l1_base_fee,
            );

            let batch_tx = create_internal_tx(chain_id, &batch_report_data);
            execute_and_commit_tx(&mut executor, &batch_tx, "BatchPostingReportV2")?;
            all_txs.push(batch_tx);
        } else if let Some(batch_gas_cost) = input.batch_gas_cost {
            let batch_report_data = internal_tx::encode_batch_posting_report(
                input.l1_timestamp,
                input.sender,
                0, // batch_number
                batch_gas_cost,
                l1_base_fee,
            );

            let batch_tx = create_internal_tx(chain_id, &batch_report_data);
            execute_and_commit_tx(&mut executor, &batch_tx, "BatchPostingReport")?;
            all_txs.push(batch_tx);
        }

        // 3. Execute parsed user transactions.
        for parsed in &parsed_txs {
            match parsed {
                ParsedTransaction::BatchPostingReport { .. }
                | ParsedTransaction::InternalStartBlock { .. } => {
                    // These are handled above as internal txs, skip.
                    continue;
                }
                _ => {}
            }

            let signed_tx = match parsed_tx_to_signed(parsed, chain_id) {
                Some(tx) => tx,
                None => {
                    debug!(
                        target: "block_producer",
                        ?parsed,
                        "Skipping unparseable transaction"
                    );
                    continue;
                }
            };

            // Recover the signer to get a Recovered<ArbTransactionSigned>.
            let recovered = match signed_tx.clone().try_into_recovered() {
                Ok(r) => r,
                Err(e) => {
                    warn!(
                        target: "block_producer",
                        error = %e,
                        "Failed to recover tx sender, skipping"
                    );
                    continue;
                }
            };

            match executor.execute_transaction_without_commit(recovered) {
                Ok(result) => {
                    match executor.commit_transaction(result) {
                        Ok(_gas_used) => {
                            all_txs.push(signed_tx);
                        }
                        Err(e) => {
                            warn!(
                                target: "block_producer",
                                error = %e,
                                "Failed to commit transaction"
                            );
                        }
                    }
                }
                Err(ref e) if e.to_string().contains("block gas limit reached") => {
                    debug!(
                        target: "block_producer",
                        "Block gas limit reached, stopping execution"
                    );
                    break;
                }
                Err(e) => {
                    warn!(
                        target: "block_producer",
                        error = %e,
                        "Transaction execution failed, skipping"
                    );
                }
            }
        }

        // Finalize execution: finish() consumes the executor and returns
        // the EVM and BlockExecutionResult containing receipts.
        let (_, exec_result) = executor
            .finish()
            .map_err(|e| BlockProducerError::Execution(format!("finish: {e}")))?;

        let receipts: Vec<arb_primitives::ArbReceipt> = exec_result.receipts;

        // After executor is dropped, we can access the db again.
        db.merge_transitions(BundleRetention::Reverts);
        let bundle = db.take_bundle();

        // Compute the state root from the bundle state overlay.
        let hashed_state =
            HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(bundle.state());
        let state_root = state_provider
            .state_root(hashed_state)
            .map_err(|e| BlockProducerError::Execution(format!("state root: {e}")))?;

        // Derive header info (send_root, send_count, etc.) from post-execution state.
        let arb_info = derive_header_info_from_state(state_provider.as_ref(), &bundle);

        let final_mix_hash = arb_info
            .as_ref()
            .map(|info| info.compute_mix_hash())
            .unwrap_or(provisional_mix_hash);

        let extra_data: Bytes = arb_info
            .as_ref()
            .map(|info| {
                let mut data = info.send_root.to_vec();
                data.resize(32, 0);
                data.into()
            })
            .unwrap_or_else(|| {
                let mut data = parent_extra.clone();
                data.resize(32, 0);
                data.into()
            });

        let send_root = arb_info
            .as_ref()
            .map(|info| info.send_root)
            .unwrap_or_else(|| {
                if parent_extra.len() >= 32 {
                    B256::from_slice(&parent_extra[..32])
                } else {
                    B256::ZERO
                }
            });

        // Compute receipt-derived fields.
        let gas_used = exec_result.gas_used;
        let logs_bloom_val = logs_bloom(receipts.iter().flat_map(|r| r.logs()));

        let transactions_root =
            proofs::calculate_transaction_root::<ArbTransactionSigned>(&all_txs);
        let receipts_root = proofs::calculate_receipt_root(
            &receipts.iter().map(|r| r.with_bloom_ref()).collect::<Vec<_>>(),
        );

        let header = Header {
            parent_hash: parent_header.hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: input.sender,
            state_root,
            transactions_root,
            receipts_root,
            withdrawals_root: None,
            logs_bloom: logs_bloom_val,
            timestamp,
            mix_hash: final_mix_hash,
            nonce: B64::from(input.delayed_messages_read.to_be_bytes()),
            base_fee_per_gas: parent_header.base_fee_per_gas(),
            number: l2_block_number,
            gas_limit: parent_header.gas_limit(),
            difficulty: U256::from(1),
            gas_used: gas_used as u64,
            extra_data,
            parent_beacon_block_root: None,
            blob_gas_used: None,
            excess_blob_gas: None,
            requests_hash: None,
        };

        let block = Block::<ArbTransactionSigned> {
            header,
            body: BlockBody {
                transactions: all_txs,
                ommers: Default::default(),
                withdrawals: None,
            },
        };

        let sealed = reth_primitives_traits::SealedBlock::seal_slow(block);
        let block_hash = sealed.hash();

        // Persist block, receipts, and state changes.
        self.persister.persist(&sealed, receipts, bundle)?;

        info!(
            target: "block_producer",
            block_num = l2_block_number,
            ?block_hash,
            ?send_root,
            ?state_root,
            num_txs = sealed.body().transactions.len(),
            gas_used,
            "Produced block"
        );

        Ok(ProducedBlock {
            block_hash,
            send_root,
        })
    }

    /// Produce a minimal block for messages with no transactions.
    #[allow(dead_code)]
    fn produce_empty_block(
        &self,
        input: &BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        // Even empty blocks need to execute the StartBlock internal tx
        // so that ArbOS state updates (pricing, retryable reaping) happen.
        self.produce_block_with_execution(input, vec![])
    }
}

#[async_trait::async_trait]
impl<Provider> BlockProducer for ArbBlockProducer<Provider>
where
    Provider: BlockNumReader
        + BlockReaderIdExt
        + HeaderProvider<Header = Header>
        + StateProviderFactory
        + Send
        + Sync
        + 'static,
{
    async fn produce_block(
        &self,
        msg_idx: u64,
        input: BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        let _lock = self.produce_lock.lock();

        // Validate that this message is the next expected one.
        let head_num = self.head_block_number()?;
        let expected_block = head_num + 1;
        let actual_block = msg_idx;

        if expected_block != actual_block {
            return Err(BlockProducerError::Unexpected(format!(
                "Expected block {expected_block} but got msg_idx {msg_idx} (block {actual_block})"
            )));
        }

        // Handle Initialize message (Kind=11) — bootstraps ArbOS state.
        if input.kind == L1_MESSAGE_TYPE_INITIALIZE {
            return self.handle_initialize_message(&input);
        }

        // Parse L2 transactions from the message.
        let parsed_txs = parse_l2_transactions(
            input.kind,
            input.sender,
            &input.l2_msg,
            input.request_id,
            input.l1_base_fee,
        )
        .map_err(|e| BlockProducerError::Parse(e.to_string()))?;

        debug!(
            target: "block_producer",
            msg_idx,
            kind = input.kind,
            num_txs = parsed_txs.len(),
            "Parsed L1 message"
        );

        self.produce_block_with_execution(&input, parsed_txs)
    }
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

/// Create an internal transaction (type 0x6A).
fn create_internal_tx(chain_id: u64, data: &[u8]) -> ArbTransactionSigned {
    use arb_primitives::signed_tx::ArbTypedTransaction;
    let tx = ArbTypedTransaction::Internal(ArbInternalTx {
        chain_id: U256::from(chain_id),
        data: Bytes::copy_from_slice(data),
    });
    let sig = alloy_primitives::Signature::new(U256::ZERO, U256::ZERO, false);
    ArbTransactionSigned::new_unhashed(tx, sig)
}

/// Execute and commit an internal transaction via the block executor.
fn execute_and_commit_tx<E>(
    executor: &mut E,
    tx: &ArbTransactionSigned,
    label: &str,
) -> Result<(), BlockProducerError>
where
    E: BlockExecutor<Transaction = ArbTransactionSigned>,
{
    let recovered = tx
        .clone()
        .try_into_recovered()
        .map_err(|e| BlockProducerError::Execution(format!("{label} recovery: {e}")))?;

    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| BlockProducerError::Execution(format!("{label} execution: {e}")))?;

    executor
        .commit_transaction(result)
        .map_err(|e| BlockProducerError::Execution(format!("{label} commit: {e}")))?;

    Ok(())
}

/// Decode a scheduled retry tx from its encoded bytes and recover the sender.
fn _decode_and_recover_retry_tx(
    encoded: &[u8],
) -> Option<Recovered<ArbTransactionSigned>> {
    let tx = ArbTransactionSigned::decode_2718(&mut &encoded[..]).ok()?;
    tx.try_into_recovered().ok()
}

/// Construct a mix_hash from send_count, l1_block_number, and arbos_version.
fn compute_mix_hash(send_count: u64, l1_block_number: u64, arbos_version: u64) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[0..8].copy_from_slice(&send_count.to_be_bytes());
    bytes[8..16].copy_from_slice(&l1_block_number.to_be_bytes());
    bytes[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    B256::from(bytes)
}

/// Derive ArbHeaderInfo from post-execution state.
fn derive_header_info_from_state(
    state_provider: &dyn StateProvider,
    bundle_state: &BundleState,
) -> Option<ArbHeaderInfo> {
    let read_slot = |addr: Address, slot: B256| -> Option<U256> {
        if let Some(account) = bundle_state.state.get(&addr) {
            let slot_u256 = U256::from_be_bytes(slot.0);
            if let Some(storage_slot) = account.storage.get(&slot_u256) {
                return Some(storage_slot.present_value);
            }
        }
        state_provider.storage(addr, slot.into()).ok().flatten()
    };

    derive_arb_header_info(&read_slot)
}
