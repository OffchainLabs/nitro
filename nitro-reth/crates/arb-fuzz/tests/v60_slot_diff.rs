//! v60 baseline state_root divergence — slot dump diagnostic.
//!
//! Submits a single ETH deposit at v60 and compares the ArbOS state
//! account (0xA4B05fff…) storage slots between Nitro and arbreth using
//! eth_getStorageAt for known root-level + per-subspace offsets.
//!
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     ARB_FUZZ_ARBOS_VERSION=60 \
//!     cargo test -p arb-fuzz --test v60_slot_diff --release \
//!     -- --ignored v60_block1_slot_diff --nocapture

use alloy_primitives::{address, keccak256, Address, B256, U256};

use arb_fuzz::shared_nodes::{next_msg_idx, shared_dual_exec};
use arb_test_harness::{
    messaging::{DepositBuilder, MessageBuilder},
    node::{BlockId, ExecutionNode},
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const ARBOS_STATE_ADDRESS: Address = address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");

const SUBSPACES: &[(&str, &[u8])] = &[
    ("root", &[]),
    ("l1pricing", &[0]),
    ("l2pricing", &[1]),
    ("retryables", &[2]),
    ("address_table", &[3]),
    ("chain_owner", &[4]),
    ("send_merkle", &[5]),
    ("blockhashes", &[6]),
    ("chain_config", &[7]),
    ("programs", &[8]),
    ("features", &[9]),
    ("native_token_owner", &[10]),
    ("tx_filtering", &[11]),
];

fn derive_sub_key(parent: B256, sub: &[u8]) -> B256 {
    let base: &[u8] = if parent == B256::ZERO {
        &[]
    } else {
        parent.as_slice()
    };
    let mut buf = Vec::with_capacity(base.len() + sub.len());
    buf.extend_from_slice(base);
    buf.extend_from_slice(sub);
    keccak256(&buf)
}

fn slot_for(storage_key: &[u8], offset: u64) -> B256 {
    const BOUNDARY: usize = 31;
    let mut key_bytes = [0u8; 32];
    key_bytes[24..32].copy_from_slice(&offset.to_be_bytes());
    let mut buf = Vec::with_capacity(storage_key.len() + BOUNDARY);
    buf.extend_from_slice(storage_key);
    buf.extend_from_slice(&key_bytes[..BOUNDARY]);
    let h = keccak256(&buf);
    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key_bytes[BOUNDARY];
    B256::from(mapped)
}

#[test]
#[ignore]
fn v60_block1_slot_diff() {
    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual exec mutex");

    let idx = next_msg_idx();
    let deposit = DepositBuilder {
        from: address!("00000000000000000000000000000000000000aa"),
        to: address!("00000000000000000000000000000000000000bb"),
        amount: U256::from(1_000_000_000_000_000_000u64),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: 1_000_000_000,
    };
    let msg = deposit.build().expect("deposit");

    let scenario = Scenario {
        name: "v60_single_deposit".into(),
        description: "single deposit at v60 — slot dump diagnostic".into(),
        setup: ScenarioSetup {
            l2_chain_id: arb_fuzz::shared_nodes::FUZZ_L2_CHAIN_ID,
            arbos_version: 60,
            genesis: None,
        },
        steps: vec![ScenarioStep::Message {
            idx,
            message: msg,
            delayed_messages_read: idx + 1,
        }],
    };

    let report = nodes.run(&scenario).expect("run");
    eprintln!(
        "block_diffs={} tx_diffs={} log_diffs={}",
        report.block_diffs.len(),
        report.tx_diffs.len(),
        report.log_diffs.len(),
    );
    for d in &report.block_diffs {
        eprintln!(
            "  block#{} field={} left={} right={}",
            d.number, d.field, d.left, d.right
        );
    }

    let left_latest = nodes.left.block(BlockId::Latest).expect("left latest");
    let right_latest = nodes.right.block(BlockId::Latest).expect("right latest");
    let at = BlockId::Number(left_latest.number.min(right_latest.number));
    eprintln!(
        "left.latest={} right.latest={} comparing at #{:?}",
        left_latest.number, right_latest.number, at,
    );

    // Build a list of slots to probe across all subspaces.
    let mut probes: Vec<(String, B256)> = Vec::new();
    for (name, sub) in SUBSPACES {
        let storage_key = if sub.is_empty() {
            B256::ZERO
        } else {
            derive_sub_key(B256::ZERO, sub)
        };
        let sk_slice: &[u8] = if storage_key == B256::ZERO {
            &[]
        } else {
            storage_key.as_slice()
        };
        for offset in 0u64..30 {
            let slot = slot_for(sk_slice, offset);
            probes.push((format!("{name}[{offset}]"), slot));
        }
    }

    let mut diffs = 0usize;
    for (label, slot) in &probes {
        let l = nodes
            .left
            .storage(ARBOS_STATE_ADDRESS, *slot, at.clone())
            .unwrap_or(B256::ZERO);
        let r = nodes
            .right
            .storage(ARBOS_STATE_ADDRESS, *slot, at.clone())
            .unwrap_or(B256::ZERO);
        if l != r {
            eprintln!("  DIFF {label:30}  slot={slot:?}\n    nitro  = {l:?}\n    arbreth= {r:?}");
            diffs += 1;
        }
    }

    eprintln!(
        "\n=== v60 ArbOS-state-account slot diff: {} differing slots out of {} probed ===",
        diffs,
        probes.len(),
    );

    // Account-level diff over genesis accounts + likely systems.
    let suspects: &[Address] = &[
        address!("00000000000000000000000000000000000000aa"), // deposit from
        address!("00000000000000000000000000000000000000bb"), // deposit to
        address!("0000000000000000000000000000000000000064"),
        address!("0000000000000000000000000000000000000065"),
        address!("0000000000000000000000000000000000000066"),
        address!("0000000000000000000000000000000000000067"),
        address!("0000000000000000000000000000000000000068"),
        address!("0000000000000000000000000000000000000069"),
        address!("000000000000000000000000000000000000006b"),
        address!("000000000000000000000000000000000000006c"),
        address!("000000000000000000000000000000000000006d"),
        address!("000000000000000000000000000000000000006e"),
        address!("000000000000000000000000000000000000006f"),
        address!("0000000000000000000000000000000000000070"),
        address!("0000000000000000000000000000000000000071"),
        address!("0000000000000000000000000000000000000072"),
        address!("0000000000000000000000000000000000000073"),
        address!("0000000000000000000000000000000000000074"),
        address!("00000000000000000000000000000000000000ff"),
        address!("00000000000000000000000000000000000A4B05"),
        ARBOS_STATE_ADDRESS,
        address!("A4B0500000000000000000000000000000000001"), // FILTERED_TX_STATE_ADDRESS
        address!("0000F90827F1C53a10cb7A02335B175320002935"), // history storage (EIP-2935)
        // Default fee accounts (zero address chain owner default).
        address!("0000000000000000000000000000000000000000"),
    ];

    eprintln!("\n--- account-level diff ---");
    let mut acct_diffs = 0usize;
    for a in suspects {
        let lb = nodes.left.balance(*a, at.clone()).unwrap_or(U256::ZERO);
        let rb = nodes.right.balance(*a, at.clone()).unwrap_or(U256::ZERO);
        let ln = nodes.left.nonce(*a, at.clone()).unwrap_or(0);
        let rn = nodes.right.nonce(*a, at.clone()).unwrap_or(0);
        let lc = nodes
            .left
            .code(*a, at.clone())
            .map(|b| b.len())
            .unwrap_or(0);
        let rc = nodes
            .right
            .code(*a, at.clone())
            .map(|b| b.len())
            .unwrap_or(0);
        if lb != rb || ln != rn || lc != rc {
            eprintln!(
                "  DIFF {a:?}: bal nitro={lb} arbreth={rb} | nonce {ln}/{rn} | codelen {lc}/{rc}"
            );
            acct_diffs += 1;
        }
    }
    eprintln!(
        "=== account-level diffs: {} of {} addresses ===",
        acct_diffs,
        suspects.len()
    );
}
