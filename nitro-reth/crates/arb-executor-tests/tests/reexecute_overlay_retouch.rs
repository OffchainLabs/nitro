//! Cross-block overlay→bundle persistence under re-execution.
//!
//! Re-execution reuses one `State` across a chunk of blocks, committing each
//! with `merge_transitions(PlainState)`. An account credited via the state
//! overlay (deposits, fee mints, internal-tx transfers) in block N is held by
//! the bundle as `Changed`; when block N+1 re-touches it the overlay must emit
//! a transition that is a valid successor of the bundle status, or the credit
//! is dropped/corrupted in the committed state. This drives the exact
//! `executor_for_block` + `execute_block` entrypoint across two blocks and
//! asserts the recipient's balance accumulates in the bundle.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::{Block, BlockBody, Header};
use alloy_primitives::{address, Address, Bytes, Signature, B256, B64, U256};
use arb_alloy_consensus::tx::ArbDepositTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::ArbosHarness;
use arbos::header::compute_arbos_mixhash;
use reth_chainspec::ChainSpec;
use reth_evm::{block::BlockExecutor, ConfigureEvm};
use reth_primitives_traits::{RecoveredBlock, SealedBlock};
use revm::database::states::bundle_state::BundleRetention;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");
const FUNDER: Address = address!("00000000000000000000000000000000d05dd05d");
const RECIPIENT: Address = address!("00000000000000000000000000000000beefbeef");
const DEPOSIT: u128 = 1_000_000_000_000_000_000;

fn deposit_block(l2_block: u64, parent_hash: B256) -> Block<ArbTransactionSigned> {
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
        timestamp: 1_700_000_000 + l2_block,
        mix_hash: compute_arbos_mixhash(0, 1_000 + l2_block, ARBOS_VERSION, false),
        nonce: B64::from(1u64.to_be_bytes()),
        base_fee_per_gas: Some(0x1315410),
        extra_data: Bytes::from(vec![0u8; 32]),
        parent_beacon_block_root: None,
        blob_gas_used: None,
        excess_blob_gas: None,
        requests_hash: None,
    };
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(CHAIN_ID),
            l1_request_id: B256::repeat_byte(l2_block as u8),
            from: FUNDER,
            to: RECIPIENT,
            value: U256::from(DEPOSIT),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    Block {
        header,
        body: BlockBody {
            transactions: vec![tx],
            ommers: Default::default(),
            withdrawals: None,
        },
    }
}

#[test]
fn reexecute_overlay_credit_accumulates_across_blocks() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();
    harness
        .state()
        .merge_transitions(BundleRetention::PlainState);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut parent_hash = B256::repeat_byte(0x11);
    for l2_block in 1u64..=3 {
        let block = deposit_block(l2_block, parent_hash);
        let sealed = SealedBlock::seal_slow(block);
        parent_hash = sealed.hash();
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
    }

    let bundled = harness
        .state()
        .bundle_state
        .state
        .get(&RECIPIENT)
        .and_then(|a| a.info.as_ref())
        .map(|i| i.balance)
        .unwrap_or(U256::ZERO);
    assert_eq!(
        bundled,
        U256::from(DEPOSIT) * U256::from(3u64),
        "three cross-block deposits must accumulate in the committed bundle (got {bundled})",
    );
}
