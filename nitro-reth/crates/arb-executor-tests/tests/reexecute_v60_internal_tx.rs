//! Re-execution-path parity for the v60 StartBlock InternalTx.
//!
//! Unlike the executor-driven tests, this drives the *exact* entrypoint the
//! `re-execute` command uses — `ConfigureEvm::executor_for_block` +
//! `BlockExecutor::execute_block` over a `RecoveredBlock` — so the block
//! context (ArbOS version, L1/L2 block number, base fee) is derived from the
//! block header and `context_for_block`, not set by hand. It then asserts the
//! StartBlock writes land in the committed bundle (plain) state, which is what
//! re-execution reads back and roots from.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::{Block, BlockBody, Header};
use alloy_primitives::{address, b256, Address, Bytes, Signature, B256, B64, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{
    arbos_versions::{HISTORY_STORAGE_ADDRESS, HISTORY_STORAGE_CODE_ARBITRUM},
    signed_tx::ArbTypedTransaction,
    ArbTransactionSigned,
};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at, ARBOS_STATE_ADDRESS};
use arb_test_utils::ArbosHarness;
use arbos::{header::compute_arbos_mixhash, internal_tx::encode_start_block};
use reth_chainspec::ChainSpec;
use reth_evm::{block::BlockExecutor, ConfigureEvm};
use reth_primitives_traits::{RecoveredBlock, SealedBlock};

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const BLOCK_NUMBER: u64 = 269_589_709;
const BLOCK_TIMESTAMP: u64 = 0x6a0c9714;
const HEADER_BASE_FEE: u64 = 0x1315410;
const SEND_COUNT: u64 = 0x1c80d;
const PARENT_HASH: B256 = b256!("19b751952803e4efe00151b1e888b0634a5e1b2ae3f88e6911e6ca0c1424180e");

const NEW_L1_BLOCK_NUMBER: u64 = 0xa606cf;
const OLD_L1_BLOCK_NUMBER: u64 = 0xa606cd;
const TIME_PASSED: u64 = 1;

const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

const ARBOS_L1_BLOCK_SLOT: B256 =
    b256!("3c79da47f96b0f39664f73c0a1f350580be90742947dddfa21ba64d578dfe600");
const ARBOS_PRESTATE_OLD_L1: u64 = 0xa606cd;
const EIP2935_SLOT_DECIMAL: u64 = 269_628;

fn zero_sig() -> Signature {
    Signature::new(U256::ZERO, U256::ZERO, false)
}

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

fn start_block_tx() -> ArbTransactionSigned {
    let calldata = encode_start_block(U256::ZERO, NEW_L1_BLOCK_NUMBER, BLOCK_NUMBER, TIME_PASSED);
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: calldata.into(),
        }),
        zero_sig(),
    )
}

#[test]
fn reexecute_v60_start_block_persists_writes_to_bundle() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    set_account_code(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        HISTORY_STORAGE_CODE_ARBITRUM.clone(),
    );
    set_account_nonce(harness.state(), HISTORY_STORAGE_ADDRESS, 1);
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
        U256::from(ARBOS_PRESTATE_OLD_L1),
    )
    .expect("seed old l1 block");
    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    // A header shaped like the stored v60 block: mix_hash encodes the ArbOS
    // version + L1 block number, extra_data is the 32-byte send root, nonce
    // carries delayed_messages_read.
    let header = Header {
        parent_hash: PARENT_HASH,
        ommers_hash: alloy_consensus::constants::EMPTY_OMMER_ROOT_HASH,
        beneficiary: address!("a4b000000000000000000073657175656e636572"),
        state_root: B256::ZERO,
        transactions_root: B256::ZERO,
        receipts_root: B256::ZERO,
        withdrawals_root: None,
        logs_bloom: Default::default(),
        difficulty: U256::from(1),
        number: BLOCK_NUMBER,
        gas_limit: 1_125_899_906_842_624,
        gas_used: 0,
        timestamp: BLOCK_TIMESTAMP,
        mix_hash: compute_arbos_mixhash(SEND_COUNT, OLD_L1_BLOCK_NUMBER, ARBOS_VERSION, false),
        nonce: B64::from(1u64.to_be_bytes()),
        base_fee_per_gas: Some(HEADER_BASE_FEE),
        extra_data: Bytes::from(vec![0u8; 32]),
        parent_beacon_block_root: None,
        blob_gas_used: None,
        excess_blob_gas: None,
        requests_hash: None,
    };

    let block = Block::<ArbTransactionSigned> {
        header,
        body: BlockBody {
            transactions: vec![start_block_tx()],
            ommers: Default::default(),
            withdrawals: None,
        },
    };
    let sealed = SealedBlock::seal_slow(block);
    // Recover senders the way the provider's `recovered_block` does, rather
    // than forcing them, so the InternalTx sender is resolved by the signer.
    let recovered = RecoveredBlock::try_recover_sealed(sealed).expect("recover senders");
    assert_eq!(
        recovered.senders()[0],
        ARBOS_ADDRESS,
        "InternalTx sender must recover to the ArbOS address",
    );

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    {
        let executor = cfg
            .executor_for_block(harness.state(), recovered.sealed_block())
            .expect("executor_for_block");
        executor
            .execute_block(recovered.transactions_recovered())
            .expect("execute_block");
    }

    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::PlainState);

    let eip2935 = read_bundle_slot(
        harness.state(),
        HISTORY_STORAGE_ADDRESS,
        U256::from(EIP2935_SLOT_DECIMAL),
    );
    assert_eq!(
        eip2935,
        U256::from_be_bytes(PARENT_HASH.0),
        "re-execute must persist the EIP-2935 parent-hash write to the bundle (got {eip2935:x})",
    );

    let arbos_l1 = read_bundle_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
    );
    assert_eq!(
        arbos_l1,
        U256::from(NEW_L1_BLOCK_NUMBER),
        "re-execute must persist the ArbOS L1 block-number write to the bundle (got {arbos_l1:x})",
    );
}
