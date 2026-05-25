//! Regression: Stylus ERC-1155 `safeTransferFrom` at Sepolia block 253,170,068.
//!
//! tx[1] 0x0779a1d0… calls Stylus contract 0x8bb9… (selector 0xf242432a).
//! Canonical receipt: status=1, gasUsed=91,243, gasUsedForL1=0, one TransferSingle
//! log. The transfer moves id 0xfa8c…5111 amount 1 from 0xe4dba6 to 0xbbb286,
//! decrementing balances[id][from] 3->2 and setting balances[id][to] 0->1.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::{transaction::Recovered, EthereumTxEnvelope, SignableTransaction, TxEip1559};
use alloy_evm::{block::BlockExecutorFactory, eth::EthBlockExecutionCtx, EvmFactory};
use alloy_primitives::{address, b256, hex, Address, Bytes, Signature, TxKind, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::{deploy_contract, fund_account, recover};
use arb_primitives::ArbTransactionSigned;
use arb_storage::write_storage_at;
use arb_test_utils::ArbosHarness;
use arbos::programs::Program;
use reth_chainspec::ChainSpec;
use reth_evm::{block::BlockExecutor, ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const CONTRACT: Address = address!("8bb9a1f6be8857d530ec73a5febb57d9d02c71a3");
const SENDER: Address = address!("e4dba6b5e1d9fc2dfe8cb802ac26e0d2f8f23458");

// balances[id][from] — id-first nested mapping at base slot 1.
const FROM_BALANCE_SLOT: B256 =
    b256!("3ae61de303b6fed0e1e4a7d243b531a840df3a83ebc8909e7e84b58007273e45");

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 51;
const STYLUS_VERSION: u16 = 2;
const PAGE_LIMIT: u16 = 128;

const BLOCK_NUMBER: u64 = 253_170_068;
const BLOCK_TIMESTAMP: u64 = 0x69c2c847;
const L1_BLOCK_NUMBER: u64 = 0xa06b05;
const BASE_FEE: u128 = 0x131e880;

const TX_NONCE: u64 = 5;
const TX_GAS_LIMIT: u64 = 0x1681c; // 92,188
const TX_MAX_FEE: u128 = 0x131e880;
const TX_MAX_PRIO: u128 = 0x131e880;
const SENDER_BALANCE: u128 = 0x5f7a7d712369d20;

const CANON_GAS_USED: u64 = 91_243;

// Activated program metadata stored on-chain at this block.
const CANON_INIT_COST: u16 = 16_760;
const CANON_CACHED_COST: u16 = 6_633;
const CANON_FOOTPRINT: u16 = 1;
const CANON_ASM_KB: u32 = 1_537;
const CANON_ACTIVATED_AT_HOURS: u32 = 98_008;

const STYLUS_HEX: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_253170068_assets/stylus_erc1155.hex"
);

const TX_INPUT_HEX: &str = "f242432a000000000000000000000000e4dba6b5e1d9fc2dfe8cb802ac26e0d2f8f23458000000000000000000000000bbb286d2b9051578974ea3e7751f09d4803a7cfcfa8c303e6c2c6254c4af25dc6e40c5bd24408e2e3320495bff55cdd1efc25111000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000";

fn stylus_bytecode() -> Vec<u8> {
    hex::decode(STYLUS_HEX.trim().trim_start_matches("0x")).expect("decode stylus hex")
}

/// The real signed transaction, so it recovers to the canonical sender.
fn build_tx() -> ArbTransactionSigned {
    let tx = TxEip1559 {
        chain_id: CHAIN_ID,
        nonce: TX_NONCE,
        gas_limit: TX_GAS_LIMIT,
        max_fee_per_gas: TX_MAX_FEE,
        max_priority_fee_per_gas: TX_MAX_PRIO,
        to: TxKind::Call(CONTRACT),
        value: U256::ZERO,
        access_list: Default::default(),
        input: Bytes::from(hex::decode(TX_INPUT_HEX).expect("decode input")),
    };
    let sig = Signature::new(
        U256::from_be_bytes(
            b256!("932bcda1adb5fd55a0e1550666e06ce75e5d4e222a6eb9ca0f9da791ddd9ed55").0,
        ),
        U256::from_be_bytes(
            b256!("5ed276154a821a964ece714be7e6b1f01ef0822b9846571dd9580ffa95431c97").0,
        ),
        true,
    );
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip1559(tx.into_signed(sig)))
}

fn run() -> (u64, bool, usize) {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    let bytecode = stylus_bytecode();
    let code_hash = alloy_primitives::keccak256(&bytecode);

    // The block's L2 base fee was 0.02 gwei; the harness defaults to the 0.1
    // gwei floor, which would reject the tx's max fee. l1_base_fee stays 0 so
    // poster (L1) gas is 0, matching gasUsedForL1=0.
    {
        let state_ptr = harness.state_ptr();
        let state = harness.arbos_state();
        let backend = unsafe { &mut *state_ptr };
        state
            .l2_pricing_state
            .set_min_base_fee_wei(backend, U256::from(BASE_FEE))
            .expect("set min base fee");
        state
            .l2_pricing_state
            .set_base_fee_wei(backend, U256::from(BASE_FEE))
            .expect("set base fee");
    }

    // Activate locally for the module hash; assert metadata matches canonical.
    let wasm = arb_stylus::decompress_wasm(&bytecode).expect("decompress");
    let mut gas = u64::MAX;
    let activation = arb_stylus::activate_program(
        &wasm,
        code_hash.as_ref(),
        STYLUS_VERSION,
        ARBOS_VERSION,
        PAGE_LIMIT,
        false,
        &mut gas,
    )
    .expect("activate");
    assert_eq!(activation.init_gas, CANON_INIT_COST, "init_cost parity");
    assert_eq!(
        activation.cached_init_gas, CANON_CACHED_COST,
        "cached_cost parity"
    );
    assert_eq!(activation.footprint, CANON_FOOTPRINT, "footprint parity");

    let program = Program {
        version: STYLUS_VERSION,
        init_cost: CANON_INIT_COST,
        cached_cost: CANON_CACHED_COST,
        footprint: CANON_FOOTPRINT,
        asm_estimate_kb: CANON_ASM_KB,
        activated_at: CANON_ACTIVATED_AT_HOURS,
        age_seconds: 0,
        cached: false,
    };
    {
        let state_ptr = harness.state_ptr();
        let state = harness.arbos_state();
        let backend = unsafe { &mut *state_ptr };
        state
            .programs
            .set_module_hash(backend, code_hash, activation.module_hash)
            .expect("set module hash");
        state
            .programs
            .set_program(backend, code_hash, program)
            .expect("set program");
    }

    deploy_contract(harness.state(), CONTRACT, bytecode, U256::ZERO);

    fund_account(harness.state(), SENDER, U256::from(SENDER_BALANCE));
    if let Some(acct) = harness
        .state()
        .cache
        .accounts
        .get_mut(&SENDER)
        .and_then(|c| c.account.as_mut())
    {
        acct.info.nonce = TX_NONCE;
    }

    write_storage_at(
        harness.state(),
        CONTRACT,
        U256::from_be_bytes(FROM_BALANCE_SLOT.0),
        U256::from(3u64),
    )
    .expect("seed from balance");

    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.cfg_env.tx_gas_limit_cap = Some(u64::MAX);
    env.block_env.timestamp = U256::from(BLOCK_TIMESTAMP);
    env.block_env.basefee = BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        L1_BLOCK_NUMBER,
        false,
    );
    let staged_ctx = Arc::new(arb_context::ArbPrecompileCtx::with_block(Arc::new(
        block_ctx,
    )));
    evm_factory.stage_ctx(staged_ctx);
    let evm = evm_factory.create_evm(harness.state(), env);
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor
        .apply_pre_execution_changes()
        .expect("pre-execution");

    let tx = build_tx();
    let recovered: Recovered<ArbTransactionSigned> = recover(tx);
    assert_eq!(recovered.signer(), SENDER, "recovered sender");

    let exec_result = executor
        .execute_transaction_without_commit(recovered)
        .expect("execute tx");

    let gas_used = exec_result.result.result.gas_used();
    let success = exec_result.result.result.is_success();
    let logs = exec_result.result.result.logs().len();
    (gas_used, success, logs)
}

#[test]
fn erc1155_safe_transfer_matches_canonical_gas() {
    let (gas_used, success, logs) = run();
    assert!(success, "tx should succeed");
    assert_eq!(logs, 1, "should emit one TransferSingle log");
    assert_eq!(
        gas_used,
        CANON_GAS_USED,
        "gas_used {gas_used} != canonical {CANON_GAS_USED} (drift {})",
        gas_used as i128 - CANON_GAS_USED as i128,
    );
}
