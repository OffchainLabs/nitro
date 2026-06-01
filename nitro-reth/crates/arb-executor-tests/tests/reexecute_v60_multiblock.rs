//! Multi-block re-execution parity for the v60 StartBlock InternalTx.
//!
//! Mirrors the `re-execute` chunk loop: one `State` reused across several
//! blocks, `executor_for_block` + `execute_block` per block, with
//! `merge_transitions(PlainState)` between. Every block's StartBlock
//! InternalTx re-touches the ArbOS state account (new L1 block number) and
//! writes a fresh EIP-2935 history entry. The test asserts each block's
//! writes land in the committed bundle — the across-block accumulation that
//! single-block tests cannot exercise.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::{Block, BlockBody, Header};
use alloy_primitives::{address, Address, Bytes, Signature, B256, B64, U256};
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
use revm::database::states::bundle_state::BundleRetention;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const HEADER_BASE_FEE: u64 = 0x1315410;
const SEND_COUNT: u64 = 0x1c80d;
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");

// ArbOS blockhashes subspace: L1 block number lives at this slot.
const ARBOS_L1_BLOCK_SLOT: B256 =
    alloy_primitives::b256!("3c79da47f96b0f39664f73c0a1f350580be90742947dddfa21ba64d578dfe600");

const HISTORY_SERVE_WINDOW: u64 = 393_168;
const FIRST_BLOCK: u64 = 269_589_702;
const FIRST_L1: u64 = 0xa606c0;
const NUM_BLOCKS: u64 = 4;

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

fn start_block_tx(l1_block: u64, l2_block: u64) -> ArbTransactionSigned {
    let calldata = encode_start_block(U256::ZERO, l1_block, l2_block, 1);
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: calldata.into(),
        }),
        zero_sig(),
    )
}

#[allow(clippy::too_many_arguments)]
fn block_for(l2_block: u64, l1_block: u64, parent_hash: B256) -> Block<ArbTransactionSigned> {
    let header = Header {
        parent_hash,
        ommers_hash: alloy_consensus::constants::EMPTY_OMMER_ROOT_HASH,
        beneficiary: SEQUENCER,
        state_root: B256::ZERO,
        transactions_root: B256::ZERO,
        receipts_root: B256::ZERO,
        withdrawals_root: None,
        logs_bloom: Default::default(),
        difficulty: U256::from(1),
        number: l2_block,
        gas_limit: 1_125_899_906_842_624,
        gas_used: 0,
        timestamp: 0x6a0c9714 + l2_block,
        // The StartBlock tx advances the recorded L1 height to `l1_block`; the
        // header (next-block snapshot) carries it.
        mix_hash: compute_arbos_mixhash(SEND_COUNT, l1_block, ARBOS_VERSION, false),
        nonce: B64::from(1u64.to_be_bytes()),
        base_fee_per_gas: Some(HEADER_BASE_FEE),
        extra_data: Bytes::from(vec![0u8; 32]),
        parent_beacon_block_root: None,
        blob_gas_used: None,
        excess_blob_gas: None,
        requests_hash: None,
    };
    Block {
        header,
        body: BlockBody {
            transactions: vec![start_block_tx(l1_block, l2_block)],
            ommers: Default::default(),
            withdrawals: None,
        },
    }
}

#[test]
fn reexecute_v60_multiblock_persists_each_start_block() {
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
    // Recorded L1 height starts just below the first block's value.
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
        U256::from(FIRST_L1 - 2),
    )
    .expect("seed old l1 block");
    harness
        .state()
        .merge_transitions(BundleRetention::PlainState);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut parent_hash = B256::repeat_byte(0x11);
    let mut expected_eip2935: Vec<(u64, B256)> = Vec::new();
    let mut last_l1 = 0u64;

    for i in 0..NUM_BLOCKS {
        let l2_block = FIRST_BLOCK + i;
        let l1_block = FIRST_L1 + 2 * i;
        last_l1 = l1_block;

        let block = block_for(l2_block, l1_block, parent_hash);
        let sealed = SealedBlock::seal_slow(block);
        let block_hash = sealed.hash();
        let recovered = RecoveredBlock::try_recover_sealed(sealed).expect("recover");

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
            .merge_transitions(BundleRetention::PlainState);

        // EIP-2935 records the parent hash under (l2_block - 1) % window.
        let slot = (l2_block - 1) % HISTORY_SERVE_WINDOW;
        expected_eip2935.push((slot, parent_hash));
        parent_hash = block_hash;
    }

    // After the whole batch, every block's EIP-2935 entry must be present in
    // the committed bundle (later blocks must not corrupt earlier writes).
    for (slot, hash) in &expected_eip2935 {
        let got = read_bundle_slot(harness.state(), HISTORY_STORAGE_ADDRESS, U256::from(*slot));
        assert_eq!(
            got,
            U256::from_be_bytes(hash.0),
            "EIP-2935 entry for slot {slot} lost/corrupted in the bundle after the batch (got {got:x})",
        );
    }

    // The ArbOS recorded L1 height is re-touched every block; the bundle must
    // hold the final value, not a stale or dropped one.
    let arbos_l1 = read_bundle_slot(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        U256::from_be_bytes(ARBOS_L1_BLOCK_SLOT.0),
    );
    assert_eq!(
        arbos_l1,
        U256::from(last_l1),
        "ArbOS L1 block number must persist its final value across the batch (got {arbos_l1:x})",
    );
}
