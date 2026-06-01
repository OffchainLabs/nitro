//! Replays the canonical user EIP-1559 transaction at Sepolia block
//! 269,589,709 (an oracle price-update call) against a harness seeded with
//! the captured pre-state, and asserts every canonical post-state change —
//! the sender's gas debit, the oracle contracts' storage writes, the credit
//! to the network-fee account, and every multi-gas backlog slot the L2
//! pricing model touches under ArbOS v60.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::{transaction::Recovered, EthereumTxEnvelope, SignableTransaction, TxEip1559};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, b256, hex, Address, Bytes, Signature, TxKind, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{
    arbos_versions::{HISTORY_STORAGE_ADDRESS, HISTORY_STORAGE_CODE_ARBITRUM},
    signed_tx::ArbTypedTransaction,
    ArbTransactionSigned,
};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at, ARBOS_STATE_ADDRESS};
use arb_test_utils::{ArbosHarness, EmptyDb};
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::State, primitives::hardfork::SpecId};
use serde::Deserialize;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;

const BLOCK_NUMBER: u64 = 269_589_709;
const BLOCK_TIMESTAMP: u64 = 0x6a0c9714;
const OLD_L1_BLOCK_NUMBER: u64 = 0xa606cd;
const NEW_L1_BLOCK_NUMBER: u64 = 0xa606cf;
const TIME_PASSED: u64 = 1;
const PARENT_HASH: B256 = b256!("19b751952803e4efe00151b1e888b0634a5e1b2ae3f88e6911e6ca0c1424180e");
// The canonical block header's baseFeePerGas (= the effective gas price the
// sender pays after the v60 tip drop).
const BLOCK_BASE_FEE: u128 = 0x1315410;
const L1_BLOCK_NUMBER: u64 = 0xa606cf;

const SENDER: Address = address!("0fc44dd1ac57e76d22e1dcc5a9b059ee69fb4fc3");
const TARGET: Address = address!("dc1480c15af7c58c804fae701ec33f01c5750d73");

const TX_NONCE: u64 = 46_507;
const TX_GAS_LIMIT: u64 = 0xb011c;
const TX_MAX_FEE: u128 = 0x2a234e1;
const TX_MAX_PRIO: u128 = 1;
const TX_VALUE: u128 = 0xa;

const NETWORK_FEE_ACCOUNT: Address = address!("a4b00000000000000000000000000000000000f6");

const CANON_GAS_USED: u64 = 468_125;
const CANON_LOG_COUNT: usize = 4;

const TX_INPUT_HEX: &str =
    include_str!("../../arb-spec-tests/fixtures/regression/sepolia_269589709/tx1_input.hex");
const PRESTATE_JSON: &str = include_str!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589709/block_start_prestate.json"
);
const POSTSTATE_JSON: &str =
    include_str!("../../arb-spec-tests/fixtures/regression/sepolia_269589709/tx1_post.json");

// ---------------------------------------------------------------------------
// Fixture model
// ---------------------------------------------------------------------------

/// One account's snapshot in the JSON fixture. Each field is optional so the
/// captured prestate can omit defaults (e.g. zero-balance accounts).
#[derive(Debug, Deserialize)]
struct AccountSnapshot {
    #[serde(default)]
    balance: Option<String>,
    #[serde(default)]
    nonce: Option<u64>,
    #[serde(default)]
    code: Option<String>,
    #[serde(default)]
    storage: BTreeMap<String, String>,
}

fn parse_hex_u256(s: &str) -> U256 {
    let s = s.strip_prefix("0x").unwrap_or(s);
    if s.is_empty() {
        return U256::ZERO;
    }
    U256::from_str_radix(s, 16).unwrap_or_else(|_| panic!("parse U256 hex `{s}`"))
}

fn parse_hex_bytes(s: &str) -> Vec<u8> {
    hex::decode(s.trim().trim_start_matches("0x")).expect("decode hex bytes")
}

fn parse_address(s: &str) -> Address {
    Address::from_slice(&parse_hex_bytes(s))
}

/// Seeds every account in the JSON snapshot into the harness's State, then
/// stages the writes into the bundle so subsequent reads observe them.
fn seed_prestate(state: &mut State<EmptyDb>, snapshot: &BTreeMap<String, AccountSnapshot>) {
    use revm::database::states::bundle_state::BundleRetention;

    for (addr_str, acct) in snapshot {
        let addr = parse_address(addr_str);

        // `fund_account` resets the AccountInfo, so seed balance first; code
        // and nonce live in the same AccountInfo and must be applied after.
        if let Some(balance) = acct.balance.as_deref() {
            let v = parse_hex_u256(balance);
            if !v.is_zero() {
                arb_executor_tests::helpers::fund_account(state, addr, v);
            }
        }
        if let Some(code) = acct.code.as_deref() {
            let bytes = parse_hex_bytes(code);
            if !bytes.is_empty() {
                set_account_code(state, addr, Bytes::from(bytes));
            }
        }
        if let Some(nonce) = acct.nonce {
            if nonce > 0 {
                set_account_nonce(state, addr, nonce);
            }
        }
        for (slot, value) in &acct.storage {
            let slot_u = parse_hex_u256(slot);
            let value_u = parse_hex_u256(value);
            write_storage_at(state, addr, slot_u, value_u).expect("seed storage");
        }
    }

    state.merge_transitions(BundleRetention::Reverts);
}

fn read_balance(state: &mut State<EmptyDb>, addr: Address) -> U256 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|c| c.account.as_ref())
        .map(|a| a.info.balance)
        .unwrap_or(U256::ZERO)
}

fn read_nonce(state: &mut State<EmptyDb>, addr: Address) -> u64 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|c| c.account.as_ref())
        .map(|a| a.info.nonce)
        .unwrap_or(0)
}

fn read_slot(state: &mut State<EmptyDb>, addr: Address, slot: U256) -> U256 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|c| c.account.as_ref())
        .and_then(|a| a.storage.get(&slot).copied())
        .unwrap_or(U256::ZERO)
}

#[derive(Debug, Default)]
struct StateMismatchReport {
    issues: Vec<String>,
}

impl StateMismatchReport {
    fn check<T: PartialEq + std::fmt::Debug>(&mut self, label: &str, got: T, expected: T) {
        if got != expected {
            self.issues
                .push(format!("{label}: got {got:?}, expected {expected:?}"));
        }
    }

    fn ok(&self) -> bool {
        self.issues.is_empty()
    }
}

// ---------------------------------------------------------------------------
// Transaction reconstruction
// ---------------------------------------------------------------------------

fn build_tx() -> ArbTransactionSigned {
    let input = parse_hex_bytes(TX_INPUT_HEX);
    let tx = TxEip1559 {
        chain_id: CHAIN_ID,
        nonce: TX_NONCE,
        gas_limit: TX_GAS_LIMIT,
        max_fee_per_gas: TX_MAX_FEE,
        max_priority_fee_per_gas: TX_MAX_PRIO,
        to: TxKind::Call(TARGET),
        value: U256::from(TX_VALUE),
        access_list: Default::default(),
        input: Bytes::from(input),
    };
    let sig = Signature::new(
        U256::from_be_bytes(
            b256!("051a1fff201db6e1a14774fe414c59b4180d8e0f73ef2c9b7eb9c66b9c1f20be").0,
        ),
        U256::from_be_bytes(
            b256!("1f47bd5f9cbee859a6c252f1aad89deccccb091cb5097d50d0df24feb39438e5").0,
        ),
        true,
    );
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip1559(tx.into_signed(sig)))
}

// ---------------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------------

#[test]
fn v60_user_call_matches_canonical_post_state() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .with_network_fee_account(NETWORK_FEE_ACCOUNT)
        .initialize();

    // Seed the EIP-2935 history contract: not present in the prestate
    // capture for tx0 (it tracks reads only, not the code/nonce that the v40
    // upgrade installed), so set them explicitly.
    arb_storage::set_account_code(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        HISTORY_STORAGE_CODE_ARBITRUM.clone(),
    );
    arb_storage::set_account_nonce(harness.state(), HISTORY_STORAGE_ADDRESS, 1);

    let prestate: BTreeMap<String, AccountSnapshot> =
        serde_json::from_str(PRESTATE_JSON).expect("parse block-start prestate JSON");
    seed_prestate(harness.state(), &prestate);

    let poststate: BTreeMap<String, AccountSnapshot> =
        serde_json::from_str(POSTSTATE_JSON).expect("parse tx1 poststate JSON");

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
    env.block_env.basefee = BLOCK_BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);
    // The canonical block was sealed by the sequencer; arbreth derives
    // arb_ctx.coinbase from block_env.beneficiary in apply_pre_execution.
    env.block_env.beneficiary = address!("a4b000000000000000000073657175656e636572");

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
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(BLOCK_BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = OLD_L1_BLOCK_NUMBER;
    executor.arb_ctx.parent_hash = PARENT_HASH;

    executor
        .apply_pre_execution_changes()
        .expect("pre-execution");

    // Replay the StartBlock InternalTx first. It loads the L1 / L2 pricing
    // context into the executor and pumps the multi-gas backlog drain, both
    // of which the user tx depends on for fee accounting.
    let start_block_calldata =
        encode_start_block(U256::ZERO, NEW_L1_BLOCK_NUMBER, BLOCK_NUMBER, TIME_PASSED);
    let internal_tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: start_block_calldata.into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let internal_recovered = Recovered::new_unchecked(
        internal_tx,
        address!("00000000000000000000000000000000000A4B05"),
    );
    let internal_result = executor
        .execute_transaction_without_commit(internal_recovered)
        .expect("execute internal tx");
    executor
        .commit_transaction(internal_result)
        .expect("commit internal tx");

    let tx = build_tx();
    let recovered: Recovered<ArbTransactionSigned> = arb_executor_tests::helpers::recover(tx);
    assert_eq!(
        recovered.signer(),
        SENDER,
        "recovered sender must be canonical"
    );

    let result = executor
        .execute_transaction_without_commit(recovered)
        .expect("execute tx");
    assert!(
        result.result.result.is_success(),
        "user tx must succeed, got {:?}",
        result.result.result,
    );
    assert_eq!(
        result.result.result.gas_used(),
        CANON_GAS_USED,
        "gas_used must match canonical",
    );
    assert_eq!(
        result.result.result.logs().len(),
        CANON_LOG_COUNT,
        "log count must match canonical",
    );
    executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");

    // Walk every canonical post entry and assert byte-exact parity.
    let mut report = StateMismatchReport::default();
    for (addr_str, expected) in &poststate {
        let addr = parse_address(addr_str);
        if let Some(bal) = expected.balance.as_deref() {
            report.check(
                &format!("{addr_str} balance"),
                read_balance(harness.state(), addr),
                parse_hex_u256(bal),
            );
        }
        if let Some(nonce) = expected.nonce {
            report.check(
                &format!("{addr_str} nonce"),
                read_nonce(harness.state(), addr),
                nonce,
            );
        }
        for (slot, value) in &expected.storage {
            report.check(
                &format!("{addr_str} storage[{slot}]"),
                read_slot(harness.state(), addr, parse_hex_u256(slot)),
                parse_hex_u256(value),
            );
        }
    }

    // Surface the legacy-sequencer balance change as a focused signal: the
    // canonical block does not modify it; any delta indicates the user-tx
    // payment is being routed to the ASCII-"sequencer" coinbase address.
    let legacy_sequencer: Address = address!("a4b000000000000000000073657175656e636572");
    let leg_pre = prestate
        .get("0xa4b000000000000000000073657175656e636572")
        .and_then(|a| a.balance.as_deref())
        .map(parse_hex_u256)
        .unwrap_or(U256::ZERO);
    let leg_post = read_balance(harness.state(), legacy_sequencer);
    if leg_pre != leg_post {
        report.issues.push(format!(
            "legacy sequencer balance changed by {} wei (pre {leg_pre}, post {leg_post}); canonical leaves it unchanged",
            leg_post.saturating_sub(leg_pre),
        ));
    }

    // ArbOS-state writes the L2 pricing model performs at end-of-tx. These
    // are the eight slots the canonical block updates; treat them as a
    // focused signal in the failure message.
    let arbos_state_slots: &[(&str, &str)] = &[
        (
            "15feb681c0d80d6dd4522f2158aecf87aabe907423539f227812e8c90210b402",
            "72316",
        ),
        (
            "17650a1d3a10f7c15c10bf5564cdb9520ed027a69d8e5c570fc89bdfeb692502",
            "72316",
        ),
        (
            "3f7acd5dfea0f400b20ee8227e84b8c9f0440358ededd3c2043024a948526402",
            "72316",
        ),
        (
            "8ecee7fb465f53613bd7b42d2402ed3df2f6042d126f420232661aac17f0f202",
            "72316",
        ),
        (
            "a9f6f085d78d1d37c5819e5c16c9e03198bd14e08cd1f6f8191bc6207b9e9706",
            "1265816",
        ),
        (
            "a9f6f085d78d1d37c5819e5c16c9e03198bd14e08cd1f6f8191bc6207b9e970b",
            "38a5c1ca94e",
        ),
        (
            "b030e80be1d37b800ea9902aab7481a1eea35835f84698282506b483b7472102",
            "72316",
        ),
        (
            "c2af9c3b011102e7665469ff70f63acdc0d1477bd7c96edb90ab98e73d499f02",
            "72316",
        ),
    ];
    for (slot_hex, expected_hex) in arbos_state_slots {
        let slot = parse_hex_u256(slot_hex);
        let got = read_slot(harness.state(), ARBOS_STATE_ADDRESS, slot);
        let expected = parse_hex_u256(expected_hex);
        if got != expected {
            report.issues.push(format!(
                "arbos_state[{slot_hex}]: got {got:x}, expected {expected:x}"
            ));
        }
    }

    assert!(
        report.ok(),
        "user-call post-state parity failed ({} issues):\n  - {}",
        report.issues.len(),
        report.issues.join("\n  - "),
    );
}
