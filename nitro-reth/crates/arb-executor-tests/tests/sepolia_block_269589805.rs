//! Replays Sepolia block 269,589,805 (v60) from captured canonical pre-state
//! through the multi-gas inspector path the block-derivation pipeline uses:
//! the StartBlock InternalTx followed by four user transactions. Asserts each
//! sender's net balance matches canonical. The block charges every tx at the
//! header base fee (20,052,000); canonical nets each to the floor (20,000,000)
//! via the v60 multi-gas refund. The fourth (last) transaction is the one whose
//! refund must reconcile correctly under per-opcode dimensional attribution.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::transaction::Recovered;
use alloy_eips::Decodable2718;
use alloy_primitives::{address, b256, hex, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::{
    config::ArbEvmConfig,
    multi_gas::{MultiGasInspector, MultiGasSink},
};
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at};
use arb_test_utils::{ArbosHarness, EmptyDb};
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::State, primitives::hardfork::SpecId};
use serde::Deserialize;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const BLOCK_NUMBER: u64 = 269_589_805;
const BLOCK_TIMESTAMP: u64 = 0x6a0c972f;
const HEADER_BASE_FEE: u128 = 0x131f820; // 20,052,000
const L1_BLOCK_NUMBER: u64 = 0xa606d0;
const PARENT_HASH: B256 = b256!("f73c0acef7ef709a61a5622012af25d75ca1c46de05b7520e56ea5f4679cc31c");
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");
const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

// (sender, expected net balance) per user tx, in execution order.
const SENDERS: [(Address, U256); 4] = [
    (
        address!("359c13f68c116b448e16a1f7c77ea70aeca8fe55"),
        U256::from_limbs([0x455dcf36cc45ca20, 0, 0, 0]),
    ),
    (
        address!("0b865d9b9ba8b6ffb984295540ba998d3b4b7d74"),
        U256::from_limbs([0x0d73ecf212b34580, 0, 0, 0]),
    ),
    (
        address!("f6fd5fca4bd769ba495b29b98dba5f2ecf4ceed3"),
        U256::from_limbs([0x153f05e096e873f0, 0, 0, 0]),
    ),
    (
        address!("43129021f6285bd9448af1f2d674bdc8103b1384"),
        U256::from_limbs([0x43cc1c1295d542a0, 0, 0, 0]),
    ),
];

const PRESTATE_JSON: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589805/block_start_prestate.json"
));
const TX1: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589805/tx1_raw.hex"
));
const TX2: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589805/tx2_raw.hex"
));
const TX3: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589805/tx3_raw.hex"
));
const TX4: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589805/tx4_raw.hex"
));

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

fn hu(s: &str) -> U256 {
    let s = s.strip_prefix("0x").unwrap_or(s);
    if s.is_empty() {
        U256::ZERO
    } else {
        U256::from_str_radix(s, 16).unwrap()
    }
}
fn hb(s: &str) -> Vec<u8> {
    hex::decode(s.trim().trim_start_matches("0x")).unwrap()
}
fn addr(s: &str) -> Address {
    Address::from_slice(&hb(s))
}

fn seed_prestate(state: &mut State<EmptyDb>, snap: &BTreeMap<String, AccountSnapshot>) {
    use revm::database::states::bundle_state::BundleRetention;
    for (a, acct) in snap {
        let ad = addr(a);
        if let Some(b) = acct.balance.as_deref() {
            let v = hu(b);
            if !v.is_zero() {
                arb_executor_tests::helpers::fund_account(state, ad, v);
            }
        }
        if let Some(c) = acct.code.as_deref() {
            let by = hb(c);
            if !by.is_empty() {
                set_account_code(state, ad, Bytes::from(by));
            }
        }
        if let Some(n) = acct.nonce {
            if n > 0 {
                set_account_nonce(state, ad, n);
            }
        }
        for (slot, val) in &acct.storage {
            write_storage_at(state, ad, hu(slot), hu(val)).unwrap();
        }
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn read_balance(state: &mut State<EmptyDb>, a: Address) -> U256 {
    state
        .cache
        .accounts
        .get(&a)
        .and_then(|c| c.account.as_ref())
        .map(|x| x.info.balance)
        .unwrap_or(U256::ZERO)
}

fn decode(raw: &str) -> Recovered<ArbTransactionSigned> {
    let bytes = hb(raw);
    let tx = ArbTransactionSigned::decode_2718(&mut bytes.as_slice()).expect("decode 2718");
    arb_executor_tests::helpers::recover(tx)
}

#[test]
fn sepolia_269589805_sender_net_charges_match_canonical() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    let prestate: BTreeMap<String, AccountSnapshot> = serde_json::from_str(PRESTATE_JSON).unwrap();
    seed_prestate(harness.state(), &prestate);

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
    env.block_env.basefee = HEADER_BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);
    env.block_env.beneficiary = SEQUENCER;

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        L1_BLOCK_NUMBER,
        false,
    );
    evm_factory.stage_ctx(Arc::new(arb_context::ArbPrecompileCtx::with_block(
        Arc::new(block_ctx),
    )));

    // Mirror the block-derivation pipeline: install the multi-gas inspector so
    // each tx's per-opcode dimensional attribution drives the v60 refund.
    let sink = MultiGasSink::default();
    let evm = evm_factory.create_evm_with_inspector(
        harness.state(),
        env,
        MultiGasInspector::with_sink(sink.clone()),
    );
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(5),
        parent_hash: PARENT_HASH,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.set_multi_gas_sink(sink);
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(HEADER_BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = L1_BLOCK_NUMBER;
    executor.apply_pre_execution_changes().expect("pre-exec");

    // StartBlock InternalTx (drains base_fee_wei to the next block's value).
    let sb = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: encode_start_block(U256::ZERO, L1_BLOCK_NUMBER, BLOCK_NUMBER, 1).into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let r = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(sb, ARBOS_ADDRESS))
        .expect("startblock");
    executor.commit_transaction(r).expect("commit sb");

    for (i, raw) in [TX1, TX2, TX3, TX4].iter().enumerate() {
        let recovered = decode(raw);
        assert_eq!(
            recovered.signer(),
            SENDERS[i].0,
            "tx{} sender recovery",
            i + 1
        );
        let r = executor
            .execute_transaction_without_commit(recovered)
            .unwrap_or_else(|e| panic!("tx{} exec: {e:?}", i + 1));
        executor.commit_transaction(r).expect("commit user");
    }
    let _ = executor.finish().expect("finish");

    let mut mismatches = Vec::new();
    for (i, (sender, want)) in SENDERS.iter().enumerate() {
        let got = read_balance(harness.state(), *sender);
        if got != *want {
            mismatches.push(format!(
                "tx{}: sender {sender} got {got:x} want {want:x} (delta {} wei)",
                i + 1,
                want.abs_diff(got),
            ));
        }
    }
    assert!(
        mismatches.is_empty(),
        "sender net charges diverge from canonical:\n{}",
        mismatches.join("\n"),
    );
}
