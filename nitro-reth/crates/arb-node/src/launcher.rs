//! Custom engine node launcher for Arbitrum.
//!
//! Extends reth's standard `EngineNodeLauncher` by capturing the engine tree
//! sender during orchestrator construction. This sender allows the block
//! producer to inject `InsertExecutedBlock` directly into reth's engine tree
//! for persistence via `PersistenceService::save_blocks(Full)`.
//!
//! This is the reth SDK-native approach: implement `LaunchNode` with custom
//! orchestrator wiring while reusing all other engine infrastructure.

use crate::engine::{build_arb_engine_orchestrator, TreeSender};
use alloy_consensus::BlockHeader;
use futures::{stream::FusedStream, stream_select, FutureExt, StreamExt};
use reth_chainspec::{EthChainSpec, EthereumHardforks};
use reth_engine_tree::{
    chain::{ChainEvent, FromOrchestrator},
    engine::{EngineApiKind, EngineApiRequest, EngineRequestHandler},
    tree::TreeConfig,
};
use reth_engine_util::EngineMessageStreamExt;
use reth_exex::ExExManagerHandle;
use reth_network::{types::BlockRangeUpdate, NetworkSyncUpdater, SyncState};
use reth_network_api::BlockDownloaderProvider;
use reth_node_api::{
    BuiltPayload, ConsensusEngineHandle, FullNodeTypes, NodeTypes, NodeTypesWithDBAdapter,
};
use reth_node_builder::{
    common::{Attached, LaunchContextWith, WithConfigs},
    hooks::NodeHooks,
    rpc::{EngineShutdown, EngineValidatorAddOn, EngineValidatorBuilder, RethRpcAddOns, RpcHandle},
    setup::build_networked_pipeline,
    AddOns, AddOnsContext, FullNode, LaunchContext, LaunchNode, NodeAdapter,
    NodeBuilderWithComponents, NodeComponents, NodeComponentsBuilder, NodeHandle, NodeTypesAdapter,
};
use reth_node_core::{
    dirs::{ChainPath, DataDirPath},
    exit::NodeExitFuture,
    primitives::Head,
};
use reth_node_events::node;
use reth_provider::{
    providers::{BlockchainProvider, NodeTypesForProvider},
    BlockNumReader, StorageSettingsCache,
};
use reth_tasks::TaskExecutor;
use reth_tokio_util::EventSender;
use reth_tracing::tracing::{debug, error, info};
use reth_trie_db::ChangesetCache;
use std::{
    future::Future,
    pin::Pin,
    sync::{Arc, OnceLock},
};
use tokio::sync::{mpsc::unbounded_channel, oneshot};
use tokio_stream::wrappers::UnboundedReceiverStream;

use arb_payload::ArbEngineTypes;
use arb_primitives::ArbPrimitives;

static TREE_SENDER: OnceLock<TreeSender<ArbEngineTypes, ArbPrimitives>> = OnceLock::new();
static ENGINE_HANDLE: OnceLock<ConsensusEngineHandle<ArbEngineTypes>> = OnceLock::new();
static FLUSH_HANDLE: OnceLock<FlushHandle> = OnceLock::new();
static PARALLEL_STATE_ROOT_FN: OnceLock<ParallelStateRootFn> = OnceLock::new();

/// Request sent to the background persistence thread.
pub struct FlushRequest {
    pub blocks: Vec<reth_chain_state::ExecutedBlock<ArbPrimitives>>,
    pub last_num_hash: alloy_eips::BlockNumHash,
}

/// Result from a completed flush.
pub struct FlushResult {
    pub last_num_hash: alloy_eips::BlockNumHash,
    pub count: usize,
    pub duration: std::time::Duration,
}

/// Handle to the background persistence thread.
struct FlushHandle {
    sender: std::sync::mpsc::Sender<FlushRequest>,
    result_rx: crossbeam_channel::Receiver<FlushResult>,
}

/// Type-erased parallel state root function.
type ParallelStateRootFn = Box<
    dyn Fn(
            reth_trie_common::TrieInput,
        ) -> Result<(alloy_primitives::B256, reth_trie::updates::TrieUpdates), String>
        + Send
        + Sync,
>;

pub fn tree_sender() -> Option<&'static TreeSender<ArbEngineTypes, ArbPrimitives>> {
    TREE_SENDER.get()
}

pub fn engine_handle() -> Option<&'static ConsensusEngineHandle<ArbEngineTypes>> {
    ENGINE_HANDLE.get()
}

/// Send blocks to the background persistence thread (non-blocking).
pub fn start_flush(request: FlushRequest) {
    if let Some(handle) = FLUSH_HANDLE.get() {
        if let Err(e) = handle.sender.send(request) {
            error!(target: "reth::cli", "Failed to send flush request: {e}");
        }
    }
}

/// Check if a background flush completed (non-blocking).
pub fn try_flush_result() -> Option<FlushResult> {
    FLUSH_HANDLE
        .get()
        .and_then(|handle| handle.result_rx.try_recv().ok())
}

/// Compute state root with parallel storage roots and cached trie node overlay.
pub fn compute_parallel_state_root(
    input: reth_trie_common::TrieInput,
) -> Result<(alloy_primitives::B256, reth_trie::updates::TrieUpdates), String> {
    let f = PARALLEL_STATE_ROOT_FN
        .get()
        .ok_or_else(|| "parallel_state_root not initialized".to_string())?;
    f(input)
}

/// Arbitrum engine node launcher.
///
/// Identical to reth's `EngineNodeLauncher` but captures the engine tree sender
/// during orchestrator construction for block injection.
#[derive(Debug)]
pub struct ArbEngineLauncher {
    pub ctx: LaunchContext,
    pub engine_tree_config: TreeConfig,
}

impl ArbEngineLauncher {
    pub const fn new(
        task_executor: TaskExecutor,
        data_dir: ChainPath<DataDirPath>,
        engine_tree_config: TreeConfig,
    ) -> Self {
        Self {
            ctx: LaunchContext::new(task_executor, data_dir),
            engine_tree_config,
        }
    }

    /// Launch the node — mirrors EngineNodeLauncher::launch_node exactly,
    /// except uses build_arb_engine_orchestrator to capture the tree sender.
    async fn launch_node<T, CB, AO>(
        self,
        target: NodeBuilderWithComponents<T, CB, AO>,
    ) -> eyre::Result<NodeHandle<NodeAdapter<T, CB::Components>, AO>>
    where
        T: FullNodeTypes<
            Types: NodeTypesForProvider<Payload = ArbEngineTypes, Primitives = ArbPrimitives>,
            Provider = BlockchainProvider<
                NodeTypesWithDBAdapter<<T as FullNodeTypes>::Types, <T as FullNodeTypes>::DB>,
            >,
        >,
        CB: NodeComponentsBuilder<T>,
        AO: RethRpcAddOns<NodeAdapter<T, CB::Components>>
            + EngineValidatorAddOn<NodeAdapter<T, CB::Components>>,
    {
        let Self {
            ctx,
            engine_tree_config,
        } = self;
        let NodeBuilderWithComponents {
            adapter: NodeTypesAdapter { database },
            components_builder,
            add_ons:
                AddOns {
                    hooks,
                    exexs: installed_exex,
                    add_ons,
                },
            config,
        } = target;
        let NodeHooks {
            on_component_initialized,
            on_node_started,
            ..
        } = hooks;

        let changeset_cache = ChangesetCache::new();

        let ctx = ctx
            .with_configured_globals(engine_tree_config.reserved_cpu_cores())
            .with_loaded_toml_config(config)?
            .with_resolved_peers()?
            .attach(database.clone())
            .with_adjusted_configs()
            .with_provider_factory::<_, <CB::Components as NodeComponents<T>>::Evm>(
                changeset_cache.clone(),
            )
            .await?
            .inspect(|_| {
                info!(target: "reth::cli", "Database opened");
            })
            .with_prometheus_server()
            .await?
            .inspect(|this| {
                debug!(target: "reth::cli", chain=%this.chain_id(), genesis=?this.genesis_hash(), "Initializing genesis");
            })
            .with_genesis()?
            .inspect(
                |this: &LaunchContextWith<
                    Attached<WithConfigs<<T::Types as NodeTypes>::ChainSpec>, _>,
                >| {
                    info!(target: "reth::cli", "\n{}", this.chain_spec().display_hardforks());
                    let settings = this.provider_factory().cached_storage_settings();
                    info!(target: "reth::cli", ?settings, "Loaded storage settings");
                },
            )
            .with_metrics_task()
            .with_blockchain_db::<T, _>(move |provider_factory| {
                Ok(BlockchainProvider::new(provider_factory)?)
            })?
            .with_components(components_builder, on_component_initialized)
            .await?;

        let maybe_exex_manager_handle = ctx.launch_exex(installed_exex).await?;

        let network_handle = ctx.components().network().clone();
        let network_client = network_handle.fetch_client().await?;
        let (consensus_engine_tx, consensus_engine_rx) = unbounded_channel();

        let node_config = ctx.node_config();

        network_handle.update_sync_state(SyncState::Syncing);

        let max_block = ctx.max_block(network_client.clone()).await?;

        let static_file_producer = ctx.static_file_producer();
        let static_file_producer_events = static_file_producer.lock().events();
        info!(target: "reth::cli", "StaticFileProducer initialized");

        let consensus = Arc::new(ctx.components().consensus().clone());

        let pipeline = build_networked_pipeline(
            &ctx.toml_config().stages,
            network_client.clone(),
            consensus.clone(),
            ctx.provider_factory().clone(),
            ctx.task_executor(),
            ctx.sync_metrics_tx(),
            ctx.prune_config(),
            max_block,
            static_file_producer,
            ctx.components().evm_config().clone(),
            maybe_exex_manager_handle
                .clone()
                .unwrap_or_else(ExExManagerHandle::empty),
            ctx.era_import_source(),
        )?;

        pipeline.move_to_static_files()?;

        let pipeline_events = pipeline.events();

        let mut pruner_builder = ctx.pruner_builder();
        if let Some(exex_manager_handle) = &maybe_exex_manager_handle {
            pruner_builder =
                pruner_builder.finished_exex_height(exex_manager_handle.finished_height());
        }
        let pruner = pruner_builder.build_with_provider_factory(ctx.provider_factory().clone());
        let pruner_events = pruner.events();
        info!(target: "reth::cli", prune_config=?ctx.prune_config(), "Pruner initialized");

        let event_sender = EventSender::default();

        let beacon_engine_handle = ConsensusEngineHandle::new(consensus_engine_tx.clone());

        let jwt_secret = ctx.auth_jwt_secret()?;

        let add_ons_ctx = AddOnsContext {
            node: ctx.node_adapter().clone(),
            config: ctx.node_config(),
            beacon_engine_handle: beacon_engine_handle.clone(),
            jwt_secret,
            engine_events: event_sender.clone(),
        };
        let validator_builder = add_ons.engine_validator_builder();

        let engine_validator = validator_builder
            .clone()
            .build_tree_validator(
                &add_ons_ctx,
                engine_tree_config.clone(),
                changeset_cache.clone(),
            )
            .await?;

        let consensus_engine_stream = UnboundedReceiverStream::from(consensus_engine_rx)
            .maybe_skip_fcu(node_config.debug.skip_fcu)
            .maybe_skip_new_payload(node_config.debug.skip_new_payload)
            .maybe_reorg(
                ctx.blockchain_db().clone(),
                ctx.components().evm_config().clone(),
                || async {
                    let reorg_cache = ChangesetCache::new();
                    validator_builder
                        .build_tree_validator(&add_ons_ctx, engine_tree_config.clone(), reorg_cache)
                        .await
                },
                node_config.debug.reorg_frequency,
                node_config.debug.reorg_depth,
            )
            .await?
            .maybe_store_messages(node_config.debug.engine_api_store.clone());

        let engine_kind = if ctx.chain_spec().is_optimism() {
            EngineApiKind::OpStack
        } else {
            EngineApiKind::Ethereum
        };

        // Spawn background persistence thread (like reth's PersistenceHandle).
        // Receives flush requests via channel, calls save_blocks(Full) + commit(),
        // sends result back. The producer continues producing while flush runs.
        {
            use reth_provider::{DatabaseProviderFactory, SaveBlocksMode};
            use reth_storage_api::DBProvider;

            let pf = ctx.provider_factory().clone();
            let (req_tx, req_rx) = std::sync::mpsc::channel::<FlushRequest>();
            let (res_tx, res_rx) = crossbeam_channel::bounded::<FlushResult>(1);

            std::thread::Builder::new()
                .name("arb-persistence".into())
                .spawn(move || {
                    while let Ok(req) = req_rx.recv() {
                        let start = std::time::Instant::now();
                        let count = req.blocks.len();
                        let last = req.last_num_hash;

                        let result = (|| -> Result<(), String> {
                            let provider_rw = pf
                                .database_provider_rw()
                                .map_err(|e| format!("database_provider_rw: {e}"))?;
                            provider_rw
                                .save_blocks(req.blocks, SaveBlocksMode::Full)
                                .map_err(|e| format!("save_blocks: {e}"))?;
                            provider_rw
                                .commit()
                                .map_err(|e| format!("commit: {e}"))?;
                            Ok(())
                        })();

                        match result {
                            Ok(()) => {
                                let _ = res_tx.send(FlushResult {
                                    last_num_hash: last,
                                    count,
                                    duration: start.elapsed(),
                                });
                            }
                            Err(e) => {
                                error!(target: "reth::cli", "Background flush failed: {e}");
                                // Send result anyway so producer can detect and recover
                                let _ = res_tx.send(FlushResult {
                                    last_num_hash: last,
                                    count: 0, // signal failure
                                    duration: start.elapsed(),
                                });
                            }
                        }
                    }
                })
                .expect("failed to spawn persistence thread");

            let _ = FLUSH_HANDLE.set(FlushHandle {
                sender: req_tx,
                result_rx: res_rx,
            });
        }

        // Create the parallel state root closure.
        // Accepts a TrieInput with cached trie nodes + accumulated state.
        // Computes storage roots in parallel (each with its own DB tx + shared overlay),
        // then walks the account trie serially with the overlay.
        {
            use alloy_primitives::B256;
            use alloy_rlp::{BufMut, Encodable};
            use reth_provider::DatabaseProviderFactory;
            use reth_trie::{
                hashed_cursor::{HashedCursorFactory, HashedPostStateCursorFactory},
                node_iter::{TrieElement, TrieNodeIter},
                trie_cursor::{InMemoryTrieCursorFactory, TrieCursorFactory},
                updates::TrieUpdates,
                walker::TrieWalker,
                HashBuilder, Nibbles, StorageRoot, TRIE_ACCOUNT_RLP_MAX_SIZE,
            };
            use reth_trie_common::TrieInputSorted;
            use reth_trie_db::{
                DatabaseTrieCursorFactory, DatabaseHashedCursorFactory,
                PackedKeyAdapter, LegacyKeyAdapter,
            };
            use reth_provider::StorageSettingsCache;
            use reth_storage_api::DBProvider;
            use reth_trie_parallel::StorageRootTargets;

            let pf = ctx.provider_factory().clone();
            let is_v2 = pf.cached_storage_settings().is_v2();

            // Helper: compute parallel state root with a specific TrieKeyAdapter.
            fn parallel_root<PF, A>(
                pf: &PF,
                sorted: TrieInputSorted,
            ) -> Result<(B256, TrieUpdates), String>
            where
                PF: DatabaseProviderFactory + Clone + Send + Sync + 'static,
                PF::Provider: reth_storage_api::DBProvider,
                A: reth_trie_db::TrieTableAdapter + Send + Sync + 'static,
            {
                let nodes = sorted.nodes.clone();
                let state = sorted.state.clone();
                let prefix_sets_frozen = sorted.prefix_sets.freeze();

                let storage_root_targets = StorageRootTargets::new(
                    prefix_sets_frozen
                        .account_prefix_set
                        .iter()
                        .map(|nibbles| B256::from_slice(&nibbles.pack())),
                    prefix_sets_frozen.storage_prefix_sets.clone(),
                );

                let mut storage_roots =
                    std::collections::HashMap::with_capacity(storage_root_targets.len());
                let mut targets_vec: Vec<_> = storage_root_targets.into_iter().collect();
                targets_vec.sort_unstable_by_key(|(addr, _)| *addr);
                for (hashed_address, prefix_set) in targets_vec {
                    let pf2 = pf.clone();
                    let nodes2 = nodes.clone();
                    let state2 = state.clone();

                    let (tx, rx) = std::sync::mpsc::sync_channel(1);
                    std::thread::spawn(move || {
                        let result = (|| -> Result<_, String> {
                            let provider_ro = pf2
                                .database_provider_ro()
                                .map_err(|e| format!("db_provider_ro: {e}"))?;
                            let db_tx = provider_ro.tx_ref();
                            let trie_cursor = InMemoryTrieCursorFactory::new(
                                DatabaseTrieCursorFactory::<_, A>::new(db_tx),
                                nodes2.as_ref(),
                            );
                            let hashed_cursor = HashedPostStateCursorFactory::new(
                                DatabaseHashedCursorFactory::new(db_tx),
                                state2.as_ref(),
                            );
                            StorageRoot::new_hashed(
                                trie_cursor, hashed_cursor, hashed_address, prefix_set,
                                Default::default(),
                            )
                            .calculate(true)
                            .map_err(|e| format!("storage root: {e}"))
                        })();
                        let _ = tx.send(result);
                    });
                    storage_roots.insert(hashed_address, rx);
                }

                let provider_ro = pf
                    .database_provider_ro()
                    .map_err(|e| format!("db_provider_ro: {e}"))?;
                let db_tx = provider_ro.tx_ref();
                let trie_cursor = InMemoryTrieCursorFactory::new(
                    DatabaseTrieCursorFactory::<_, A>::new(db_tx),
                    nodes.as_ref(),
                );
                let hashed_cursor = HashedPostStateCursorFactory::new(
                    DatabaseHashedCursorFactory::new(db_tx),
                    state.as_ref(),
                );

                let walker = TrieWalker::<_>::state_trie(
                    trie_cursor.account_trie_cursor().map_err(|e| format!("trie cursor: {e}"))?,
                    prefix_sets_frozen.account_prefix_set,
                )
                .with_deletions_retained(true);
                let mut account_node_iter = TrieNodeIter::state_trie(
                    walker,
                    hashed_cursor.hashed_account_cursor().map_err(|e| format!("hashed cursor: {e}"))?,
                );

                let mut hash_builder = HashBuilder::default().with_updates(true);
                let mut trie_updates = TrieUpdates::default();
                let mut account_rlp = Vec::with_capacity(TRIE_ACCOUNT_RLP_MAX_SIZE);

                while let Some(node) =
                    account_node_iter.try_next().map_err(|e| format!("node iter: {e}"))?
                {
                    match node {
                        TrieElement::Branch(node) => {
                            hash_builder.add_branch(node.key, node.value, node.children_are_in_trie);
                        }
                        TrieElement::Leaf(hashed_address, account) => {
                            let storage_root_result = match storage_roots.remove(&hashed_address) {
                                Some(rx) => rx
                                    .recv()
                                    .map_err(|_| format!("channel closed for {hashed_address}"))??,
                                None => {
                                    StorageRoot::new_hashed(
                                        trie_cursor.clone(),
                                        hashed_cursor.clone(),
                                        hashed_address,
                                        Default::default(),
                                        Default::default(),
                                    )
                                    .calculate(true)
                                    .map_err(|e| format!("storage root: {e}"))?
                                }
                            };

                            let (storage_root, _, updates) = match storage_root_result {
                                reth_trie::StorageRootProgress::Complete(root, _, updates) => {
                                    (root, (), updates)
                                }
                                reth_trie::StorageRootProgress::Progress(..) => {
                                    return Err("StorageRoot returned Progress".to_string())
                                }
                            };

                            trie_updates.insert_storage_updates(hashed_address, updates);

                            account_rlp.clear();
                            let account: reth_primitives_traits::Account = account;
                            let trie_account = account.into_trie_account(storage_root);
                            trie_account.encode(&mut account_rlp as &mut dyn BufMut);
                            hash_builder.add_leaf(Nibbles::unpack(hashed_address), &account_rlp);
                        }
                    }
                }

                let root = hash_builder.root();
                let removed_keys = account_node_iter.walker.take_removed_keys();
                trie_updates.finalize(
                    hash_builder,
                    removed_keys,
                    prefix_sets_frozen.destroyed_accounts,
                );

                Ok((root, trie_updates))
            }

            let state_root_fn: ParallelStateRootFn = Box::new(move |input| {
                let sorted = TrieInputSorted::from_unsorted(input);
                if is_v2 {
                    parallel_root::<_, PackedKeyAdapter>(&pf, sorted)
                } else {
                    parallel_root::<_, LegacyKeyAdapter>(&pf, sorted)
                }
            });
            let _ = PARALLEL_STATE_ROOT_FN.set(state_root_fn);
        }

        let (mut orchestrator, arb_tree_sender) = build_arb_engine_orchestrator(
            engine_kind,
            consensus.clone(),
            network_client.clone(),
            Box::pin(consensus_engine_stream),
            pipeline,
            ctx.task_executor().clone(),
            ctx.provider_factory().clone(),
            ctx.blockchain_db().clone(),
            pruner,
            ctx.components().payload_builder_handle().clone(),
            engine_validator,
            engine_tree_config,
            ctx.sync_metrics_tx(),
            ctx.components().evm_config().clone(),
            changeset_cache,
        );

        let _ = TREE_SENDER.set(arb_tree_sender);
        let _ = ENGINE_HANDLE.set(beacon_engine_handle.clone());
        info!(target: "reth::cli", "Arbitrum engine tree sender and handle captured");

        info!(target: "reth::cli", "Consensus engine initialized");

        #[allow(clippy::needless_continue)]
        let events = stream_select!(
            event_sender.new_listener().map(Into::into),
            pipeline_events.map(Into::into),
            ctx.consensus_layer_events(),
            pruner_events.map(Into::into),
            static_file_producer_events.map(Into::into),
        );

        ctx.task_executor().spawn_critical_task(
            "events task",
            Box::pin(node::handle_events(
                Some(Box::new(ctx.components().network().clone())),
                Some(ctx.head().number),
                events,
            )),
        );

        let RpcHandle {
            rpc_server_handles,
            rpc_registry,
            engine_events,
            beacon_engine_handle,
            engine_shutdown: _,
        } = add_ons.launch_add_ons(add_ons_ctx).await?;

        let (engine_shutdown, shutdown_rx) = EngineShutdown::new();

        let initial_target = ctx.initial_backfill_target()?;
        let mut built_payloads = ctx
            .components()
            .payload_builder_handle()
            .subscribe()
            .await
            .map_err(|e| eyre::eyre!("Failed to subscribe to payload builder events: {:?}", e))?
            .into_built_payload_stream()
            .fuse();

        let chainspec = ctx.chain_spec();
        let provider = ctx.blockchain_db().clone();
        let (exit, rx) = oneshot::channel();
        let terminate_after_backfill = ctx.terminate_after_initial_backfill();
        let startup_sync_state_idle = ctx.node_config().debug.startup_sync_state_idle;

        info!(target: "reth::cli", "Starting consensus engine");
        let consensus_engine = async move {
            if let Some(initial_target) = initial_target {
                debug!(target: "reth::cli", %initial_target, "start backfill sync");
                orchestrator.start_backfill_sync(initial_target);
            } else if startup_sync_state_idle {
                network_handle.update_sync_state(SyncState::Idle);
            }

            let mut res = Ok(());
            let mut shutdown_rx = shutdown_rx.fuse();

            loop {
                tokio::select! {
                    event = orchestrator.next() => {
                        let Some(event) = event else { break };
                        debug!(target: "reth::cli", "Event: {event}");
                        match event {
                            ChainEvent::BackfillSyncFinished => {
                                if terminate_after_backfill {
                                    debug!(target: "reth::cli", "Terminating after initial backfill");
                                    break
                                }
                                if startup_sync_state_idle {
                                    network_handle.update_sync_state(SyncState::Idle);
                                }
                            }
                            ChainEvent::BackfillSyncStarted => {
                                network_handle.update_sync_state(SyncState::Syncing);
                            }
                            ChainEvent::FatalError => {
                                error!(target: "reth::cli", "Fatal error in consensus engine");
                                res = Err(eyre::eyre!("Fatal error in consensus engine"));
                                break
                            }
                            ChainEvent::Handler(ev) => {
                                if let Some(head) = ev.canonical_header() {
                                    network_handle.update_sync_state(SyncState::Idle);
                                    let head_block = Head {
                                        number: head.number(),
                                        hash: head.hash(),
                                        difficulty: head.difficulty(),
                                        timestamp: head.timestamp(),
                                        total_difficulty: chainspec.final_paris_total_difficulty()
                                            .filter(|_| chainspec.is_paris_active_at_block(head.number()))
                                            .unwrap_or_default(),
                                    };
                                    network_handle.update_status(head_block);

                                    let updated = BlockRangeUpdate {
                                        earliest: provider.earliest_block_number().unwrap_or_default(),
                                        latest: head.number(),
                                        latest_hash: head.hash(),
                                    };
                                    network_handle.update_block_range(updated);
                                }
                                event_sender.notify(ev);
                            }
                        }
                    }
                    payload = built_payloads.select_next_some(), if !built_payloads.is_terminated() => {
                        if let Some(executed_block) = payload.executed_block() {
                            debug!(target: "reth::cli", block=?executed_block.recovered_block.num_hash(), "inserting built payload");
                            orchestrator.handler_mut().handler_mut().on_event(EngineApiRequest::InsertExecutedBlock(executed_block.into_executed_payload()).into());
                        }
                    }
                    shutdown_req = &mut shutdown_rx => {
                        if let Ok(req) = shutdown_req {
                            debug!(target: "reth::cli", "received engine shutdown request");
                            orchestrator.handler_mut().handler_mut().on_event(
                                FromOrchestrator::Terminate { tx: req.done_tx }.into()
                            );
                        }
                    }
                }
            }

            let _ = exit.send(res);
        };
        ctx.task_executor()
            .spawn_critical_task("consensus engine", Box::pin(consensus_engine));

        let engine_events_for_ethstats = engine_events.new_listener();

        let full_node = FullNode {
            evm_config: ctx.components().evm_config().clone(),
            pool: ctx.components().pool().clone(),
            network: ctx.components().network().clone(),
            provider: ctx.node_adapter().provider.clone(),
            payload_builder_handle: ctx.components().payload_builder_handle().clone(),
            task_executor: ctx.task_executor().clone(),
            config: ctx.node_config().clone(),
            data_dir: ctx.data_dir().clone(),
            add_ons_handle: RpcHandle {
                rpc_server_handles,
                rpc_registry,
                engine_events,
                beacon_engine_handle,
                engine_shutdown,
            },
        };
        on_node_started.on_event(FullNode::clone(&full_node))?;

        ctx.spawn_ethstats(engine_events_for_ethstats).await?;

        let handle = NodeHandle {
            node_exit_future: NodeExitFuture::new(
                async { rx.await? },
                full_node.config.debug.terminate,
            ),
            node: full_node,
        };

        Ok(handle)
    }
}

impl<T, CB, AO> LaunchNode<NodeBuilderWithComponents<T, CB, AO>> for ArbEngineLauncher
where
    T: FullNodeTypes<
        Types: NodeTypesForProvider<Payload = ArbEngineTypes, Primitives = ArbPrimitives>,
        Provider = BlockchainProvider<
            NodeTypesWithDBAdapter<<T as FullNodeTypes>::Types, <T as FullNodeTypes>::DB>,
        >,
    >,
    CB: NodeComponentsBuilder<T> + 'static,
    AO: RethRpcAddOns<NodeAdapter<T, CB::Components>>
        + EngineValidatorAddOn<NodeAdapter<T, CB::Components>>
        + 'static,
{
    type Node = NodeHandle<NodeAdapter<T, CB::Components>, AO>;
    type Future = Pin<Box<dyn Future<Output = eyre::Result<Self::Node>> + Send>>;

    fn launch_node(self, target: NodeBuilderWithComponents<T, CB, AO>) -> Self::Future {
        Box::pin(self.launch_node(target))
    }
}
