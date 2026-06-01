//! Rebuild derived state from the canonical blocks already in the database.
//!
//! Re-executes every block forward from genesis over the stored headers and
//! bodies, rewriting the execution state, changesets, hashed state, trie, and
//! history indices. Headers, bodies, and receipts (the canonical block data)
//! are left untouched. The merkle stage validates each block's recomputed
//! state root against the canonical header, so a state divergence halts the
//! run at the offending block rather than persisting incorrect state.

use clap::Parser;
use reth_chainspec::{EthChainSpec, EthereumHardforks, Hardforks};
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{
    AccessRights, CliComponentsBuilder, CliNodeComponents, CliNodeTypes, Environment,
    EnvironmentArgs,
};
use reth_config::Config;
use reth_consensus::noop::NoopConsensus;
use reth_evm::ConfigureEvm;
use reth_provider::{
    providers::ProviderNodeTypes, BlockNumReader, ChainSpecProvider, ProviderFactory,
};
use reth_stages::{sets::OfflineStages, Pipeline};
use reth_static_file::StaticFileProducer;
use tracing::*;

/// `arb-reth repair` command.
///
/// Reset derived state to genesis and re-execute all stored blocks forward,
/// validating each block's state root against its canonical header.
#[derive(Debug, Parser)]
pub struct Command<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Highest block to rebuild to. Defaults to the chain tip. Bounding the
    /// rebuild is useful for validating the merkle state root over a smaller
    /// range before committing to a full rebuild.
    #[arg(long)]
    to: Option<u64>,
}

impl<C: ChainSpecParser<ChainSpec: EthChainSpec + Hardforks + EthereumHardforks>> Command<C> {
    /// Execute the `repair` command.
    pub async fn execute<N>(
        self,
        components: impl CliComponentsBuilder<N>,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()>
    where
        N: CliNodeTypes<ChainSpec = C::ChainSpec>,
    {
        let Environment {
            provider_factory,
            config,
            ..
        } = self.env.init::<N>(AccessRights::RW, runtime)?;

        let components = components(provider_factory.chain_spec());
        let chain_tip = provider_factory.provider()?.last_block_number()?;
        let target = self.to.unwrap_or(chain_tip).min(chain_tip);

        info!(
            target: "arb::repair",
            target,
            "Rebuilding derived state from canonical blocks; headers and bodies are preserved"
        );

        let mut pipeline = build_pipeline(
            &config,
            provider_factory,
            components.evm_config().clone(),
            target,
        );

        pipeline.move_to_static_files()?;

        // Re-execute forward. The derived stages are expected to already be at
        // genesis (reset via `stage drop`), so this unwind is a no-op; running
        // it keeps the command correct when the stages are mid-chain. The merkle
        // stage validates each checkpoint's state root against the canonical
        // header, halting the run on mismatch instead of persisting bad state.
        pipeline.unwind(0, None)?;
        info!(target: "arb::repair", target, "Re-executing forward");
        pipeline.run().await?;

        info!(
            target: "arb::repair",
            target,
            "Repair complete; recomputed state root matches the canonical header"
        );
        Ok(())
    }
}

fn build_pipeline<N>(
    config: &Config,
    provider_factory: ProviderFactory<N>,
    evm_config: impl ConfigureEvm<Primitives = N::Primitives> + 'static,
    tip: u64,
) -> Pipeline<N>
where
    N: ProviderNodeTypes,
{
    let prune_modes = config.prune.segments.clone();
    Pipeline::<N>::builder()
        .with_max_block(tip)
        .with_fail_on_unwind(true)
        .add_stages(OfflineStages::new(
            evm_config,
            NoopConsensus::arc(),
            config.stages.clone(),
            prune_modes.clone(),
        ))
        .build(
            provider_factory.clone(),
            StaticFileProducer::new(provider_factory, prune_modes),
        )
}
