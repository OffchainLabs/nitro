use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{B256, U256};
use arb_e2e_tests::helpers::{deploy_contract, fund_account, recover};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    context::{BlockEnv, CfgEnv},
    primitives::hardfork::SpecId,
};

use super::{BlockInput, RunnerConfig, Workload};
use crate::metrics::{
    clock::Stopwatch, memory::RssMonitor, rolling::build_windows, BlockMetric, HostInfo, RunResult,
    SummaryMetrics,
};

/// In-process runner: drives `ArbBlockExecutor` directly with no networking.
/// One instance executes one workload to completion.
pub struct InProcessRunner {
    config: RunnerConfig,
    rss: RssMonitor,
}

impl InProcessRunner {
    pub fn new(config: RunnerConfig) -> Self {
        Self {
            config,
            rss: RssMonitor::new(),
        }
    }

    /// Execute a workload and return the run result.
    pub fn run(&mut self, workload: Workload) -> eyre::Result<RunResult> {
        let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
        let cfg = ArbEvmConfig::new(chain_spec);

        let mut harness = ArbosHarness::new()
            .with_arbos_version(workload.arbos_version)
            .with_chain_id(workload.chain_id)
            .initialize();

        for (addr, bal) in &workload.funded_accounts {
            fund_account(harness.state(), *addr, *bal);
        }
        for c in &workload.deployed_contracts {
            deploy_contract(
                harness.state(),
                c.address,
                c.runtime_code.clone(),
                c.balance,
            );
        }

        let mut blocks = Vec::with_capacity(workload.blocks.len());
        for block in &workload.blocks {
            let metric = self.execute_one_block(&cfg, &mut harness, workload.chain_id, block)?;
            blocks.push(metric);
        }

        let windows = build_windows(&blocks, self.config.rolling_window_blocks);
        let summary = SummaryMetrics::from_blocks(&blocks, &windows);
        Ok(RunResult {
            manifest_name: workload.manifest_name,
            blocks,
            windows,
            summary,
            host: HostInfo::collect(),
        })
    }

    fn execute_one_block(
        &mut self,
        cfg: &ArbEvmConfig,
        harness: &mut ArbosHarness,
        chain_id: u64,
        block: &BlockInput,
    ) -> eyre::Result<BlockMetric> {
        let mut env: EvmEnv<SpecId> = EvmEnv {
            cfg_env: CfgEnv::default(),
            block_env: BlockEnv::default(),
        };
        env.cfg_env.chain_id = chain_id;
        env.cfg_env.disable_base_fee = true;
        env.block_env.timestamp = U256::from(block.timestamp);
        env.block_env.basefee = block.base_fee;
        env.block_env.gas_limit = block.gas_limit;
        env.block_env.number = U256::from(block.block_number);

        let evm = cfg
            .block_executor_factory()
            .evm_factory()
            .create_evm(harness.state(), env);

        let exec_ctx = EthBlockExecutionCtx {
            tx_count_hint: Some(block.txs.len()),
            parent_hash: B256::ZERO,
            parent_beacon_block_root: None,
            ommers: &[],
            withdrawals: None,
            extra_data: vec![0u8; 32].into(),
        };

        let mut executor = cfg
            .block_executor_factory()
            .create_arb_executor(evm, exec_ctx, chain_id);

        let sw = Stopwatch::start();
        let mut gas_used: u64 = 0;
        let mut success_count: usize = 0;

        executor
            .apply_pre_execution_changes()
            .map_err(|e| eyre::eyre!("pre-execution: {e:?}"))?;

        for tx in &block.txs {
            let recovered = recover(tx.clone());
            match executor.execute_transaction_without_commit(recovered) {
                Ok(result) => {
                    if result.result.result.is_success() {
                        success_count += 1;
                    }
                    gas_used = gas_used.saturating_add(result.result.result.gas_used());
                    executor
                        .commit_transaction(result)
                        .map_err(|e| eyre::eyre!("commit: {e:?}"))?;
                }
                Err(e) if !self.config.abort_on_block_error => {
                    tracing::trace!(?e, "tx rejected");
                }
                Err(e) => return Err(eyre::eyre!("tx error: {e:?}")),
            }
        }

        let _ = executor
            .finish()
            .map_err(|e| eyre::eyre!("finish: {e:?}"))?;

        let (wall, cpu) = sw.elapsed_ns();
        let rss = self.rss.current_rss();

        Ok(BlockMetric {
            block_number: block.block_number,
            wall_clock_ns: wall,
            cpu_ns: cpu,
            gas_used,
            tx_count: block.txs.len(),
            success_count,
            rss_bytes: rss,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::capture::synthetic::generate;

    #[test]
    fn runner_executes_transfer_train() {
        let mut r = InProcessRunner::new(RunnerConfig {
            rolling_window_blocks: 2,
            abort_on_block_error: false,
        });
        let g = generate(
            "test/transfer_train",
            421614,
            30,
            "transfer_train",
            &serde_json::json!({ "block_count": 4, "txs_per_block": 3 }),
        )
        .unwrap();
        let result = r.run(g).expect("run ok");
        assert_eq!(result.blocks.len(), 4);
        assert!(result.blocks.iter().all(|b| b.tx_count == 3));
        assert!(result.summary.total_gas > 0);
        assert!(!result.windows.is_empty());
    }

    #[test]
    fn runner_handles_revert_storm_without_aborting() {
        let mut r = InProcessRunner::new(RunnerConfig::default());
        let g = generate(
            "test/revert_storm",
            421614,
            30,
            "revert_storm",
            &serde_json::json!({ "block_count": 2, "txs_per_block": 4 }),
        )
        .unwrap();
        let result = r.run(g).expect("run ok");
        assert_eq!(result.blocks.len(), 2);
    }
}
