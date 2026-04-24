pub mod abba;
pub mod in_process;
pub mod subprocess;

use alloy_primitives::{Address, U256};
use arb_primitives::ArbTransactionSigned;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone)]
pub struct BlockInput {
    pub block_number: u64,
    pub timestamp: u64,
    pub base_fee: u64,
    pub gas_limit: u64,
    pub txs: Vec<ArbTransactionSigned>,
}

#[derive(Debug, Clone)]
pub struct Workload {
    pub manifest_name: String,
    pub chain_id: u64,
    pub arbos_version: u64,
    pub funded_accounts: Vec<(Address, U256)>,
    pub deployed_contracts: Vec<DeployedContract>,
    pub blocks: Vec<BlockInput>,
    pub prewarm_alloc: Option<PrewarmAlloc>,
}

#[derive(Debug, Clone)]
pub struct PrewarmAlloc {
    pub count: u64,
    pub seed: u64,
    pub balance: U256,
}

#[derive(Debug, Clone)]
pub struct DeployedContract {
    pub address: Address,
    pub runtime_code: Vec<u8>,
    pub balance: U256,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerConfig {
    pub rolling_window_blocks: usize,
    pub abort_on_block_error: bool,
}

impl Default for RunnerConfig {
    fn default() -> Self {
        Self {
            rolling_window_blocks: 500,
            abort_on_block_error: false,
        }
    }
}
