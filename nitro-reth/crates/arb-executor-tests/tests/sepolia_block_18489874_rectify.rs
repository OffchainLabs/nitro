//! Replays the canonical EIP-1559 `rectifyChainOwner(address)` call to
//! ArbOwnerPublic at Sepolia block 18,489,874 against a harness seeded with
//! the captured block-start state. The chain-owner set carries a stale
//! `byAddress` entry (a residue of the pre-v11 remove path), so the canonical
//! block re-maps the owner and succeeds with a `ChainOwnerRectified` log.

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
use alloy_primitives::{
    address, b256, hex, keccak256, Address, Bytes, Signature, TxKind, B256, U256,
};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at, ARBOS_STATE_ADDRESS};
use arb_test_utils::{ArbosHarness, EmptyDb};
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::State, primitives::hardfork::SpecId};
use serde::Deserialize;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 11;

const BLOCK_NUMBER: u64 = 18_489_874;
const BLOCK_TIMESTAMP: u64 = 0x65e0_0800;
const L1_BLOCK_NUMBER: u64 = 0x0052_2c08;
const TIME_PASSED: u64 = 1;
const PARENT_HASH: B256 = b256!("d39a25a34525781aae398e0cb83a1728f6249efd082e04a2f9b05945cba04d46");
const BLOCK_BASE_FEE: u128 = 0x5f5_e100;

const SENDER: Address = address!("69e4fb7b4d0df0d338a07e53a49a840cdc70efef");
const ARBOWNERPUBLIC: Address = address!("000000000000000000000000000000000000006b");
const NETWORK_FEE_ACCOUNT: Address = address!("71b61c2e250afa05dfc36304d6c91501be0965d8");

const TX_NONCE: u64 = 0x19;
const TX_GAS_LIMIT: u64 = 0x1_6e91;
const TX_MAX_FEE: u128 = 0xbebc_2000;
const TX_MAX_PRIO: u128 = 0xb2d0_5e00;

const CANON_LOG_COUNT: usize = 1;

// chain-owner AddressSet slots on the ArbOS state account (subspace keccak(4)).
const CHAIN_OWNERS_SIZE_SLOT: &str =
    "41e0d7d38ffe0727248ee6ed6ea1250b08279ad004e3ab07b7ffe78352d8c400";
const CHAIN_OWNERS_BACKING_2_SLOT: &str =
    "41e0d7d38ffe0727248ee6ed6ea1250b08279ad004e3ab07b7ffe78352d8c402";
const CHAIN_OWNERS_BYADDR_TARGET_SLOT: &str =
    "ff922cd4c96a7d831e52b53a42855d4ff5e131718cca2d7063af2a0269f6a3d8";

const TX_INPUT_HEX: &str =
    include_str!("../../arb-spec-tests/fixtures/regression/sepolia_18489874/tx1_input.hex");
const PRESTATE_JSON: &str = include_str!(
    "../../arb-spec-tests/fixtures/regression/sepolia_18489874/block_start_prestate.json"
);

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

fn seed_prestate(state: &mut State<EmptyDb>, snapshot: &BTreeMap<String, AccountSnapshot>) {
    use revm::database::states::bundle_state::BundleRetention;

    for (addr_str, acct) in snapshot {
        let addr = parse_address(addr_str);
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
            write_storage_at(state, addr, parse_hex_u256(slot), parse_hex_u256(value))
                .expect("seed storage");
        }
    }

    state.merge_transitions(BundleRetention::Reverts);
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

fn build_tx() -> ArbTransactionSigned {
    let tx = TxEip1559 {
        chain_id: CHAIN_ID,
        nonce: TX_NONCE,
        gas_limit: TX_GAS_LIMIT,
        max_fee_per_gas: TX_MAX_FEE,
        max_priority_fee_per_gas: TX_MAX_PRIO,
        to: TxKind::Call(ARBOWNERPUBLIC),
        value: U256::ZERO,
        access_list: Default::default(),
        input: Bytes::from(parse_hex_bytes(TX_INPUT_HEX)),
    };
    let sig = Signature::new(
        U256::from_be_bytes(
            b256!("b2d3e8217acbe75609654ea40831042c614654d4f63ffa3fec3bf6e7ac4ae8bc").0,
        ),
        U256::from_be_bytes(
            b256!("57ad72f14e414d825cabcdafb28337e178273d0087694734cb9c7757da24d278").0,
        ),
        false,
    );
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip1559(tx.into_signed(sig)))
}

#[test]
fn rectify_chain_owner_matches_canonical() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .with_network_fee_account(NETWORK_FEE_ACCOUNT)
        .initialize();

    let prestate: BTreeMap<String, AccountSnapshot> =
        serde_json::from_str(PRESTATE_JSON).expect("parse block-start prestate JSON");
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
    env.block_env.basefee = BLOCK_BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);
    env.block_env.beneficiary = address!("a4b000000000000000000073657175656e636572");

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        L1_BLOCK_NUMBER,
        BLOCK_NUMBER,
        false,
    );
    let staged_ctx = Arc::new(arb_context::ArbPrecompileCtx::with_block(Arc::new(
        block_ctx,
    )));
    evm_factory.stage_ctx(staged_ctx);
    let evm = evm_factory.create_evm(harness.state(), env);
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(2),
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
    executor.arb_ctx.l1_block_number = L1_BLOCK_NUMBER;
    executor.arb_ctx.parent_hash = PARENT_HASH;

    executor
        .apply_pre_execution_changes()
        .expect("pre-execution");

    let start_block_calldata =
        encode_start_block(U256::ZERO, L1_BLOCK_NUMBER, BLOCK_NUMBER, TIME_PASSED);
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
        .expect("execute rectify tx");

    let logs = result.result.result.logs().to_vec();
    let status = result.result.result.is_success();
    let gas_used = result.result.result.gas_used();
    executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");

    let size = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        parse_hex_u256(CHAIN_OWNERS_SIZE_SLOT),
    );
    let backing2 = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        parse_hex_u256(CHAIN_OWNERS_BACKING_2_SLOT),
    );
    let byaddr = read_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        parse_hex_u256(CHAIN_OWNERS_BYADDR_TARGET_SLOT),
    );

    eprintln!(
        "status={status} gas_used={gas_used} logs={} size={size} backing[2]={backing2:x} byAddr[target]={byaddr}",
        logs.len()
    );

    assert!(status, "rectify tx must succeed (canonical status 0x1)");
    assert_eq!(
        logs.len(),
        CANON_LOG_COUNT,
        "log count must match canonical"
    );
    assert_eq!(
        logs[0].topics()[0],
        keccak256("ChainOwnerRectified(address)"),
        "log topic0 must be ChainOwnerRectified",
    );
    assert_eq!(
        size,
        U256::from(2u64),
        "chain-owner set size must grow to 2"
    );
    assert_eq!(
        backing2,
        parse_hex_u256("71b61c2e250afa05dfc36304d6c91501be0965d8"),
        "rectified owner must be appended at backing slot 2",
    );
    assert_eq!(
        byaddr,
        U256::from(2u64),
        "byAddress[target] must be remapped to slot 2",
    );
}
