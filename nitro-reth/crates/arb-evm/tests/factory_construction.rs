use std::sync::Arc;

use alloy_consensus::Header;
use alloy_primitives::B256;
use arb_evm::{config::ArbEvmConfig, evm::ArbEvmFactory, receipt::ArbReceiptBuilder};
use reth_chainspec::ChainSpec;
use reth_evm::ConfigureEvm;
use revm::primitives::hardfork::SpecId;

#[test]
fn arb_evm_factory_is_default_constructible() {
    let _ = ArbEvmFactory::new();
}

#[test]
fn arb_evm_config_constructs_with_chain_spec() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec.clone());
    assert!(Arc::ptr_eq(cfg.chain_spec(), &chain_spec));
}

#[test]
fn arb_evm_config_exposes_executor_factory() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let _factory: &_ = cfg.block_executor_factory();
}

#[test]
fn arb_evm_config_exposes_block_assembler() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let _assembler: &_ = cfg.block_assembler();
}

#[test]
#[allow(clippy::default_constructed_unit_structs)]
fn receipt_builder_is_default_constructible() {
    let _ = ArbReceiptBuilder::default();
    let _ = ArbReceiptBuilder;
}

fn header_with_arbos_version(arbos_version: u64) -> Header {
    let mut mix_hash = [0u8; 32];
    mix_hash[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    Header {
        mix_hash: B256::from(mix_hash),
        ..Default::default()
    }
}

#[test]
fn evm_env_for_v40_header_uses_prague_spec() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let env = cfg.evm_env(&header_with_arbos_version(40)).unwrap();
    assert_eq!(env.cfg_env.spec, SpecId::PRAGUE);
}

#[test]
fn evm_env_for_v32_header_still_uses_cancun_spec() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let env = cfg.evm_env(&header_with_arbos_version(32)).unwrap();
    assert_eq!(env.cfg_env.spec, SpecId::CANCUN);
}
