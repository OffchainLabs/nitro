//! Deterministic Stylus differential matrix vs Nitro.
//!
//! Compiles the project's _wat/*.wat hostio primitives at test time, deploys
//! and activates each on both nodes, then sweeps calldata variants targeting
//! per-hostio edge cases. Diffs receipts via the existing DualExec.
//!
//! Marked `#[ignore]`. Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test stylus_matrix --release \
//!     -- --ignored --nocapture
//! Outputs to /tmp/stylus_matrix/{summary.json, divergences/}.

use std::{
    collections::HashSet,
    fs,
    path::PathBuf,
    sync::{
        atomic::{AtomicU64, AtomicUsize, Ordering},
        Mutex, OnceLock,
    },
    time::Instant,
};

static SEEN_BLOCK_DIFFS: OnceLock<Mutex<HashSet<(u64, String)>>> = OnceLock::new();
static SEEN_TX_DIFFS: OnceLock<Mutex<HashSet<(String, String)>>> = OnceLock::new();

fn seen_block() -> &'static Mutex<HashSet<(u64, String)>> {
    SEEN_BLOCK_DIFFS.get_or_init(|| Mutex::new(HashSet::new()))
}
fn seen_tx() -> &'static Mutex<HashSet<(String, String)>> {
    SEEN_TX_DIFFS.get_or_init(|| Mutex::new(HashSet::new()))
}

use alloy_primitives::{b256, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    shared_nodes::{fuzz_arbos_version, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const FUZZ_GAS_CAP: u64 = 4_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x71,
]);

static GLOBAL_MSG_IDX: AtomicU64 = AtomicU64::new(1);
static GLOBAL_DELAYED: AtomicU64 = AtomicU64::new(0);
static EOA_NONCE: AtomicU64 = AtomicU64::new(0);
static EOA_FUNDED: OnceLock<()> = OnceLock::new();

fn signing_key() -> B256 {
    b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
}
fn eoa() -> Address {
    derive_address(signing_key())
}

// -- WAT primitives -------------------------------------------------------

const WAT_KECCAK: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_keccak.wat"
));
const WAT_MSG_SENDER: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_msg_sender.wat"
));
const WAT_EMIT_LOG: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_emit_log.wat"
));
const WAT_STORAGE_LOAD: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_storage_load_bytes32.wat"
));
const WAT_STORAGE_STORE: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_storage_store_bytes32.wat"
));
const WAT_ACCOUNT_BALANCE: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_account_balance.wat"
));
const WAT_ACCOUNT_CODEHASH: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_account_codehash.wat"
));
const WAT_BLOCK_BASEFEE: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_block_basefee.wat"
));
const WAT_BLOCK_NUMBER: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_block_number.wat"
));
const WAT_BLOCK_TIMESTAMP: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_block_timestamp.wat"
));
const WAT_CHAINID: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_chainid.wat"
));
const WAT_MSG_VALUE: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_msg_value.wat"
));
const WAT_PAY_FOR_MEMORY_GROW: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_pay_for_memory_grow.wat"
));
const WAT_TRANSIENT_LOAD: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_transient_load_bytes32.wat"
));
const WAT_TRANSIENT_STORE: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_transient_store_bytes32.wat"
));
const WAT_TX_ORIGIN: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_tx_origin.wat"
));

#[derive(Clone)]
struct Primitive {
    name: &'static str,
    wat: &'static str,
    /// Per-primitive calldata variants. (variant_label, calldata_bytes).
    variants: Vec<(String, Vec<u8>)>,
}

fn keccak_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    for (label, len) in [
        ("len_0", 0usize),
        ("len_1", 1),
        ("len_31", 31),
        ("len_32", 32),
        ("len_33", 33),
        ("len_64", 64),
        ("len_136", 136), // sha3 rate boundary
        ("len_137", 137),
        ("len_256", 256),
        ("len_1024", 1024),
        ("len_2048", 2048),
    ] {
        let mut data = vec![0u8; len];
        for (i, b) in data.iter_mut().enumerate() {
            *b = (i & 0xff) as u8;
        }
        v.push((label.to_string(), data));
    }
    v
}

fn msg_sender_variants() -> Vec<(String, Vec<u8>)> {
    vec![("eoa_call".into(), vec![])]
}

fn emit_log_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    // calldata = [num_topics_byte, topics(32*N), payload]
    for topics in 0u8..=4u8 {
        for payload_len in [0usize, 1, 32, 256] {
            let mut data = vec![topics];
            for t in 0..topics {
                let mut topic = [0u8; 32];
                topic[0] = t;
                data.extend_from_slice(&topic);
            }
            for i in 0..payload_len {
                data.push((i & 0xff) as u8);
            }
            v.push((format!("t{topics}_p{payload_len}"), data));
        }
    }
    // Bad inputs (must revert in canon):
    let mut data = vec![5u8]; // > 4 topics
    data.extend_from_slice(&[0u8; 32 * 5]);
    v.push(("t5_invalid".into(), data));
    v
}

fn storage_load_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    let mut slot_zero = vec![0u8; 32];
    v.push(("slot_zero_cold".into(), slot_zero.clone()));
    let mut slot_one = vec![0u8; 32];
    slot_one[31] = 1;
    v.push(("slot_one_cold".into(), slot_one));
    let mut slot_max = vec![0xff; 32];
    v.push(("slot_max_cold".into(), slot_max));
    slot_zero[31] = 0;
    v.push(("slot_zero_again".into(), slot_zero));
    v
}

fn storage_store_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    fn pair(key_byte: u8, val_pat: u8) -> Vec<u8> {
        let mut out = vec![0u8; 64];
        out[31] = key_byte;
        for i in 32..64 {
            out[i] = val_pat;
        }
        out
    }
    v.push(("set_zero_to_one".into(), pair(0xa0, 0x01)));
    v.push(("reset_one_to_two".into(), pair(0xa0, 0x02)));
    v.push(("clear_to_zero".into(), pair(0xa0, 0x00)));
    let mut huge = vec![0u8; 64];
    huge[31] = 0xa1;
    huge[32..].fill(0xff);
    v.push(("set_zero_to_max".into(), huge));
    v
}

fn account_addr_variants() -> Vec<(String, Vec<u8>)> {
    let arbwasm = vec![
        0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
    ];
    let zero = vec![0u8; 20];
    let mut max = vec![0u8; 20];
    max.fill(0xff);
    let mut sequencer = vec![
        0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75,
        0x65, 0x6e, 0x63, 0x65, 0x72,
    ];
    let _ = sequencer.len();
    let mut eoa_addr = vec![0u8; 20];
    let eoa = derive_address(signing_key());
    eoa_addr.copy_from_slice(eoa.as_slice());
    vec![
        ("zero_addr".into(), zero),
        ("self_eoa".into(), eoa_addr),
        ("arbwasm_precompile".into(), arbwasm),
        ("max_addr".into(), max),
        ("sequencer_alias".into(), sequencer),
    ]
}

fn empty_calldata_variants() -> Vec<(String, Vec<u8>)> {
    vec![("empty".into(), vec![])]
}

fn pay_for_memory_grow_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    for pages in [0u32, 1, 2, 4, 16, 32, 64, 100] {
        let bytes = pages.to_be_bytes().to_vec();
        v.push((format!("pages_{pages}"), bytes));
    }
    v
}

fn transient_load_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    let mut zero = vec![0u8; 32];
    v.push(("key_zero".into(), zero.clone()));
    zero[31] = 1;
    v.push(("key_one".into(), zero));
    let max = vec![0xff; 32];
    v.push(("key_max".into(), max));
    v
}

fn transient_store_variants() -> Vec<(String, Vec<u8>)> {
    let mut v = Vec::new();
    fn pair(k: u8, val_pat: u8) -> Vec<u8> {
        let mut out = vec![0u8; 64];
        out[31] = k;
        out[32..].fill(val_pat);
        out
    }
    v.push(("set_zero_to_one".into(), pair(0xb0, 0x01)));
    v.push(("overwrite_to_max".into(), pair(0xb0, 0xff)));
    v.push(("clear_to_zero".into(), pair(0xb0, 0x00)));
    v
}

fn primitives() -> Vec<Primitive> {
    vec![
        Primitive {
            name: "keccak",
            wat: WAT_KECCAK,
            variants: keccak_variants(),
        },
        Primitive {
            name: "msg_sender",
            wat: WAT_MSG_SENDER,
            variants: msg_sender_variants(),
        },
        Primitive {
            name: "emit_log",
            wat: WAT_EMIT_LOG,
            variants: emit_log_variants(),
        },
        Primitive {
            name: "storage_load",
            wat: WAT_STORAGE_LOAD,
            variants: storage_load_variants(),
        },
        Primitive {
            name: "storage_store",
            wat: WAT_STORAGE_STORE,
            variants: storage_store_variants(),
        },
        Primitive {
            name: "account_balance",
            wat: WAT_ACCOUNT_BALANCE,
            variants: account_addr_variants(),
        },
        Primitive {
            name: "account_codehash",
            wat: WAT_ACCOUNT_CODEHASH,
            variants: account_addr_variants(),
        },
        Primitive {
            name: "block_basefee",
            wat: WAT_BLOCK_BASEFEE,
            variants: empty_calldata_variants(),
        },
        Primitive {
            name: "block_number",
            wat: WAT_BLOCK_NUMBER,
            variants: empty_calldata_variants(),
        },
        Primitive {
            name: "block_timestamp",
            wat: WAT_BLOCK_TIMESTAMP,
            variants: empty_calldata_variants(),
        },
        Primitive {
            name: "chainid",
            wat: WAT_CHAINID,
            variants: empty_calldata_variants(),
        },
        Primitive {
            name: "msg_value",
            wat: WAT_MSG_VALUE,
            variants: empty_calldata_variants(),
        },
        Primitive {
            name: "pay_for_memory_grow",
            wat: WAT_PAY_FOR_MEMORY_GROW,
            variants: pay_for_memory_grow_variants(),
        },
        Primitive {
            name: "transient_load",
            wat: WAT_TRANSIENT_LOAD,
            variants: transient_load_variants(),
        },
        Primitive {
            name: "transient_store",
            wat: WAT_TRANSIENT_STORE,
            variants: transient_store_variants(),
        },
        Primitive {
            name: "tx_origin",
            wat: WAT_TX_ORIGIN,
            variants: empty_calldata_variants(),
        },
    ]
}

// -- Scenario building ---------------------------------------------------

fn build_init_code(wasm: &[u8]) -> Vec<u8> {
    let mut body = Vec::with_capacity(3 + wasm.len());
    body.extend_from_slice(&[0xEF, 0xF0, 0x00]);
    body.extend_from_slice(wasm);
    let size = body.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(12 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0c, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
        0xF3,
    ]);
    out.extend_from_slice(&body);
    out
}

fn create_address(sender: Address, nonce: u64) -> Address {
    let nonce_rlp = if nonce == 0 {
        vec![0x80u8]
    } else {
        let bytes = nonce.to_be_bytes();
        let trimmed: &[u8] = bytes
            .iter()
            .position(|b| *b != 0)
            .map(|i| &bytes[i..])
            .unwrap_or(&bytes[..0]);
        if trimmed.len() == 1 && trimmed[0] < 0x80 {
            vec![trimmed[0]]
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut payload = Vec::new();
    payload.push(0x80 + 20);
    payload.extend_from_slice(sender.as_slice());
    payload.extend_from_slice(&nonce_rlp);
    let mut rlp = vec![0xC0 + payload.len() as u8];
    rlp.extend_from_slice(&payload);
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.as_slice()[12..])
}

fn next_idx() -> u64 {
    GLOBAL_MSG_IDX.fetch_add(1, Ordering::Relaxed)
}
fn next_delayed() -> u64 {
    GLOBAL_DELAYED.fetch_add(1, Ordering::Relaxed) + 1
}
fn current_delayed() -> u64 {
    GLOBAL_DELAYED.load(Ordering::Relaxed)
}
fn next_nonce() -> u64 {
    EOA_NONCE.fetch_add(1, Ordering::Relaxed)
}

fn signed_eip1559(
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to,
        value,
        data,
        gas_limit: gas,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn fund_step_once() -> Vec<ScenarioStep> {
    let mut out = Vec::new();
    if EOA_FUNDED.set(()).is_err() {
        return out;
    }
    let dep = DepositBuilder {
        from: eoa(),
        to: eoa(),
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: 0,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = dep.build() {
        let idx = next_idx();
        let delayed = next_delayed();
        out.push(message_step(idx, msg, delayed));
    }
    out
}

/// Deploy + activate a Stylus program from raw WASM bytes, return the
/// deployed contract address. Returns the scenario steps (deploy then
/// activate).
fn deploy_and_activate(wasm: &[u8]) -> (Vec<ScenarioStep>, Address) {
    let mut steps = Vec::new();
    let init_code = build_init_code(wasm);

    let deploy_nonce = next_nonce();
    let deploy_addr = create_address(eoa(), deploy_nonce);

    let deploy = signed_eip1559(
        deploy_nonce,
        None,
        Bytes::from(init_code),
        U256::ZERO,
        FUZZ_GAS_CAP,
    );
    if let Ok(msg) = deploy.build() {
        let idx = next_idx();
        let delayed = current_delayed();
        steps.push(message_step(idx, msg, delayed));
    }

    let activate_nonce = next_nonce();
    let mut activate_data = Vec::with_capacity(4 + 32);
    activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]); // activateProgram(address)
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(deploy_addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let activate = signed_eip1559(
        activate_nonce,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)), // 0.001 ETH
        FUZZ_GAS_CAP,
    );
    if let Ok(msg) = activate.build() {
        let idx = next_idx();
        let delayed = current_delayed();
        steps.push(message_step(idx, msg, delayed));
    }

    (steps, deploy_addr)
}

fn build_invoke_scenario(
    name: String,
    deploy_addr: Address,
    calldata: Vec<u8>,
) -> Option<Scenario> {
    let invoke_nonce = next_nonce();
    let invoke = signed_eip1559(
        invoke_nonce,
        Some(deploy_addr),
        Bytes::from(calldata),
        U256::ZERO,
        FUZZ_GAS_CAP,
    );
    let msg = invoke.build().ok()?;
    let idx = next_idx();
    let delayed = current_delayed();
    Some(Scenario {
        name,
        description: String::new(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps: vec![message_step(idx, msg, delayed)],
    })
}

// -- Test ----------------------------------------------------------------

#[test]
#[ignore]
fn stylus_diff_matrix() {
    let out_dir = PathBuf::from(
        std::env::var("ARB_STYLUS_MATRIX_OUT")
            .unwrap_or_else(|_| "/tmp/stylus_matrix".to_string()),
    );
    let _ = fs::remove_dir_all(&out_dir);
    let div_dir = out_dir.join("divergences");
    fs::create_dir_all(&div_dir).expect("mkdir");

    let limit_per_primitive: usize = std::env::var("ARB_STYLUS_MATRIX_LIMIT_PER_PRIMITIVE")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(usize::MAX);
    let primitive_filter = std::env::var("ARB_STYLUS_MATRIX_PRIMITIVE").ok();

    let nodes = shared_dual_exec();
    let total = AtomicUsize::new(0);
    let diverged = AtomicUsize::new(0);
    let harness_errs = AtomicUsize::new(0);
    let activate_failed = AtomicUsize::new(0);

    let start = Instant::now();

    // Fund the EOA exactly once (first scenario).
    {
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        let fund_steps = fund_step_once();
        if !fund_steps.is_empty() {
            let scen = Scenario {
                name: "fund_eoa".into(),
                description: "fund the matrix's shared EOA".into(),
                setup: ScenarioSetup {
                    l2_chain_id: FUZZ_L2_CHAIN_ID,
                    arbos_version: fuzz_arbos_version(),
                    genesis: None,
                },
                steps: fund_steps,
            };
            let _ = nodes.run(&scen);
        }
    }

    for primitive in primitives() {
        if let Some(ref f) = primitive_filter {
            if primitive.name != f {
                continue;
            }
        }
        eprintln!("[stylus_matrix] === primitive '{}' ===", primitive.name);

        let wasm = match wat::parse_bytes(primitive.wat.as_bytes()) {
            Ok(b) => b.into_owned(),
            Err(e) => {
                eprintln!("[stylus_matrix] WAT compile failed for {}: {e}", primitive.name);
                harness_errs.fetch_add(1, Ordering::Relaxed);
                continue;
            }
        };

        // Deploy + activate ONCE per primitive.
        let (deploy_steps, deploy_addr) = deploy_and_activate(&wasm);
        let setup_scen = Scenario {
            name: format!("{}_deploy_activate", primitive.name),
            description: format!("deploy+activate {} at {}", primitive.name, deploy_addr),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: fuzz_arbos_version(),
                genesis: None,
            },
            steps: deploy_steps,
        };
        {
            let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
            match nodes.run(&setup_scen) {
                Ok(report) => {
                    let mut sb = seen_block().lock().unwrap();
                    let new_block_diffs: Vec<_> = report
                        .block_diffs
                        .iter()
                        .filter(|d| d.field != "state_root" && d.field != "parent_hash")
                        .filter(|d| sb.insert((d.number, d.field.clone())))
                        .collect();
                    drop(sb);
                    let mut st = seen_tx().lock().unwrap();
                    let new_tx_diffs: Vec<_> = report
                        .tx_diffs
                        .iter()
                        .filter(|d| {
                            st.insert((format!("{:?}", d.tx_hash), d.field.clone()))
                        })
                        .collect();
                    drop(st);
                    let real = !new_block_diffs.is_empty()
                        || !new_tx_diffs.is_empty()
                        || !report.state_diffs.is_empty()
                        || !report.log_diffs.is_empty();
                    if real {
                        activate_failed.fetch_add(1, Ordering::Relaxed);
                        let payload = serde_json::json!({
                            "case": format!("{}_deploy_activate", primitive.name),
                            "primitive": primitive.name,
                            "phase": "deploy_activate",
                            "deploy_addr": format!("{deploy_addr}"),
                            "block_diffs_new": format!("{:#?}", new_block_diffs),
                            "tx_diffs_new": format!("{:#?}", new_tx_diffs),
                            "state_diffs": format!("{:#?}", report.state_diffs),
                            "log_diffs": format!("{:#?}", report.log_diffs),
                        });
                        let path = div_dir.join(format!("{}_deploy_activate.json", primitive.name));
                        let _ = fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
                        eprintln!(
                            "[stylus_matrix] DEPLOY/ACTIVATE DIVERGED for {} -> {}",
                            primitive.name,
                            path.display()
                        );
                    }
                }
                Err(e) => {
                    eprintln!("[stylus_matrix] activate err for {}: {e}", primitive.name);
                }
            }
        }

        for (i, (variant_label, calldata)) in primitive
            .variants
            .iter()
            .enumerate()
            .take(limit_per_primitive)
        {
            let case_name = format!("{}_{}", primitive.name, variant_label);
            let Some(scen) = build_invoke_scenario(case_name.clone(), deploy_addr, calldata.clone())
            else {
                continue;
            };

            let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
            match nodes.run(&scen) {
                Ok(report) => {
                    let mut sb = seen_block().lock().unwrap();
                    let new_block_diffs: Vec<_> = report
                        .block_diffs
                        .iter()
                        .filter(|d| d.field != "state_root" && d.field != "parent_hash")
                        .filter(|d| sb.insert((d.number, d.field.clone())))
                        .collect();
                    drop(sb);
                    let mut st = seen_tx().lock().unwrap();
                    let new_tx_diffs: Vec<_> = report
                        .tx_diffs
                        .iter()
                        .filter(|d| {
                            st.insert((format!("{:?}", d.tx_hash), d.field.clone()))
                        })
                        .collect();
                    drop(st);
                    let real = !new_block_diffs.is_empty()
                        || !new_tx_diffs.is_empty()
                        || !report.state_diffs.is_empty()
                        || !report.log_diffs.is_empty();
                    if real {
                        diverged.fetch_add(1, Ordering::Relaxed);
                        let payload = serde_json::json!({
                            "case": case_name,
                            "primitive": primitive.name,
                            "variant": variant_label,
                            "calldata_len": calldata.len(),
                            "calldata_hex": format!("0x{}", hex::encode(calldata)),
                            "deploy_addr": format!("{deploy_addr}"),
                            "block_diffs_new": format!("{:#?}", new_block_diffs),
                            "tx_diffs_new": format!("{:#?}", new_tx_diffs),
                            "state_diffs": format!("{:#?}", report.state_diffs),
                            "log_diffs": format!("{:#?}", report.log_diffs),
                        });
                        let path = div_dir.join(format!("{case_name}.json"));
                        let _ = fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
                        eprintln!("[stylus_matrix] DIVERGE {case_name}");
                    }
                }
                Err(e) => {
                    harness_errs.fetch_add(1, Ordering::Relaxed);
                    eprintln!("[stylus_matrix] HARNESS ERR [{i}] {case_name}: {e}");
                }
            }
            total.fetch_add(1, Ordering::Relaxed);
        }
    }

    let summary = serde_json::json!({
        "total_invokes": total.load(Ordering::Relaxed),
        "diverged": diverged.load(Ordering::Relaxed),
        "harness_errors": harness_errs.load(Ordering::Relaxed),
        "activate_failed_primitives": activate_failed.load(Ordering::Relaxed),
        "elapsed_secs": start.elapsed().as_secs(),
    });
    fs::write(
        out_dir.join("summary.json"),
        serde_json::to_vec_pretty(&summary).unwrap(),
    )
    .expect("write summary");

    eprintln!("[stylus_matrix] done: {:#}", summary);
}
