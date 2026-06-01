//! Regression: ArbInternalTx `startBlock` at Sepolia block 269,589,709 (v60).
//!
//! Replays the canonical tx[0] (`startBlock(0, 10880719, 269589709, 1)`)
//! against a harness whose ArbOS-state and EIP-2935 storage slots are
//! seeded with the canonical pre-state captured via
//! `debug_traceBlockByNumber(diffMode=true)` from arb-sepolia (Alchemy).
//!
//! The four canonical post-state writes the StartBlock InternalTx makes are:
//!
//! - EIP-2935 history contract slot `0x41d3c` ((l2_block-1) % 393168 = 269628) ← parent hash
//!   0x19b751…4180e
//! - ArbOS-state `0x3c79…fe600` ← new L1 block number 10,880,719 (0xa606cf)
//! - ArbOS-state `0x3c79…fe6cf` ← parent hash (legacy ring-buffer slot for the previous L1 block)
//! - ArbOS-state `0xe54d…8202` ← 0x1312d00 (L2 base fee post-drain)
//!
//! The test asserts each value byte-exactly.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, b256, Address, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{
    arbos_versions::{HISTORY_STORAGE_ADDRESS, HISTORY_STORAGE_CODE_ARBITRUM},
    signed_tx::ArbTypedTransaction,
    ArbTransactionSigned,
};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at, ARBOS_STATE_ADDRESS};
use arb_test_utils::ArbosHarness;
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;

const BLOCK_NUMBER: u64 = 269_589_709;
const BLOCK_TIMESTAMP: u64 = 0x6a1029ba;
const PARENT_HASH: B256 = b256!("19b751952803e4efe00151b1e888b0634a5e1b2ae3f88e6911e6ca0c1424180e");

const NEW_L1_BLOCK_NUMBER: u64 = 0xa606cf;
const OLD_L1_BLOCK_NUMBER: u64 = 0xa606cd;
const TIME_PASSED: u64 = 1;

const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

// Canonical pre-state slots on the ArbOS-state account.
const ARBOS_L1_BLOCK_SLOT: B256 =
    b256!("3c79da47f96b0f39664f73c0a1f350580be90742947dddfa21ba64d578dfe600");
const ARBOS_PREV_PARENT_SLOT: B256 =
    b256!("3c79da47f96b0f39664f73c0a1f350580be90742947dddfa21ba64d578dfe6cf");
const ARBOS_L2_BASE_FEE_SLOT: B256 =
    b256!("e54de2a4cdacc0a0059d2b6e16348103df8c4aff409c31e40ec73d11926c8202");

const ARBOS_PRESTATE_PREV_PARENT: B256 =
    b256!("5a1ebee2afdb002e11cdd3de14427a1597c135783b3e2477778b0f5db17d0a0c");
const ARBOS_PRESTATE_L2_BASE_FEE: u64 = 0x1315410;
const ARBOS_POST_L2_BASE_FEE: u64 = 0x1312d00;

// EIP-2935 slot (l2_block-1) % 393168 = 269628 = 0x41d3c
const EIP2935_SLOT_DECIMAL: u64 = 269628;
const EIP2935_PRESTATE_VALUE: B256 =
    b256!("b2d49ba6e5526e912a33ee32efcaa3278e29dfeb3f20db8927a1ac178b1b26ee");

fn zero_sig() -> alloy_primitives::Signature {
    alloy_primitives::Signature::new(U256::ZERO, U256::ZERO, false)
}

fn read_slot<D: revm::Database>(
    state: &mut revm::database::State<D>,
    addr: Address,
    slot: U256,
) -> U256 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|c| c.account.as_ref())
        .and_then(|a| a.storage.get(&slot).copied())
        .unwrap_or(U256::ZERO)
}

/// Read a slot from the committed bundle (plain) state — what re-execution
/// reads back and what the state root is computed from. A write that lands in
/// the cache but not here is invisible to the next block and to the root.
fn read_bundle_slot<D: revm::Database>(
    state: &revm::database::State<D>,
    addr: Address,
    slot: U256,
) -> U256 {
    state
        .bundle_state
        .state
        .get(&addr)
        .and_then(|a| a.storage.get(&slot))
        .map(|s| s.present_value)
        .unwrap_or(U256::ZERO)
}

#[test]
fn v60_internal_tx_start_block_writes_canonical_slots() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    // Seed L2 pricing min_base_fee_wei to the canonical post-block value so
    // the v60 multi-gas update settles to the canonical 0x1312d00. Without
    // this, the harness's default 0.1-gwei floor traps the recompute at 100M.
    {
        let state_ptr = harness.state_ptr();
        let state = harness.arbos_state();
        // SAFETY: state_ptr aliases the harness's owned State; `state` borrows only
        // storage slots through `backend`, so the two handles do not race, and the
        // harness outlives the borrow.
        let backend = unsafe { &mut *state_ptr };
        state
            .l2_pricing_state
            .set_min_base_fee_wei(backend, U256::from(ARBOS_POST_L2_BASE_FEE))
            .expect("set min base fee");
    }

    // Seed EIP-2935 history contract: code, nonce=1, and prestate slot value.
    set_account_code(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        HISTORY_STORAGE_CODE_ARBITRUM.clone(),
    );
    set_account_nonce(harness.state(), HISTORY_STORAGE_ADDRESS, 1);
    write_storage_at(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        U256::from(EIP2935_SLOT_DECIMAL),
        U256::from_be_bytes(EIP2935_PRESTATE_VALUE.0),
    )
    .expect("seed eip2935 prestate");

    // Seed ArbOS-state slots with canonical pre-block values.
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
        U256::from(OLD_L1_BLOCK_NUMBER),
    )
    .expect("seed old l1 block");
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_PREV_PARENT_SLOT.0),
        U256::from_be_bytes(ARBOS_PRESTATE_PREV_PARENT.0),
    )
    .expect("seed prev parent");
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L2_BASE_FEE_SLOT.0),
        U256::from(ARBOS_PRESTATE_L2_BASE_FEE),
    )
    .expect("seed l2 base fee");

    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    let calldata = encode_start_block(U256::ZERO, NEW_L1_BLOCK_NUMBER, BLOCK_NUMBER, TIME_PASSED);
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: calldata.into(),
        }),
        zero_sig(),
    );

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
    env.block_env.basefee = ARBOS_PRESTATE_L2_BASE_FEE;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        OLD_L1_BLOCK_NUMBER, // pre-StartBlock L1 block number
        false,
    );
    let staged_ctx = Arc::new(arb_context::ArbPrecompileCtx::with_block(Arc::new(
        block_ctx,
    )));
    evm_factory.stage_ctx(staged_ctx);
    let evm = evm_factory.create_evm(harness.state(), env);

    // Pass the L2 block number through extra_data per the executor wiring
    // (bytes 40..48 hold the L2 block number for the EIP-2935 slot derivation).
    let mut extra = vec![0u8; 48];
    extra[40..48].copy_from_slice(&BLOCK_NUMBER.to_be_bytes());

    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: PARENT_HASH,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: extra.into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(ARBOS_PRESTATE_L2_BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = OLD_L1_BLOCK_NUMBER;
    executor.arb_ctx.parent_hash = PARENT_HASH;

    executor
        .apply_pre_execution_changes()
        .expect("pre-execution");

    let recovered = Recovered::new_unchecked(tx, ARBOS_ADDRESS);
    let result = executor
        .execute_transaction_without_commit(recovered)
        .expect("execute internal tx");
    executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");

    let eip2935 = read_slot(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        U256::from(EIP2935_SLOT_DECIMAL),
    );
    assert_eq!(
        eip2935,
        U256::from_be_bytes(PARENT_HASH.0),
        "EIP-2935 slot 0x{EIP2935_SLOT_DECIMAL:x} must hold the parent hash 0x{PARENT_HASH}, got {eip2935:x}",
    );

    let arbos_l1 = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
    );
    assert_eq!(
        arbos_l1,
        U256::from(NEW_L1_BLOCK_NUMBER),
        "ArbOS L1 block slot must update to 0x{NEW_L1_BLOCK_NUMBER:x}, got {arbos_l1:x}",
    );

    let arbos_prev_parent = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_PREV_PARENT_SLOT.0),
    );
    assert_eq!(
        arbos_prev_parent,
        U256::from_be_bytes(PARENT_HASH.0),
        "ArbOS legacy parent-hash slot must update to {PARENT_HASH}, got {arbos_prev_parent:x}",
    );

    let arbos_l2_basefee = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L2_BASE_FEE_SLOT.0),
    );
    assert_eq!(
        arbos_l2_basefee,
        U256::from(ARBOS_POST_L2_BASE_FEE),
        "ArbOS L2 base fee slot must update to 0x{ARBOS_POST_L2_BASE_FEE:x}, got {arbos_l2_basefee:x}",
    );

    // The writes above are visible in the cache. Re-execution reads the
    // committed bundle (plain) state and roots from it: assert the same
    // writes survive `merge_transitions(PlainState)`.
    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::PlainState);

    let eip2935_bundle = read_bundle_slot(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        U256::from(EIP2935_SLOT_DECIMAL),
    );
    assert_eq!(
        eip2935_bundle,
        U256::from_be_bytes(PARENT_HASH.0),
        "EIP-2935 write must persist to the bundle (got {eip2935_bundle:x})",
    );

    let arbos_l1_bundle = read_bundle_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
    );
    assert_eq!(
        arbos_l1_bundle,
        U256::from(NEW_L1_BLOCK_NUMBER),
        "ArbOS L1 block slot write must persist to the bundle (got {arbos_l1_bundle:x})",
    );

    let arbos_basefee_bundle = read_bundle_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L2_BASE_FEE_SLOT.0),
    );
    assert_eq!(
        arbos_basefee_bundle,
        U256::from(ARBOS_POST_L2_BASE_FEE),
        "ArbOS L2 base fee write must persist to the bundle (got {arbos_basefee_bundle:x})",
    );
}
