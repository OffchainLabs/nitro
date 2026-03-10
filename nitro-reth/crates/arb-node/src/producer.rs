//! Block producer implementation.
//!
//! Produces blocks from L1 incoming messages by parsing transactions,
//! executing them against the current state, and persisting the results.

use std::sync::Arc;

use alloy_consensus::transaction::SignerRecoverable;
use alloy_consensus::{Block, BlockBody, BlockHeader, Header, TxReceipt, proofs, EMPTY_OMMER_ROOT_HASH};
use alloy_eips::eip2718::Decodable2718;
use alloy_evm::block::{BlockExecutor, BlockExecutorFactory};
use alloy_evm::EvmFactory;
use alloy_primitives::{Address, Bytes, B64, B256, U256, address};
use alloy_rpc_types_eth::BlockNumberOrTag;
use parking_lot::Mutex;
use reth_chainspec::ChainSpec;
use reth_evm::ConfigureEvm;
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider, StateProviderFactory};
use reth_primitives_traits::{SealedHeader, logs_bloom};
use reth_revm::database::StateProviderDatabase;
use reth_storage_api::StateProvider;
use reth_trie_common::HashedPostState;
use reth_trie_common::updates::TrieUpdates;
use revm::database::{BundleState, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use tracing::{debug, info, warn};

use arb_evm::config::{ArbEvmConfig, arbos_version_from_mix_hash};
use arb_primitives::signed_tx::ArbTransactionSigned;
use arb_primitives::tx_types::ArbInternalTx;
use arb_rpc::block_producer::{BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock};
use arbos::arbos_types::parse_init_message;
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
                HashedPostState,
                TrieUpdates,
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
        hashed_state: HashedPostState,
        trie_updates: TrieUpdates,
    ) -> Result<(), BlockProducerError> {
        (self.persist_fn)(sealed, receipts, bundle_state, hashed_state, trie_updates)
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
    /// Cached Init message params, applied during the first block's execution.
    cached_init: Mutex<Option<arbos::arbos_types::ParsedInitMessage>>,
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
                HashedPostState,
                TrieUpdates,
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
                state_provider.storage(addr, slot.into()).ok().flatten()
            };
            arbos::header::read_l2_base_fee(&read_slot)
                .or(parent_header.base_fee_per_gas())
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
                .map_err(|e| BlockProducerError::Execution(e))?;
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

        // --- Diagnostic: enable gasBacklog tracing for target block window ---
        let diag_window = (616858..=616866).contains(&l2_block_number);
        arb_storage::GASBACKLOG_TRACE.store(diag_window, std::sync::atomic::Ordering::Relaxed);
        if diag_window {
            eprintln!("[DIAG] ===== block {} =====", l2_block_number);
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
                    let report_data = if parent_arbos_version
                        >= arb_chainspec::arbos_version::ARBOS_VERSION_50
                    {
                        // V2: pass raw batch data stats + extra_gas.
                        let (length, non_zeros) =
                            input.batch_data_stats.unwrap_or((0, 0));
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
                                    if let Some(retry_tx) = retry_tx
                                    {
                                        let retry_signed = retry_tx.clone();
                                        match retry_tx.try_into_recovered() {
                                            Ok(recovered_retry) => {
                                                match executor.execute_transaction_without_commit(recovered_retry) {
                                                    Ok(retry_result) => {
                                                        match executor.commit_transaction(retry_result) {
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
        let keccak_empty_hash =
            alloy_primitives::B256::from(alloy_primitives::keccak256(&[]));
        for addr in &finalise_deleted {
            // Zombie accounts were re-created after Finalise deleted them.
            // They're back in cache and handled by augment_bundle_from_cache.
            if zombie_accounts.contains(addr) {
                continue;
            }
            if bundle.state.contains_key(addr) {
                // Account is in bundle from EVM transitions. Check whether
                // it existed in the trie before this block.
                let existed_before = state_provider.basic_account(addr)
                    .ok()
                    .flatten()
                    .is_some();
                if existed_before {
                    // Account was in the trie. Only mark as deleted if it's
                    // still empty — it may have been re-created with non-zero
                    // state (e.g., nonce=1) by a later tx in this block.
                    let still_empty = bundle.state.get(addr)
                        .and_then(|a| a.info.as_ref())
                        .map_or(true, |info| {
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
                    let still_empty = bundle.state.get(addr)
                        .and_then(|a| a.info.as_ref())
                        .map_or(true, |info| {
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
                    && acct
                        .bytecode_hash
                        .map_or(true, |h| h == keccak_empty_hash);
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

        // --- Diagnostic: check gasBacklog at each pipeline stage ---
        if diag_window {
            let arbos_addr = alloy_primitives::address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
            let bl_slot = arb_storage::GASBACKLOG_SLOT;

            // Stage 1: after merge+take+augment+finalise_deleted, BEFORE filter
            let stage1 = bundle.state.get(&arbos_addr)
                .and_then(|a| a.storage.get(&bl_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));
            eprintln!("[DIAG] block={} STAGE1(pre-filter) backlog={:?}", l2_block_number, stage1);

            // Also check: how many ArbOS storage slots are in the bundle?
            let arbos_slot_count = bundle.state.get(&arbos_addr)
                .map(|a| a.storage.len()).unwrap_or(0);
            eprintln!("[DIAG] block={} ArbOS bundle slots={}", l2_block_number, arbos_slot_count);

            // Check cache value
            let cache_val = db.cache.accounts.get(&arbos_addr)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&bl_slot).copied());
            eprintln!("[DIAG] block={} cache backlog={:?}", l2_block_number, cache_val);

            // Check DB value (should be pre-block)
            let db_val = state_provider.storage(arbos_addr, alloy_primitives::B256::from(bl_slot)).ok().flatten().unwrap_or(U256::ZERO);
            eprintln!("[DIAG] block={} db(pre-block) backlog={}", l2_block_number, db_val);
        }

        // Filter bundle to only include actually changed storage slots.
        // revm's bundle may include storage slots that were loaded (read) but
        // not modified. Including unchanged slots in the HashedPostState would
        // produce an incorrect state root.
        filter_unchanged_storage(&mut bundle);

        if diag_window {
            let arbos_addr = alloy_primitives::address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
            let bl_slot = arb_storage::GASBACKLOG_SLOT;
            let stage2 = bundle.state.get(&arbos_addr)
                .and_then(|a| a.storage.get(&bl_slot))
                .map(|s| s.present_value);
            eprintln!("[DIAG] block={} STAGE2(post-filter) backlog={:?}", l2_block_number, stage2);
        }

        // Delete empty accounts from the bundle (EIP-161).
        // Zombie accounts are preserved.
        delete_empty_accounts(&mut bundle, &zombie_accounts, &*state_provider);

        if diag_window {
            let arbos_addr = alloy_primitives::address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
            let bl_slot = arb_storage::GASBACKLOG_SLOT;
            let stage3 = bundle.state.get(&arbos_addr)
                .and_then(|a| a.storage.get(&bl_slot))
                .map(|s| s.present_value);
            let arbos_in_bundle = bundle.state.contains_key(&arbos_addr);
            eprintln!("[DIAG] block={} STAGE3(post-delete) backlog={:?} arbos_in_bundle={}", l2_block_number, stage3, arbos_in_bundle);
        }

        let hashed_state =
            HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(
                bundle.state(),
            );

        if diag_window {
            let arbos_addr = alloy_primitives::address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
            let arbos_hash = alloy_primitives::keccak256(arbos_addr);
            let has_arbos_in_hps = hashed_state.storages.contains_key(&arbos_hash);
            let arbos_hps_slots = hashed_state.storages.get(&arbos_hash)
                .map(|s| s.storage.len()).unwrap_or(0);
            eprintln!("[DIAG] block={} HashedPostState has_arbos={} arbos_slots={}", l2_block_number, has_arbos_in_hps, arbos_hps_slots);
        }

        let (state_root, trie_updates) = state_provider
            .state_root_with_updates(hashed_state.clone())
            .map_err(|e| BlockProducerError::Execution(format!("state root: {e}")))?;

        if diag_window {
            let ref_root_616862 = alloy_primitives::b256!("0bc0865ee2ad0e15c96ecde70990664a76f2a593300e29e08ed6b0d2073766b4");
            eprintln!("[DIAG] block={} computed_root={:?} expected_616862={:?} match={}",
                l2_block_number, state_root,
                if l2_block_number == 616862 { format!("{:?}", ref_root_616862) } else { "n/a".to_string() },
                l2_block_number == 616862 && state_root == ref_root_616862
            );
        }

        // At block 616862: comprehensive state comparison against reference
        if l2_block_number == 616862 {
            use reth_storage_api::StateRootProvider;
            let ref_root = alloy_primitives::b256!("0bc0865ee2ad0e15c96ecde70990664a76f2a593300e29e08ed6b0d2073766b4");
            let keccak_empty = alloy_primitives::B256::from(alloy_primitives::keccak256(&[]));
            let arbos_addr = alloy_primitives::address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");

            // --- 1. Dump all accounts in our bundle ---
            let mut addrs: Vec<_> = bundle.state.keys().cloned().collect();
            addrs.sort();
            eprintln!("[B616862] BUNDLE: {} accounts", addrs.len());
            eprintln!("[B616862] zombie_accounts: {:?}", zombie_accounts);
            eprintln!("[B616862] finalise_deleted: {:?}", finalise_deleted);
            for addr in &addrs {
                let acct = bundle.state.get(addr).unwrap();
                let info_str = match &acct.info {
                    Some(i) => format!("n={} b={} code={}",
                        i.nonce, i.balance,
                        if i.code_hash == keccak_empty { "empty".to_string() } else { format!("{:?}", i.code_hash) }),
                    None => "DELETED".to_string(),
                };
                eprintln!("[B616862]   {:?} status={:?} {} storage={}",
                    addr, acct.status, info_str, acct.storage.len());
                // Print ALL storage for ALL accounts (not just ArbOS)
                for (k, v) in &acct.storage {
                    eprintln!("[B616862]     slot {} = {} (orig={})", k, v.present_value, v.previous_or_original_value);
                }
            }

            // --- 2. Check expected accounts that SHOULD have changed (from reference) ---
            // Accounts that changed between 616861→616862 on Arbitrum Sepolia:
            let expected_changes: Vec<(Address, u64, &str)> = vec![
                // (address, expected_nonce_at_616862, description)
                (address!("fd86e9a33fd52e4085fb94d24b759448a621cd36"), 1, "Sender/Poster"),
                (address!("4453d0eaf066a61c9b81ddc18bb5a2bf2fc52224"), 2, "RetryTarget"),
                (address!("35aa95ac4747d928e2cd42fe4461f6d9d1826346"), 1, "Contract3(bal changed)"),
                (address!("d50e4a971bc8ed55af6aebc0a2178456069e87b5"), 1, "FeeRefund(bal+nonce changed)"),
            ];
            eprintln!("[B616862] --- Expected changed accounts ---");
            for (addr, expected_nonce, desc) in &expected_changes {
                let in_bundle = bundle.state.get(addr);
                match in_bundle {
                    Some(acct) => {
                        let n = acct.info.as_ref().map(|i| i.nonce).unwrap_or(u64::MAX);
                        let b = acct.info.as_ref().map(|i| i.balance).unwrap_or(U256::MAX);
                        let nonce_ok = n == *expected_nonce;
                        eprintln!("[B616862] {} {:?}: IN BUNDLE n={} b={} nonce_match={}",
                            desc, addr, n, b, nonce_ok);
                    }
                    None => {
                        eprintln!("[B616862] {} {:?}: MISSING FROM BUNDLE!", desc, addr);
                    }
                }
            }

            // --- 3. Check escrow account (should NOT exist in trie at 616862) ---
            let escrow = address!("fb3504a7e996cb0a8dcdc95d58b9dafaa51249b4");
            let escrow_in_bundle = bundle.state.get(&escrow);
            let escrow_in_db = state_provider.basic_account(&escrow).ok().flatten();
            eprintln!("[B616862] Escrow {:?}: in_bundle={:?} in_db={:?}",
                escrow,
                escrow_in_bundle.map(|a| format!("info={:?} status={:?} storage={}",
                    a.info.as_ref().map(|i| format!("n={} b={}", i.nonce, i.balance)),
                    a.status, a.storage.len())),
                escrow_in_db.map(|a| format!("n={} b={}", a.nonce, a.balance)));

            // --- 4. Binary search: try removing each account ---
            eprintln!("[B616862] Binary search (remove each account)...");
            for addr in &addrs {
                let mut test = bundle.clone();
                test.state.remove(addr);
                let hps = HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(test.state());
                if let Ok((root, _)) = state_provider.state_root_with_updates(hps) {
                    if root == ref_root {
                        eprintln!("[B616862] >>> REMOVING {:?} FIXES ROOT <<<", addr);
                    }
                }
            }

            // --- 5. Binary search: try removing each ArbOS storage slot ---
            if let Some(arbos_acct) = bundle.state.get(&arbos_addr) {
                for (slot_key, slot_val) in &arbos_acct.storage {
                    let mut test = bundle.clone();
                    if let Some(a) = test.state.get_mut(&arbos_addr) {
                        a.storage.remove(slot_key);
                    }
                    let hps = HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(test.state());
                    if let Ok((root, _)) = state_provider.state_root_with_updates(hps) {
                        if root == ref_root {
                            eprintln!("[B616862] >>> REMOVING ArbOS slot {} FIXES ROOT <<<", slot_key);
                        }
                    }
                }
            }

            // --- 6. Try adding missing accounts (accounts that changed on ref but aren't in bundle) ---
            eprintln!("[B616862] Testing missing accounts...");
            for (addr, expected_nonce, desc) in &expected_changes {
                if !bundle.state.contains_key(addr) {
                    // Account is missing - try adding it with expected state
                    let db_info = state_provider.basic_account(addr).ok().flatten();
                    eprintln!("[B616862] {} {:?} missing, db_info={:?}", desc, addr, db_info);
                    // Try adding with the expected nonce
                    let mut test = bundle.clone();
                    test.state.insert(*addr, revm_database::BundleAccount {
                        info: Some(revm::state::AccountInfo {
                            nonce: *expected_nonce,
                            balance: U256::ZERO,
                            code_hash: keccak_empty,
                            code: None,
                            account_id: None,
                        }),
                        original_info: None,
                        storage: Default::default(),
                        status: revm_database::AccountStatus::Changed,
                    });
                    let hps = HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(test.state());
                    if let Ok((root, _)) = state_provider.state_root_with_updates(hps) {
                        if root == ref_root {
                            eprintln!("[B616862] >>> ADDING {:?} WITH n={} FIXES ROOT <<<", addr, expected_nonce);
                        }
                    }
                }
            }

            // --- 7. For each account, try setting its nonce to the reference value ---
            eprintln!("[B616862] Testing nonce corrections...");
            for (addr, expected_nonce, desc) in &expected_changes {
                if let Some(acct) = bundle.state.get(addr) {
                    if let Some(ref info) = acct.info {
                        if info.nonce != *expected_nonce {
                            let mut test = bundle.clone();
                            if let Some(a) = test.state.get_mut(addr) {
                                if let Some(ref mut i) = a.info {
                                    i.nonce = *expected_nonce;
                                }
                            }
                            let hps = HashedPostState::from_bundle_state::<reth_trie_common::KeccakKeyHasher>(test.state());
                            if let Ok((root, _)) = state_provider.state_root_with_updates(hps) {
                                if root == ref_root {
                                    eprintln!("[B616862] >>> FIXING {} nonce to {} FIXES ROOT <<<", desc, expected_nonce);
                                }
                            }
                        }
                    }
                }
            }

            // --- 8. If nothing above fixed it, try empty bundle (no changes at all) ---
            {
                let empty_hps = HashedPostState::default();
                if let Ok((root, _)) = state_provider.state_root_with_updates(empty_hps) {
                    eprintln!("[B616862] Empty bundle (no changes) root: {:?} match={}", root, root == ref_root);
                }
            }

            eprintln!("[B616862] === Diagnostics complete ===");
        }

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
            base_fee_per_gas: l2_base_fee,
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

        // Persist block, receipts, state changes, hashed state, and trie updates.
        self.persister.persist(&sealed, receipts, bundle, hashed_state, trie_updates)?;

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
    let keccak_empty = alloy_primitives::B256::from(alloy_primitives::keccak256(&[]));
    let mut to_remove = Vec::new();
    for (addr, account) in bundle.state.iter_mut() {
        if let Some(ref info) = account.info {
            let is_empty = info.nonce == 0
                && info.balance.is_zero()
                && info.code_hash == keccak_empty;
            if is_empty && !zombie_accounts.contains(addr) {
                let existed_before = state_provider.basic_account(addr)
                    .ok()
                    .flatten()
                    .is_some();
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
        state_provider.storage(addr, slot.into()).ok().flatten()
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
                        || orig.bytecode_hash.unwrap_or(alloy_primitives::KECCAK256_EMPTY)
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

