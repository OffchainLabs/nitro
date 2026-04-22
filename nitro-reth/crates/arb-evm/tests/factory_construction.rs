use std::sync::Arc;

use arb_evm::{config::ArbEvmConfig, evm::ArbEvmFactory, receipt::ArbReceiptBuilder};
use reth_chainspec::ChainSpec;
use reth_evm::ConfigureEvm;

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
