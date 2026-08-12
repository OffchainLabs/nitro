//! Build and run the MEL node
//!
//! Port of the MEL components in `arbnode`.

use std::sync::Arc;

use alloy_eips::BlockNumberOrTag;
use alloy_provider::{Provider, RootProvider};
use arb_consensus_db::{ConsensusDb, kv::LibmdbxKvStore};
use arb_mel_runner::{DaReaderRegistry, RollupAddresses};
use arb_mel_types::MelState;
use arb_parent_chain_client::{ParentChainReader, RpcParentChainReader};
use tracing::info;

use crate::{config::MelNodeConfig, consumer::LogConsumer, database::MelDatabase};

type MelDb = arb_mel_db::MelDb<LibmdbxKvStore>;
type MessageExtractor = arb_mel_runner::MessageExtractor<
    RpcParentChainReader,
    MelDatabase,
    LogConsumer,
    DaReaderRegistry,
>;

pub struct MelNode {
    extractor: MessageExtractor,
}

impl MelNode {
    pub async fn build(config: &MelNodeConfig) -> eyre::Result<Self> {
        let (parent_chain_reader, parent_chain_id) = {
            let provider = RootProvider::connect(&config.parent_chain_url).await?;
            let chain_id = provider.get_chain_id().await?;
            let reader = Arc::new(RpcParentChainReader::from(provider));
            (reader, chain_id)
        };

        let mut mel_db = {
            let store = LibmdbxKvStore::open(&config.datadir)?;
            let consensus_db = ConsensusDb::open(store)?;
            MelDb::open(consensus_db)?
        };
        // Seed a fresh database with the initial MEL state (nitro's
        // validateAndInitializeDBForMEL + createInitialMELState).
        if mel_db.head_state_block_num()?.is_none() {
            let deploy_block = config
                .deployed_at
                .checked_sub(1)
                .ok_or_else(|| eyre::eyre!("--deployed-at must be greater than 0"))?;
            let start_block = parent_chain_reader
                .header_by_number(BlockNumberOrTag::Number(deploy_block))
                .await?
                .ok_or_else(|| eyre::eyre!("parent chain has no block {deploy_block}"))?;
            let initial_state = MelState {
                version: 0,
                parent_chain_id,
                parent_chain_block_number: start_block.number,
                batch_posting_target_address: config.sequencer_inbox,
                delayed_message_posting_target_address: config.bridge,
                parent_chain_block_hash: start_block.hash,
                parent_chain_previous_block_hash: start_block.parent_hash,
                ..Default::default()
            };
            mel_db.save_state(&initial_state)?;
        }
        let db = Arc::new(MelDatabase::new(mel_db));

        let da_providers = Arc::new(DaReaderRegistry::new());
        let msg_consumer = Arc::new(LogConsumer);

        // Assemble the extractor (the tail of nitro's getMessageExtractor).
        let extractor = MessageExtractor::new(
            config.extraction_config(),
            parent_chain_reader,
            RollupAddresses {
                bridge: config.bridge,
                inbox: config.inbox,
                sequencer_inbox: config.sequencer_inbox,
                rollup: config.rollup,
            },
            db,
            msg_consumer,
            da_providers,
        );
        info!("Message extractor enabled");
        Ok(Self { extractor })
    }

    /// Drives the extractor FSM until ctrl-c. Dropping the extractor future also
    /// aborts its safe/finalized tip-updater task; mdbx is transactional, so
    /// stopping mid-tick cannot corrupt the database.
    pub async fn run(self) -> eyre::Result<()> {
        tokio::select! {
            () = self.extractor.run() => unreachable!("extractor loop never returns"),
            result = tokio::signal::ctrl_c() => {
                result?;
                info!("received ctrl-c, shutting down");
                Ok(())
            }
        }
    }
}
