use alloc::sync::Arc;
use core::{convert::Infallible, fmt::Debug};

use alloy_consensus::{BlockHeader, Header};
use alloy_eips::Decodable2718;
use alloy_evm::eth::{spec::EthExecutorSpec, EthBlockExecutionCtx};
use alloy_primitives::{Address, Bytes, B256, U256};
use alloy_rpc_types_engine::ExecutionData;
use arb_chainspec::ArbitrumChainSpec;
use arb_primitives::ArbPrimitives;
use reth_chainspec::{EthChainSpec, Hardforks};
use reth_evm::{
    ConfigureEngineEvm, ConfigureEvm, EvmEnv, EvmEnvFor, ExecutableTxIterator, ExecutionCtxFor,
    NextBlockEnvAttributes,
};

use crate::{assembler::ArbBlockAssembler, receipt::ArbReceiptBuilder};
use reth_primitives_traits::{SealedBlock, SealedHeader, SignedTransaction, TxTy};
use reth_storage_errors::any::AnyError;
use revm::{
    context::{BlockEnv, CfgEnv},
    primitives::hardfork::SpecId,
};

use crate::{build::ArbBlockExecutorFactory, context::ArbBlockExecutionCtx, evm::ArbEvmFactory};

/// Arbitrum EVM configuration.
///
/// Wraps the Ethereum EVM config and overrides environment construction
/// to use ArbOS versioning from the mix_hash field.
#[derive(Debug, Clone)]
pub struct ArbEvmConfig<ChainSpec = reth_chainspec::ChainSpec> {
    pub executor_factory: ArbBlockExecutorFactory<ArbReceiptBuilder, Arc<ChainSpec>, ArbEvmFactory>,
    pub block_assembler: ArbBlockAssembler<ChainSpec>,
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
                ArbReceiptBuilder::default(),
                chain_spec.clone(),
                evm_factory,
            ),
            block_assembler: ArbBlockAssembler::new(chain_spec.clone()),
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
    type Primitives = ArbPrimitives;
    type Error = Infallible;
    type NextBlockEnvCtx = NextBlockEnvAttributes;
    type BlockExecutorFactory =
        ArbBlockExecutorFactory<ArbReceiptBuilder, Arc<ChainSpec>, ArbEvmFactory>;
    type BlockAssembler = ArbBlockAssembler<ChainSpec>;

    fn block_executor_factory(&self) -> &Self::BlockExecutorFactory {
        &self.executor_factory
    }

    fn block_assembler(&self) -> &Self::BlockAssembler {
        &self.block_assembler
    }

    fn evm_env(&self, header: &Header) -> Result<EvmEnv<SpecId>, Self::Error> {
        let chain_id = self.chain_spec.chain().id();
        let mix_hash = header.mix_hash().unwrap_or_default();
        let arbos_version = arbos_version_from_mix_hash(&mix_hash);
        let spec = self.chain_spec.spec_id_by_arbos_version(arbos_version);

        // Arbitrum overrides NUMBER to return the L1 block number, not L2.
        let l1_block_number = l1_block_number_from_mix_hash(&mix_hash);

        let cfg_env = arb_cfg_env(chain_id, spec, arbos_version);
        // Arbitrum sets PREVRANDAO to BigToHash(difficulty), which is 0x...0001.
        let prevrandao = B256::from(U256::from(1));
        let block_env = BlockEnv {
            number: U256::from(l1_block_number),
            beneficiary: header.beneficiary(),
            timestamp: U256::from(header.timestamp()),
            difficulty: header.difficulty(),
            prevrandao: Some(prevrandao),
            gas_limit: header.gas_limit(),
            basefee: header.base_fee_per_gas().unwrap_or_default(),
            // Arbitrum has no blobs — BLOBBASEFEE opcode returns 0.
            blob_excess_gas_and_price: None,
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

        let cfg_env = arb_cfg_env(chain_id, spec, arbos_version);
        // Arbitrum overrides NUMBER to return the L1 block number, not L2.
        let l1_block_number = l1_block_number_from_mix_hash(&attributes.prev_randao);
        // Arbitrum sets PREVRANDAO to BigToHash(difficulty), which is 0x...0001.
        let prevrandao = B256::from(U256::from(1));
        let block_env = BlockEnv {
            number: U256::from(l1_block_number),
            beneficiary: attributes.suggested_fee_recipient,
            timestamp: U256::from(attributes.timestamp),
            difficulty: U256::from(1),
            prevrandao: Some(prevrandao),
            gas_limit: attributes.gas_limit,
            basefee: parent.base_fee_per_gas().unwrap_or_default(),
            // Arbitrum has no blobs — BLOBBASEFEE opcode returns 0.
            blob_excess_gas_and_price: None,
        };

        Ok(EvmEnv { cfg_env, block_env })
    }

    fn context_for_block<'a>(
        &self,
        block: &'a SealedBlock<alloy_consensus::Block<arb_primitives::ArbTransactionSigned>>,
    ) -> Result<EthBlockExecutionCtx<'a>, Self::Error> {
        // Encode delayed_messages_read (from header nonce) as bytes 32-39 of extra_data,
        // and L2 block number as bytes 40-47.
        let mut extra = block.header().extra_data.to_vec();
        extra.extend_from_slice(&block.header().nonce.0);
        extra.extend_from_slice(&block.header().number.to_be_bytes());
        Ok(EthBlockExecutionCtx {
            tx_count_hint: Some(block.transaction_count()),
            parent_hash: block.header().parent_hash,
            parent_beacon_block_root: block.header().parent_beacon_block_root,
            ommers: &[],
            withdrawals: None,
            extra_data: extra.into(),
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
    ChainSpec:
        EthExecutorSpec + EthChainSpec<Header = Header> + ArbitrumChainSpec + Hardforks + 'static,
{
    fn evm_env_for_payload(&self, payload: &ExecutionData) -> Result<EvmEnvFor<Self>, Self::Error> {
        let prev_randao = payload.payload.as_v1().prev_randao;
        let arbos_version = arbos_version_from_mix_hash(&prev_randao);
        let spec = self.chain_spec.spec_id_by_arbos_version(arbos_version);

        let cfg_env = arb_cfg_env(self.chain_spec.chain().id(), spec, arbos_version);

        // Arbitrum overrides NUMBER to return the L1 block number, not L2.
        let l1_block_number = l1_block_number_from_mix_hash(&prev_randao);
        // Arbitrum sets PREVRANDAO to BigToHash(difficulty), which is 0x...0001.
        let prevrandao = B256::from(U256::from(1));
        let block_env = BlockEnv {
            number: U256::from(l1_block_number),
            beneficiary: payload.payload.fee_recipient(),
            timestamp: U256::from(payload.payload.timestamp()),
            difficulty: U256::from(1),
            prevrandao: Some(prevrandao),
            gas_limit: payload.payload.gas_limit(),
            basefee: payload.payload.saturated_base_fee_per_gas(),
            // Arbitrum has no blobs — BLOBBASEFEE opcode returns 0.
            blob_excess_gas_and_price: None,
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
            l2_block_number: header.number,
            chain_id: self.chain_spec.chain().id(),
            block_timestamp: header.timestamp,
            basefee: U256::from(header.base_fee_per_gas.unwrap_or_default()),
            time_passed: 0,
            l1_base_fee: U256::ZERO,
            arbos_version: arbos_version_from_mix_hash(&mix_hash),
            coinbase: header.beneficiary,
            // State-derived fields populated by block executor after state open.
            l1_price_per_unit: U256::ZERO,
            brotli_compression_level: 0,
            network_fee_account: Address::ZERO,
            infra_fee_account: Address::ZERO,
            min_base_fee: U256::ZERO,
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
            l2_block_number: parent.number().saturating_add(1),
            chain_id: self.chain_spec.chain().id(),
            block_timestamp: parent.timestamp(),
            basefee: U256::from(parent.base_fee_per_gas().unwrap_or_default()),
            time_passed: 0,
            l1_base_fee: U256::ZERO,
            arbos_version: 0,
            coinbase: Address::ZERO,
            l1_price_per_unit: U256::ZERO,
            brotli_compression_level: 0,
            network_fee_account: Address::ZERO,
            infra_fee_account: Address::ZERO,
            min_base_fee: U256::ZERO,
        }
    }
}

/// Build a `CfgEnv` with Arbitrum-specific overrides.
///
/// Disables EIP-3541 (0xEF rejection) for Stylus-era blocks so that
/// Stylus WASM programs can be deployed. Disables the priority fee
/// ordering check (Arbitrum tips are always dropped). Disables EIP-7623
/// increased calldata cost (irrelevant on L2 without blobs).
fn arb_cfg_env(chain_id: u64, spec: SpecId, arbos_version: u64) -> CfgEnv {
    let mut cfg = CfgEnv::new()
        .with_chain_id(chain_id)
        .with_spec_and_mainnet_gas_params(spec);
    // Arbitrum drops tips — max_priority_fee can exceed max_fee.
    cfg.disable_priority_fee_check = true;
    // EIP-7623 increases calldata cost for blob-less chains; irrelevant on L2.
    cfg.disable_eip7623 = true;
    // EIP-3607 rejects txs from senders with deployed code. Arbitrum L1-to-L2
    // tx types (ContractTx, RetryTx) may have L1 contract alias senders with
    // code on L2. skipTransactionChecks() skips this for those types.
    cfg.disable_eip3607 = true;
    // Stylus programs start with 0xEF; allow deployment once Stylus is live.
    if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS {
        cfg.disable_eip3541 = true;
    }
    // Disable revm's nonce and balance validation globally. Arbitrum's internal,
    // deposit, and retryable tx types need to bypass these checks (special
    // balance/nonce semantics). We manually validate balance for user txs in
    // execute_transaction_without_commit. disable_nonce_check only disables
    // validation, not increment — the nonce is still incremented after execution.
    cfg.disable_balance_check = true;
    cfg.disable_nonce_check = true;
    // Disable base fee validation for Arbitrum tx types (RetryTx, etc.)
    // whose gas_fee_cap may not follow standard EIP-1559 rules.
    // Also needed for debug_traceTransaction to replay these tx types.
    cfg.disable_base_fee = true;
    cfg
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
