use alloc::sync::Arc;
use core::convert::Infallible;
use core::fmt::Debug;

use alloy_consensus::{BlockHeader, Header};
use alloy_eips::Decodable2718;
use alloy_evm::eth::EthBlockExecutionCtx;
use alloy_evm::eth::spec::EthExecutorSpec;
use alloy_primitives::{Bytes, B256, U256};
use alloy_rpc_types_engine::ExecutionData;
use arb_chainspec::ArbitrumChainSpec;
use reth_chainspec::{EthChainSpec, Hardforks};
use reth_ethereum_primitives::EthPrimitives;
use reth_evm::{
    ConfigureEngineEvm, ConfigureEvm, EvmEnv, EvmEnvFor, ExecutableTxIterator, ExecutionCtxFor,
    NextBlockEnvAttributes,
};
use reth_evm_ethereum::{EthBlockAssembler, RethReceiptBuilder};
use reth_primitives_traits::{SealedBlock, SealedHeader, SignedTransaction, TxTy};
use reth_storage_errors::any::AnyError;
use revm::context::{BlockEnv, CfgEnv};
use revm::context_interface::block::BlobExcessGasAndPrice;
use revm::primitives::hardfork::SpecId;

use crate::build::ArbBlockExecutorFactory;
use crate::context::ArbBlockExecutionCtx;
use crate::evm::ArbEvmFactory;

/// Arbitrum EVM configuration.
///
/// Wraps the Ethereum EVM config and overrides environment construction
/// to use ArbOS versioning from the mix_hash field.
#[derive(Debug, Clone)]
pub struct ArbEvmConfig<ChainSpec = reth_chainspec::ChainSpec> {
    pub executor_factory:
        ArbBlockExecutorFactory<RethReceiptBuilder, Arc<ChainSpec>, ArbEvmFactory>,
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
            executor_factory: ArbBlockExecutorFactory::new(
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

impl<ChainSpec> ConfigureEvm for ArbEvmConfig<ChainSpec>
where
    ChainSpec:
        EthExecutorSpec + EthChainSpec<Header = Header> + ArbitrumChainSpec + Hardforks + 'static,
{
    type Primitives = EthPrimitives;
    type Error = Infallible;
    type NextBlockEnvCtx = NextBlockEnvAttributes;
    type BlockExecutorFactory =
        ArbBlockExecutorFactory<RethReceiptBuilder, Arc<ChainSpec>, ArbEvmFactory>;
    type BlockAssembler = EthBlockAssembler<ChainSpec>;

    fn block_executor_factory(&self) -> &Self::BlockExecutorFactory {
        &self.executor_factory
    }

    fn block_assembler(&self) -> &Self::BlockAssembler {
        &self.block_assembler
    }

    fn evm_env(&self, header: &Header) -> Result<EvmEnv<SpecId>, Self::Error> {
        let chain_id = self.chain_spec.chain().id();
        let arbos_version =
            arbos_version_from_mix_hash(&header.mix_hash().unwrap_or_default());
        let spec = self.chain_spec.spec_id_by_arbos_version(arbos_version);

        let cfg_env = CfgEnv::new()
            .with_chain_id(chain_id)
            .with_spec_and_mainnet_gas_params(spec);
        let block_env = BlockEnv {
            number: U256::from(header.number()),
            beneficiary: header.beneficiary(),
            timestamp: U256::from(header.timestamp()),
            difficulty: header.difficulty(),
            prevrandao: header.mix_hash(),
            gas_limit: header.gas_limit(),
            basefee: header.base_fee_per_gas().unwrap_or_default(),
            blob_excess_gas_and_price: Some(BlobExcessGasAndPrice {
                excess_blob_gas: 0,
                blob_gasprice: 0,
            }),
        };

        Ok(EvmEnv { cfg_env, block_env })
    }

    fn next_evm_env(
        &self,
        parent: &Header,
        attributes: &NextBlockEnvAttributes,
    ) -> Result<EvmEnv<SpecId>, Self::Error> {
        let chain_id = self.chain_spec.chain().id();
        let arbos_version = arbos_version_from_mix_hash(&attributes.prev_randao);
        let spec = self.chain_spec.spec_id_by_arbos_version(arbos_version);

        let cfg_env = CfgEnv::new()
            .with_chain_id(chain_id)
            .with_spec_and_mainnet_gas_params(spec);
        let next_number = parent.number().saturating_add(1);
        let block_env = BlockEnv {
            number: U256::from(next_number),
            beneficiary: attributes.suggested_fee_recipient,
            timestamp: U256::from(attributes.timestamp),
            difficulty: U256::from(1),
            prevrandao: Some(attributes.prev_randao),
            gas_limit: attributes.gas_limit,
            basefee: parent.base_fee_per_gas().unwrap_or_default(),
            blob_excess_gas_and_price: Some(BlobExcessGasAndPrice {
                excess_blob_gas: 0,
                blob_gasprice: 0,
            }),
        };

        Ok(EvmEnv { cfg_env, block_env })
    }

    fn context_for_block<'a>(
        &self,
        block: &'a SealedBlock<reth_ethereum_primitives::Block>,
    ) -> Result<EthBlockExecutionCtx<'a>, Self::Error> {
        Ok(EthBlockExecutionCtx {
            tx_count_hint: Some(block.transaction_count()),
            parent_hash: block.header().parent_hash,
            parent_beacon_block_root: block.header().parent_beacon_block_root,
            ommers: &[],
            withdrawals: None,
            extra_data: block.header().extra_data.clone(),
        })
    }

    fn context_for_next_block(
        &self,
        parent: &SealedHeader<Header>,
        attributes: NextBlockEnvAttributes,
    ) -> Result<EthBlockExecutionCtx<'_>, Self::Error> {
        Ok(EthBlockExecutionCtx {
            tx_count_hint: None,
            parent_hash: parent.hash(),
            parent_beacon_block_root: attributes.parent_beacon_block_root,
            ommers: &[],
            withdrawals: None,
            extra_data: attributes.extra_data,
        })
    }
}

impl<ChainSpec> ConfigureEngineEvm<ExecutionData> for ArbEvmConfig<ChainSpec>
where
    ChainSpec: EthExecutorSpec
        + EthChainSpec<Header = Header>
        + ArbitrumChainSpec
        + Hardforks
        + 'static,
{
    fn evm_env_for_payload(
        &self,
        payload: &ExecutionData,
    ) -> Result<EvmEnvFor<Self>, Self::Error> {
        let prev_randao = payload.payload.as_v1().prev_randao;
        let arbos_version = arbos_version_from_mix_hash(&prev_randao);
        let spec = self.chain_spec.spec_id_by_arbos_version(arbos_version);

        let cfg_env = CfgEnv::new()
            .with_chain_id(self.chain_spec.chain().id())
            .with_spec_and_mainnet_gas_params(spec);

        let block_env = BlockEnv {
            number: U256::from(payload.payload.block_number()),
            beneficiary: payload.payload.fee_recipient(),
            timestamp: U256::from(payload.payload.timestamp()),
            difficulty: U256::ZERO,
            prevrandao: Some(prev_randao),
            gas_limit: payload.payload.gas_limit(),
            basefee: payload.payload.saturated_base_fee_per_gas(),
            blob_excess_gas_and_price: Some(BlobExcessGasAndPrice {
                excess_blob_gas: 0,
                blob_gasprice: 0,
            }),
        };

        Ok(EvmEnv { cfg_env, block_env })
    }

    fn context_for_payload<'a>(
        &self,
        payload: &'a ExecutionData,
    ) -> Result<ExecutionCtxFor<'a, Self>, Self::Error> {
        Ok(EthBlockExecutionCtx {
            tx_count_hint: Some(payload.payload.transactions().len()),
            parent_hash: payload.parent_hash(),
            parent_beacon_block_root: payload.sidecar.parent_beacon_block_root(),
            ommers: &[],
            withdrawals: None,
            extra_data: payload.payload.as_v1().extra_data.clone(),
        })
    }

    fn tx_iterator_for_payload(
        &self,
        payload: &ExecutionData,
    ) -> Result<impl ExecutableTxIterator<Self>, Self::Error> {
        let txs = payload.payload.transactions().clone();
        let convert = |tx: Bytes| {
            let tx =
                TxTy::<Self::Primitives>::decode_2718_exact(tx.as_ref()).map_err(AnyError::new)?;
            let signer = tx.try_recover().map_err(AnyError::new)?;
            Ok::<_, AnyError>(tx.with_signer(signer))
        };
        Ok((txs, convert))
    }
}

impl<ChainSpec> ArbEvmConfig<ChainSpec>
where
    ChainSpec: EthChainSpec + 'static,
{
    /// Build an `ArbBlockExecutionCtx` from a sealed block header.
    pub fn arb_context_for_block(
        &self,
        header: &Header,
        parent_hash: B256,
    ) -> ArbBlockExecutionCtx {
        let mix_hash = header.mix_hash;
        ArbBlockExecutionCtx {
            parent_hash,
            parent_beacon_block_root: header.parent_beacon_block_root,
            extra_data: header.extra_data.to_vec(),
            delayed_messages_read: u64::from_be_bytes(header.nonce.0),
            l1_block_number: l1_block_number_from_mix_hash(&mix_hash),
            chain_id: self.chain_spec.chain().id(),
            block_timestamp: header.timestamp,
            basefee: U256::from(header.base_fee_per_gas.unwrap_or_default()),
        }
    }

    /// Build an `ArbBlockExecutionCtx` from next-block attributes.
    pub fn arb_context_for_next_block(
        &self,
        parent: &SealedHeader<Header>,
        prev_randao: &B256,
        extra_data: &[u8],
    ) -> ArbBlockExecutionCtx {
        let l1_block_number = l1_block_number_from_mix_hash(prev_randao);
        ArbBlockExecutionCtx {
            parent_hash: parent.hash(),
            parent_beacon_block_root: parent.parent_beacon_block_root(),
            extra_data: extra_data.to_vec(),
            delayed_messages_read: 0,
            l1_block_number,
            chain_id: self.chain_spec.chain().id(),
            block_timestamp: parent.timestamp(),
            basefee: U256::from(parent.base_fee_per_gas().unwrap_or_default()),
        }
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
