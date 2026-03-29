//! Block producer implementation.
//!
//! Produces blocks from L1 incoming messages by parsing transactions,
//! executing them against the current state, and persisting the results.

use std::sync::{
    atomic::{AtomicBool, AtomicU64, Ordering},
    Arc,
};

use alloy_consensus::{
    proofs, transaction::SignerRecoverable, Block, BlockBody, BlockHeader, Header, TxReceipt,
    EMPTY_OMMER_ROOT_HASH,
};
use alloy_eips::eip2718::Decodable2718;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    EvmFactory,
};
use alloy_primitives::{Address, Bytes, B256, B64, U256};
use alloy_rpc_types_eth::BlockNumberOrTag;
use parking_lot::Mutex;
use reth_chain_state::{CanonicalInMemoryState, ExecutedBlock, NewCanonicalChain};
use reth_chainspec::ChainSpec;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::{logs_bloom, NodePrimitives, SealedHeader};
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider, StateProviderFactory};
use reth_revm::database::StateProviderDatabase;
use reth_storage_api::StateProvider;
use reth_trie_common::{HashedPostState, TrieInput};
use revm::database::{BundleState, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use tracing::{debug, info, warn};

use arb_evm::config::{arbos_version_from_mix_hash, ArbEvmConfig};
use arb_primitives::{signed_tx::ArbTransactionSigned, tx_types::ArbInternalTx, ArbPrimitives};
use arb_rpc::block_producer::{
    BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock,
};
use arbos::{
    arbos_types::parse_init_message,
    header::{derive_arb_header_info, ArbHeaderInfo},
    internal_tx,
    parse_l2::{parse_l2_transactions, parsed_tx_to_signed, ParsedTransaction},
};

use crate::genesis;

/// Trait to access the in-memory canonical state from a provider.
///
/// `BlockchainProvider` has `canonical_in_memory_state()` as an inherent method
/// but it's not exposed via any reth trait. This trait bridges that gap so
/// the block producer can receive the handle generically.
pub trait InMemoryStateAccess {
    type Primitives: NodePrimitives;
    fn canonical_in_memory_state(&self) -> CanonicalInMemoryState<Self::Primitives>;
}

/// Implement `InMemoryStateAccess` for reth's `BlockchainProvider`.
impl<N> InMemoryStateAccess for reth_provider::providers::BlockchainProvider<N>
where
    N: reth_provider::providers::ProviderNodeTypes,
{
    type Primitives = N::Primitives;
    fn canonical_in_memory_state(&self) -> CanonicalInMemoryState<Self::Primitives> {
        self.canonical_in_memory_state()
    }
}

/// Default number of blocks to buffer before flushing via save_blocks(Full).
pub const DEFAULT_FLUSH_INTERVAL: u64 = 128;

/// Block producer using reth's save_blocks(Full) for persistence.
///
/// Execution happens on the producer thread. After execution, blocks are:
/// 1. Added to `CanonicalInMemoryState` (immediate, for next block's state)
/// 2. Accumulated until flush_interval, then persisted via save_blocks(Full)
///
/// save_blocks(Full) writes ALL tables including history indices — this is
/// the same persistence path as reth's Pipeline/ExecutionStage.
pub struct ArbBlockProducer<Provider> {
    provider: Provider,
    chain_spec: Arc<ChainSpec>,
    evm_config: ArbEvmConfig,
    in_memory_state: CanonicalInMemoryState<ArbPrimitives>,
    head_block_num: AtomicU64,
    blocks_since_flush: AtomicU64,
    flush_interval: u64,
    accumulated_trie_input: Mutex<TrieInput>,
    flushing_trie_input: Mutex<Option<TrieInput>>,
    pending_flush: AtomicBool,
    produce_lock: Mutex<()>,
    cached_init: Mutex<Option<arbos::arbos_types::ParsedInitMessage>>,
}

impl<Provider> ArbBlockProducer<Provider>
where
    Provider: BlockNumReader,
{
    pub fn new(
        provider: Provider,
        chain_spec: Arc<ChainSpec>,
        evm_config: ArbEvmConfig,
        in_memory_state: CanonicalInMemoryState<ArbPrimitives>,
        flush_interval: u64,
    ) -> Self {
        let head = provider.last_block_number().unwrap_or(0);
        Self {
            provider,
            chain_spec,
            evm_config,
            in_memory_state,
            head_block_num: AtomicU64::new(head),
            blocks_since_flush: AtomicU64::new(0),
            flush_interval,
            accumulated_trie_input: Mutex::new(TrieInput::default()),
            flushing_trie_input: Mutex::new(None),
            pending_flush: AtomicBool::new(false),
            produce_lock: Mutex::new(()),
            cached_init: Mutex::new(None),
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
    /// Get the current head block number (includes in-memory buffered blocks).
    fn head_block_number(&self) -> Result<u64, BlockProducerError> {
        let head = self.head_block_num.load(Ordering::SeqCst);
        if head > 0 {
            Ok(head)
        } else {
            self.provider
                .last_block_number()
                .map_err(|e| BlockProducerError::StateAccess(e.to_string()))
        }
    }

    /// Get the parent sealed header for block production.
    fn parent_header(&self, head_num: u64) -> Result<SealedHeader<Header>, BlockProducerError> {
        self.provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(head_num))
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?
            .ok_or_else(|| {
                BlockProducerError::StateAccess(format!("Parent block {head_num} not found"))
            })
    }

    /// Produce a block with full transaction execution.
    fn produce_block_with_execution(
        &self,
        input: &BlockProductionInput,
        parsed_txs: Vec<ParsedTransaction>,
    ) -> Result<ProducedBlock, BlockProducerError> {
        // Check if a background flush completed.
        if self.pending_flush.load(Ordering::SeqCst) {
            if let Some(result) = crate::launcher::try_flush_result() {
                self.in_memory_state
                    .remove_persisted_blocks(result.last_num_hash);
                *self.flushing_trie_input.lock() = None;
                self.pending_flush.store(false, Ordering::SeqCst);
                info!(
                    target: "block_producer",
                    flushed = result.count,
                    last_block = result.last_num_hash.number,
                    duration_ms = result.duration.as_millis(),
                    "Background flush completed"
                );
            }
        }

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

        // Open state at parent block via block hash (matches reth fork pattern).
        let state_provider = self
            .provider
            .state_by_block_hash(parent_header.hash())
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?;

        // Read the L2 baseFee from the parent's committed state.
        // This is the value written by the parent block's StartBlock — the correct
        // baseFee for this block's header and EVM execution.
        let l2_base_fee = {
            let read_slot = |addr: Address, slot: B256| -> Option<U256> {
                state_provider.storage(addr, slot).ok().flatten()
            };
            arbos::header::read_l2_base_fee(&read_slot).or(parent_header.base_fee_per_gas())
        };

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
            base_fee_per_gas: l2_base_fee,
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

        // without_state_clear() disables EIP-161 empty account pruning.
        // Arbitrum needs this for zombie accounts (e.g. retryable escrow
        // accounts that are created and destroyed within a single block).
        let mut db = StateBuilder::new()
            .with_database(StateProviderDatabase::new(state_provider.as_ref()))
            .with_bundle_update()
            .without_state_clear()
            .build();

        let chain_id = self.chain_spec.chain().id();

        // If Init params were cached, apply ArbOS initialization now.
        // This makes the Init state changes part of block 1's delta so the
        // state root correctly includes both Init and execution changes.
        // Skip if genesis alloc already includes ArbOS state.
        if let Some(init_msg) = self.cached_init.lock().take() {
            if !genesis::is_arbos_initialized(&mut db) {
                info!(
                    target: "block_producer",
                    "Applying cached ArbOS Init during block {} execution",
                    l2_block_number
                );
                genesis::initialize_arbos_state(
                    &mut db,
                    &init_msg,
                    chain_id,
                    genesis::INITIAL_ARBOS_VERSION,
                    genesis::DEFAULT_CHAIN_OWNER,
                )
                .map_err(BlockProducerError::Execution)?;
            } else {
                debug!(
                    target: "block_producer",
                    "ArbOS already initialized in genesis alloc, skipping Init"
                );
            }
        }

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
            .create_arb_executor(evm, exec_ctx, chain_id);
        executor.arb_ctx.l2_block_number = l2_block_number;
        executor.arb_ctx.l1_block_number = l1_block_number;

        // Add the parent hash to the L2 block hash cache for arbBlockHash().
        // The cache persists across blocks (static HashMap), so we only need
        // to add one new entry per block. Old entries remain from previous blocks.
        // First block populates the full 256 entries; subsequent blocks add 1.
        {
            let parent_num = l2_block_number.saturating_sub(1);
            arb_precompiles::set_l2_block_hash(parent_num, parent_header.hash());

            // If cache is mostly empty (first block or after restart), do a full populate.
            if arb_precompiles::get_l2_block_hash(parent_num.saturating_sub(1)).is_none()
                && parent_num > 1
            {
                let mut hash = parent_header.parent_hash();
                for i in 2..=256u64 {
                    let n = l2_block_number.checked_sub(i);
                    if let Some(n) = n {
                        arb_precompiles::set_l2_block_hash(n, hash);
                        match self
                            .provider
                            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(n))
                        {
                            Ok(Some(h)) => hash = h.parent_hash(),
                            _ => break,
                        }
                    }
                }
            }
        }

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

        // 2. Execute parsed user transactions.
        for parsed in &parsed_txs {
            match parsed {
                ParsedTransaction::InternalStartBlock { .. } => {
                    // StartBlock is handled above, skip.
                    continue;
                }
                ParsedTransaction::BatchPostingReport {
                    batch_timestamp,
                    batch_poster,
                    batch_number,
                    l1_base_fee_estimate,
                    extra_gas,
                    ..
                } => {
                    // Delayed message kind=13 contains a batch posting report.
                    // Encode as V1 or V2 based on parent ArbOS version.
                    let report_data =
                        if parent_arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_50 {
                            // V2: pass raw batch data stats + extra_gas.
                            let (length, non_zeros) = input.batch_data_stats.unwrap_or((0, 0));
                            internal_tx::encode_batch_posting_report_v2(
                                *batch_timestamp,
                                *batch_poster,
                                *batch_number,
                                length,
                                non_zeros,
                                *extra_gas,
                                *l1_base_fee_estimate,
                            )
                        } else {
                            // V1: combine legacy gas cost + extra_gas into single field.
                            let legacy_gas = input.batch_gas_cost.unwrap_or(0);
                            let batch_data_gas = legacy_gas.saturating_add(*extra_gas);
                            internal_tx::encode_batch_posting_report(
                                *batch_timestamp,
                                *batch_poster,
                                *batch_number,
                                batch_data_gas,
                                *l1_base_fee_estimate,
                            )
                        };
                    let report_tx = create_internal_tx(chain_id, &report_data);
                    execute_and_commit_tx(&mut executor, &report_tx, "BatchPostingReport")?;
                    all_txs.push(report_tx);
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

                            // Drain and execute any scheduled txs (auto-redeems).
                            // After a SubmitRetryable or manual Redeem precompile call,
                            // the executor queues retry txs that must execute in the
                            // same block, immediately after the triggering tx.
                            loop {
                                let scheduled = executor.drain_scheduled_txs();
                                debug!(
                                    target: "block_producer",
                                    count = scheduled.len(),
                                    "Drained scheduled txs"
                                );
                                if scheduled.is_empty() {
                                    break;
                                }
                                for encoded in scheduled {
                                    let retry_tx: Option<ArbTransactionSigned> =
                                        ArbTransactionSigned::decode_2718(&mut &encoded[..]).ok();
                                    if let Some(retry_tx) = retry_tx {
                                        let retry_signed = retry_tx.clone();
                                        match retry_tx.try_into_recovered() {
                                            Ok(recovered_retry) => {
                                                match executor.execute_transaction_without_commit(
                                                    recovered_retry,
                                                ) {
                                                    Ok(retry_result) => {
                                                        match executor
                                                            .commit_transaction(retry_result)
                                                        {
                                                            Ok(_) => {
                                                                all_txs.push(retry_signed);
                                                            }
                                                            Err(e) => {
                                                                warn!(
                                                                    target: "block_producer",
                                                                    error = %e,
                                                                    "Failed to commit auto-redeem tx"
                                                                );
                                                            }
                                                        }
                                                    }
                                                    Err(e) => {
                                                        warn!(
                                                            target: "block_producer",
                                                            error = %e,
                                                            "Auto-redeem tx execution failed"
                                                        );
                                                    }
                                                }
                                            }
                                            Err(e) => {
                                                warn!(
                                                    target: "block_producer",
                                                    error = %e,
                                                    "Failed to recover auto-redeem tx sender"
                                                );
                                            }
                                        }
                                    }
                                }
                            }
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

        // Extract zombie accounts before finish() consumes the executor.
        // Zombie accounts are empty accounts preserved by pre-Stylus ArbOS
        // (CreateZombieIfDeleted) and must NOT be deleted during EIP-161 cleanup.
        let zombie_accounts = executor.zombie_accounts().clone();
        let finalise_deleted = executor.finalise_deleted().clone();

        // Finalize execution: finish() consumes the executor and returns
        // the EVM and BlockExecutionResult containing receipts.
        let (_, exec_result) = executor
            .finish()
            .map_err(|e| BlockProducerError::Execution(format!("finish: {e}")))?;

        let receipts: Vec<arb_primitives::ArbReceipt> = exec_result.receipts;

        // After executor is dropped, we can access the db again.
        db.merge_transitions(BundleRetention::Reverts);
        let mut bundle = db.take_bundle();

        // Augment bundle with direct cache modifications (bypass txs,
        // post-commit hooks) that didn't go through revm's commit.
        augment_bundle_from_cache(&mut bundle, &db.cache, &*state_provider);

        // Mark per-tx finalise deletions in the bundle.
        // Accounts deleted by per-tx EIP-161 cleanup are kept in the cache
        // as Destroyed (account=None) so augment_bundle_from_cache handles
        // them. This loop serves as a safety net and handles zombie checks.
        //
        // Skip accounts that were later re-created as zombies — those are
        // valid empty accounts that must persist in the trie.
        let keccak_empty_hash = alloy_primitives::B256::from(alloy_primitives::keccak256([]));
        for addr in &finalise_deleted {
            // Zombie accounts were re-created after Finalise deleted them.
            // They're back in cache and handled by augment_bundle_from_cache.
            if zombie_accounts.contains(addr) {
                continue;
            }
            if bundle.state.contains_key(addr) {
                // Account is in bundle from EVM transitions. Check whether
                // it existed in the trie before this block.
                let existed_before = state_provider.basic_account(addr).ok().flatten().is_some();
                if existed_before {
                    // Account was in the trie. Only mark as deleted if it's
                    // still empty — it may have been re-created with non-zero
                    // state (e.g., nonce=1) by a later tx in this block.
                    let still_empty = bundle
                        .state
                        .get(addr)
                        .and_then(|a| a.info.as_ref())
                        .is_none_or(|info| {
                            info.nonce == 0
                                && info.balance.is_zero()
                                && info.code_hash == keccak_empty_hash
                        });
                    if still_empty {
                        if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                            bundle_acct.info = None;
                        }
                    }
                } else {
                    // Account was created within this block. It may have been
                    // emptied and then re-created (e.g., sender emptied after
                    // SubmitRetryable, then nonce incremented during RetryTx).
                    // Only remove if the account is still empty.
                    let still_empty = bundle
                        .state
                        .get(addr)
                        .and_then(|a| a.info.as_ref())
                        .is_none_or(|info| {
                            info.nonce == 0
                                && info.balance.is_zero()
                                && info.code_hash == keccak_empty_hash
                        });
                    if still_empty {
                        bundle.state.remove(addr);
                    }
                }
                continue;
            }
            if let Ok(Some(acct)) = state_provider.basic_account(addr) {
                let was_originally_empty = acct.balance.is_zero()
                    && acct.nonce == 0
                    && acct.bytecode_hash.is_none_or(|h| h == keccak_empty_hash);
                if was_originally_empty {
                    continue;
                }
                bundle.state.insert(
                    *addr,
                    revm_database::BundleAccount {
                        info: None, // signals trie deletion
                        original_info: None,
                        storage: Default::default(),
                        status: revm_database::AccountStatus::Changed,
                    },
                );
            }
        }

        // Filter bundle to only include actually changed storage slots.
        // revm's bundle may include storage slots that were loaded (read) but
        // not modified. Including unchanged slots in the HashedPostState would
        // produce an incorrect state root.
        filter_unchanged_storage(&mut bundle);

        // Delete empty accounts from the bundle (EIP-161).
        // Zombie accounts are preserved.
        delete_empty_accounts(&mut bundle, &zombie_accounts, &*state_provider);

        let hashed_state =
            HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(bundle.state());

        let (state_root, trie_updates) = {
            let mut acc = self.accumulated_trie_input.lock();
            let flushing = self.flushing_trie_input.lock();

            // Build merged input: flushing overlay (if flush in progress) + current + this block.
            // During a flush, the DB hasn't been updated yet, so we need BOTH the flushing
            // overlay (blocks from before flush) and the current accumulator (blocks since flush).
            let mut input = if let Some(ref flushing) = *flushing {
                let mut base = flushing.clone();
                base.nodes.extend(acc.nodes.clone());
                base.state.extend(acc.state.clone());
                base.prefix_sets.extend(acc.prefix_sets.clone());
                base
            } else {
                acc.clone()
            };
            drop(flushing); // release lock before computation

            input.append(hashed_state.clone());
            let (root, updates) = crate::launcher::compute_parallel_state_root(input)
                .map_err(|e| BlockProducerError::Execution(format!("state root: {e}")))?;
            acc.append_cached(updates.clone(), hashed_state.clone());
            (root, updates)
        };

        debug!(
            target: "block_producer",
            changed_accounts = hashed_state.accounts.len(),
            changed_storages = hashed_state.storages.len(),
            total_storage_slots = hashed_state.storages.values().map(|s| s.storage.len()).sum::<usize>(),
            ?state_root,
            "HashedPostState from bundle"
        );

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
            &receipts
                .iter()
                .map(|r| r.with_bloom_ref())
                .collect::<Vec<_>>(),
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
            base_fee_per_gas: l2_base_fee,
            number: l2_block_number,
            gas_limit: parent_header.gas_limit(),
            difficulty: U256::from(1),
            gas_used,
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

        // Buffer block in memory instead of persisting immediately.
        // This avoids a per-block fsync, batching writes for much higher throughput.
        {
            use alloy_evm::block::BlockExecutionResult;
            use reth_chain_state::ComputedTrieData;
            use reth_execution_types::BlockExecutionOutput;
            use reth_primitives_traits::RecoveredBlock;

            let recovered = Arc::new(RecoveredBlock::new_sealed(sealed.clone(), vec![]));
            let exec_output = Arc::new(BlockExecutionOutput {
                state: bundle,
                result: BlockExecutionResult {
                    receipts,
                    requests: Default::default(),
                    gas_used,
                    blob_gas_used: 0,
                },
            });
            let computed = ComputedTrieData {
                hashed_state: Arc::new(hashed_state.into_sorted()),
                trie_updates: Arc::new(trie_updates.into_sorted()),
                anchored_trie_input: None,
            };
            let executed = ExecutedBlock::new(recovered, exec_output, computed);

            self.in_memory_state
                .update_chain(NewCanonicalChain::Commit {
                    new: vec![executed],
                });

            let sealed_header = SealedHeader::new(sealed.header().clone(), sealed.hash());
            self.in_memory_state.set_canonical_head(sealed_header);
        }

        self.head_block_num.store(l2_block_number, Ordering::SeqCst);

        // Start async flush when buffer threshold reached (non-blocking).
        let since_flush = self.blocks_since_flush.fetch_add(1, Ordering::SeqCst) + 1;
        if since_flush >= self.flush_interval && !self.pending_flush.load(Ordering::SeqCst) {
            self.start_async_flush();
        }

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

    /// Start an async flush: send blocks to the background persistence thread.
    /// Does NOT block — the producer continues immediately.
    fn start_async_flush(&self) {
        let mut blocks: Vec<ExecutedBlock<ArbPrimitives>> = Vec::new();
        if let Some(head_state) = self.in_memory_state.head_state() {
            for block_state in head_state.chain() {
                blocks.push(block_state.block().clone());
            }
        }
        blocks.reverse();

        if blocks.is_empty() {
            return;
        }

        let last = blocks.last().unwrap();
        let last_num_hash = alloy_eips::BlockNumHash::new(
            last.recovered_block().number(),
            last.recovered_block().hash(),
        );

        // Move current accumulator to flushing slot (double-buffer).
        // New blocks will accumulate into a fresh TrieInput.
        let current = std::mem::take(&mut *self.accumulated_trie_input.lock());
        *self.flushing_trie_input.lock() = Some(current);

        self.blocks_since_flush.store(0, Ordering::SeqCst);
        self.pending_flush.store(true, Ordering::SeqCst);

        let count = blocks.len();
        crate::launcher::start_flush(crate::launcher::FlushRequest {
            blocks,
            last_num_hash,
        });

        debug!(
            target: "block_producer",
            count,
            last_block = last_num_hash.number,
            "Started async flush"
        );
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
    fn cache_init_message(&self, l2_msg: &[u8]) -> Result<(), BlockProducerError> {
        let init_msg = parse_init_message(l2_msg)
            .map_err(|e| BlockProducerError::Parse(format!("init message: {e}")))?;

        info!(
            target: "block_producer",
            chain_id = %init_msg.chain_id,
            initial_l1_base_fee = %init_msg.initial_l1_base_fee,
            "Cached Init message params"
        );

        *self.cached_init.lock() = Some(init_msg);
        Ok(())
    }

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

        // Parse L2 transactions from the message.
        let chain_id = self.chain_spec.chain().id();
        let parsed_txs = parse_l2_transactions(
            input.kind,
            input.sender,
            &input.l2_msg,
            input.request_id,
            input.l1_base_fee,
            chain_id,
        )
        .unwrap_or_else(|e| {
            // If ParseL2Transactions returns an error, treat as empty.
            warn!(target: "block_producer", error=%e, "Error parsing L2 message, treating as empty");
            vec![]
        });

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

/// Construct a mix_hash from send_count, l1_block_number, and arbos_version.
fn compute_mix_hash(send_count: u64, l1_block_number: u64, arbos_version: u64) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[0..8].copy_from_slice(&send_count.to_be_bytes());
    bytes[8..16].copy_from_slice(&l1_block_number.to_be_bytes());
    bytes[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    B256::from(bytes)
}

/// EIP-161: mark empty non-zombie accounts for trie deletion.
///
/// Accounts that existed in the trie before this block are marked as deleted
/// (info=None). Accounts that were created and emptied within this block are
/// removed from the bundle entirely (no trie operation needed).
fn delete_empty_accounts(
    bundle: &mut BundleState,
    zombie_accounts: &std::collections::HashSet<Address>,
    state_provider: &dyn StateProvider,
) {
    let keccak_empty = alloy_primitives::B256::from(alloy_primitives::keccak256([]));
    let mut to_remove = Vec::new();
    for (addr, account) in bundle.state.iter_mut() {
        if let Some(ref info) = account.info {
            let is_empty =
                info.nonce == 0 && info.balance.is_zero() && info.code_hash == keccak_empty;
            if is_empty && !zombie_accounts.contains(addr) {
                let existed_before = state_provider.basic_account(addr).ok().flatten().is_some();
                if existed_before {
                    debug!(
                        target: "block_producer",
                        addr = ?addr,
                        "EIP-161: deleting empty account from state"
                    );
                    account.info = None;
                } else {
                    to_remove.push(*addr);
                }
            }
        }
    }
    for addr in to_remove {
        bundle.state.remove(&addr);
    }
}

/// Remove unchanged storage slots from the bundle.
fn filter_unchanged_storage(bundle: &mut BundleState) {
    for (_addr, account) in bundle.state.iter_mut() {
        account
            .storage
            .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
    }
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
        state_provider.storage(addr, slot).ok().flatten()
    };

    derive_arb_header_info(&read_slot)
}

/// Augment the bundle with cache modifications not captured by transitions.
/// Diffs cache against state provider and adds missing/changed entries.
fn augment_bundle_from_cache(
    bundle: &mut BundleState,
    cache: &revm_database::CacheState,
    state_provider: &dyn StateProvider,
) {
    use revm_database::states::plain_account::StorageSlot;

    for (addr, cache_acct) in &cache.accounts {
        let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
        let current_storage = cache_acct
            .account
            .as_ref()
            .map(|a| &a.storage)
            .cloned()
            .unwrap_or_default();

        if let Some(bundle_acct) = bundle.state.get_mut(addr) {
            // Account already in bundle — update info and storage from cache
            // to capture post-commit modifications.
            bundle_acct.info = current_info;

            for (key, value) in &current_storage {
                if let Some(slot) = bundle_acct.storage.get_mut(key) {
                    slot.present_value = *value;
                } else {
                    // Storage slot written via direct cache mod, not by EVM.
                    let original_value = state_provider
                        .storage(*addr, B256::from(*key))
                        .ok()
                        .flatten()
                        .unwrap_or(U256::ZERO);
                    if *value != original_value {
                        bundle_acct.storage.insert(
                            *key,
                            StorageSlot {
                                previous_or_original_value: original_value,
                                present_value: *value,
                            },
                        );
                    }
                }
            }
        } else {
            // Account not in bundle — check if it was modified from original.
            let original = state_provider.basic_account(addr).ok().flatten();

            let info_changed = match (&original, &current_info) {
                (None, None) => false,
                (Some(_), None) | (None, Some(_)) => true,
                (Some(orig), Some(curr)) => {
                    orig.balance != curr.balance
                        || orig.nonce != curr.nonce
                        || orig
                            .bytecode_hash
                            .unwrap_or(alloy_primitives::KECCAK256_EMPTY)
                            != curr.code_hash
                }
            };

            let storage_changes: alloy_primitives::map::HashMap<U256, StorageSlot> =
                current_storage
                    .iter()
                    .filter_map(|(key, value)| {
                        let original_value = state_provider
                            .storage(*addr, B256::from(*key))
                            .ok()
                            .flatten()
                            .unwrap_or(U256::ZERO);
                        if original_value != *value {
                            Some((
                                *key,
                                StorageSlot {
                                    previous_or_original_value: original_value,
                                    present_value: *value,
                                },
                            ))
                        } else {
                            None
                        }
                    })
                    .collect();

            if info_changed || !storage_changes.is_empty() {
                // For OriginalValuesKnown::No persistence, original_info
                // is not relied upon; set to None for simplicity.
                let original_info = None;

                let status = if original.is_some() {
                    revm_database::AccountStatus::Changed
                } else {
                    revm_database::AccountStatus::InMemoryChange
                };

                bundle.state.insert(
                    *addr,
                    revm_database::BundleAccount {
                        info: current_info,
                        original_info,
                        storage: storage_changes,
                        status,
                    },
                );
            }
        }
    }
}
