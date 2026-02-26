use alloc::sync::Arc;
use core::fmt::Debug;

use alloy_evm::eth::EthBlockExecutorFactory;
use alloy_primitives::B256;
use reth_chainspec::EthChainSpec;
use reth_evm_ethereum::{EthBlockAssembler, RethReceiptBuilder};
use revm::primitives::hardfork::SpecId;

use crate::evm::ArbEvmFactory;

/// Arbitrum EVM configuration.
///
/// Wraps the Ethereum EVM config and overrides environment construction
/// to use ArbOS versioning from the mix_hash field.
#[derive(Debug, Clone)]
pub struct ArbEvmConfig<ChainSpec = reth_chainspec::ChainSpec> {
    pub executor_factory:
        EthBlockExecutorFactory<RethReceiptBuilder, Arc<ChainSpec>, ArbEvmFactory>,
    pub block_assembler: EthBlockAssembler<ChainSpec>,
    chain_spec: Arc<ChainSpec>,
}

impl<ChainSpec> ArbEvmConfig<ChainSpec>
where
    ChainSpec: EthChainSpec + 'static,
{
    /// Creates a new Arbitrum EVM configuration with the given chain spec.
    pub fn new(chain_spec: Arc<ChainSpec>) -> Self {
        let evm_factory = ArbEvmFactory::new();
        Self {
            executor_factory: EthBlockExecutorFactory::new(
                RethReceiptBuilder::default(),
                chain_spec.clone(),
                evm_factory,
            ),
            block_assembler: EthBlockAssembler::new(chain_spec.clone()),
            chain_spec,
        }
    }

    /// Returns a reference to the chain spec.
    pub fn chain_spec(&self) -> &Arc<ChainSpec> {
        &self.chain_spec
    }
}

/// Extract ArbOS version from header mix_hash (bytes 16-23).
pub fn arbos_version_from_mix_hash(mix_hash: &B256) -> u64 {
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&mix_hash.0[16..24]);
    u64::from_be_bytes(buf)
}

/// Extract L1 block number from header mix_hash (bytes 8-15).
pub fn l1_block_number_from_mix_hash(mix_hash: &B256) -> u64 {
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&mix_hash.0[8..16]);
    u64::from_be_bytes(buf)
}

/// Map ArbOS version to the appropriate SpecId.
///
/// Arbitrum uses ArbOS version (encoded in mix_hash) rather than
/// block number/timestamp to determine the active EVM spec.
pub fn arbos_version_to_spec_id(arbos_version: u64) -> SpecId {
    match arbos_version {
        0..=10 => SpecId::LONDON,
        11..=19 => SpecId::LONDON,
        20..=30 => SpecId::CANCUN,
        31..=39 => SpecId::CANCUN,
        40..=49 => SpecId::CANCUN,
        _ => SpecId::OSAKA,
    }
}
