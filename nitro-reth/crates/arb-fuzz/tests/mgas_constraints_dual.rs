//! Differential test for multi-gas pricing under ACTIVE constraints.
//!
//! Boots a fresh `NitroDocker` + `ArbrethProcess` pair (NOT shared, because the
//! chain owner is test-specific so an owner can configure constraints). The
//! owner installs a constraint weighting `StorageAccessWrite`, then drives
//! `SSTORE`-reset transactions. With per-resource attribution the write gas
//! grows that constraint's backlog and the multi-gas base fee escalates
//! identically on both nodes; lumping execution gas into computation would
//! leave the write backlog flat and diverge.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test mgas_constraints_dual --release \
//!     -- --ignored --nocapture

use std::sync::{
    atomic::{AtomicU64, Ordering},
    Mutex,
};

/// Each test spawns its own Nitro + arbreth pair, so they must not run
/// concurrently (Docker / port / process contention). Serialize them.
static SERIAL: Mutex<()> = Mutex::new(());

use alloy_primitives::{address, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::interop::{wrap_init_code, WhichProgram},
    scaffolding::selector4,
};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
        TxRequest,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const L2_CHAIN_ID: u64 = 412_348;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;

const SEQUENCER_ALIAS: Address = address!("a4b000000000000000000073657175656e636572");
const ARBOWNER: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x70,
]);
const ARBGASINFO: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6c,
]);
const FUNDER: Address = Address::new([0xa1; 20]);

/// StorageAccessRead resource kind (`ResourceKind` discriminant).
const KIND_STORAGE_READ: u8 = 3;
/// StorageAccessWrite resource kind (`ResourceKind` discriminant).
const KIND_STORAGE_WRITE: u8 = 4;
/// ArbAddressTable precompile address (0x66).
const ARBADDRESSTABLE: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x66,
]);

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn word(v: u64) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[24..].copy_from_slice(&v.to_be_bytes());
    w
}

/// Hand-encoded `setMultiGasPricingConstraints` with a single constraint and a
/// single resource weight, matching the `(((uint8,uint64)[],uint32,uint64,uint64)[])`
/// layout the precompile decodes.
fn set_constraint_calldata(
    window: u32,
    target: u64,
    backlog: u64,
    resource: u8,
    weight: u64,
) -> Vec<u8> {
    let mut d = Vec::with_capacity(4 + 320);
    d.extend_from_slice(&selector4(
        "setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])",
    ));
    d.extend_from_slice(&word(0x20)); // offset to constraints array
    d.extend_from_slice(&word(1)); // constraints.length
    d.extend_from_slice(&word(0x20)); // element[0] offset (relative to array data)
    d.extend_from_slice(&word(0x80)); // resources offset (relative to struct)
    d.extend_from_slice(&word(window as u64));
    d.extend_from_slice(&word(target));
    d.extend_from_slice(&word(backlog));
    d.extend_from_slice(&word(1)); // resources.length
    d.extend_from_slice(&word(resource as u64));
    d.extend_from_slice(&word(weight));
    d
}

/// Runtime that stores `calldata[0:32]` into slot 0: PUSH1 0, CALLDATALOAD,
/// PUSH1 0, SSTORE, STOP.
const SSTORE_RUNTIME: [u8; 7] = [0x60, 0x00, 0x35, 0x60, 0x00, 0x55, 0x00];

fn create_address(deployer: Address, nonce: u64) -> Address {
    use alloy_primitives::keccak256;
    let mut rlp = Vec::new();
    rlp.push(0xd6);
    rlp.push(0x94);
    rlp.extend_from_slice(deployer.as_slice());
    if nonce == 0 {
        rlp.push(0x80);
    } else if nonce < 0x80 {
        rlp.push(nonce as u8);
    } else {
        let b = nonce.to_be_bytes();
        let start = b.iter().position(|&x| x != 0).unwrap_or(7);
        let trimmed = &b[start..];
        rlp.push(0x80 + trimmed.len() as u8);
        rlp.extend_from_slice(trimmed);
        rlp[0] = 0xd6 + (trimmed.len() as u8);
    }
    Address::from_slice(&keccak256(&rlp)[12..])
}

struct Rig {
    dual: DualExec<NitroDocker, ArbrethProcess>,
}

impl Rig {
    fn spawn(owner: Address) -> Self {
        let mock = MockL1::start(L1_CHAIN_ID).expect("mock l1 start");
        let genesis = GenesisBuilder::new(L2_CHAIN_ID, ARBOS_VERSION)
            .with_initial_chain_owner(owner)
            .build()
            .expect("genesis build");
        let ctx = NodeStartCtx {
            binary: None,
            l2_chain_id: L2_CHAIN_ID,
            l1_chain_id: L1_CHAIN_ID,
            mock_l1_rpc: mock.rpc_url(),
            genesis,
            jwt_hex: String::new(),
            workdir: std::path::PathBuf::new(),
            http_port: 0,
            authrpc_port: 0,
        };
        let nitro = NitroDocker::start(&ctx).expect("nitro docker start");
        let arbreth = ArbrethProcess::start(&ctx).expect("arbreth start");
        std::mem::forget(mock);
        Rig {
            dual: DualExec::new(nitro, arbreth),
        }
    }
}

struct Idx(AtomicU64);
impl Idx {
    fn new() -> Self {
        Self(AtomicU64::new(1))
    }
    fn next(&self) -> u64 {
        self.0.fetch_add(1, Ordering::SeqCst)
    }
}

fn owner_tx(nonce: u64, to: Option<Address>, data: Vec<u8>, gas: u64) -> SignedL2TxBuilder {
    owner_tx_l1(nonce, to, data, gas, FUZZ_L1_BASE_FEE)
}

fn owner_tx_l1(
    nonce: u64,
    to: Option<Address>,
    data: Vec<u8>,
    gas: u64,
    base_fee_l1: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value: U256::ZERO,
        data: Bytes::from(data),
        gas_limit: gas,
        gas_price: 1_000_000_000,
        max_fee_per_gas: 1_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1,
    }
}

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

#[test]
#[ignore]
fn multi_gas_write_constraint_backlog_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    // Install a constraint weighting StorageAccessWrite heavily so write gas
    // dominates the backlog and the base fee escalates measurably.
    let cons = set_constraint_calldata(60, 100_000, 0, KIND_STORAGE_WRITE, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(0, Some(ARBOWNER), cons, 2_000_000)
            .build()
            .expect("set"),
        1,
    ));

    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(1, None, wrap_init_code(&SSTORE_RUNTIME), 2_000_000)
            .build()
            .expect("deploy"),
        1,
    ));
    let contract = create_address(owner, 1);

    // First write initialises slot 0 (zero→nonzero, a storage-growth create);
    // subsequent writes are resets that price into StorageAccessWrite.
    for (n, v) in [(2u64, 1u64), (3, 2), (4, 3)] {
        let i = idx.next();
        steps.push(msg_step(
            i,
            owner_tx(n, Some(contract), word(v).to_vec(), 2_000_000)
                .build()
                .expect("call"),
            1,
        ));
    }

    let scenario = Scenario {
        name: "mgas_write_constraint".into(),
        description: "write-weighted constraint backlog escalation".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "execution diverged under active write constraint: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // Preconditions: the owner txs must have taken effect, so the write
    // backlog had something to grow. Empty code or an empty constraint set
    // would make the comparison below pass trivially.
    let code = rig
        .dual
        .right
        .code(contract, BlockId::Latest)
        .expect("code");
    assert!(!code.is_empty(), "storage contract did not deploy");
    let nonce = rig.dual.right.nonce(owner, BlockId::Latest).expect("nonce");
    assert!(nonce >= 5, "owner txs did not all execute (nonce {nonce})");

    // Per-resource base fees (uint256[]) derived from the constraint backlog
    // must match byte-for-byte, and the write dimension must have escalated
    // above the floor — proving the SSTORE gas reached StorageAccessWrite and
    // arbreth priced it exactly as Nitro.
    let base_fee_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let l = rig
        .dual
        .left
        .eth_call(base_fee_call.clone(), BlockId::Latest)
        .ok();
    let r = rig.dual.right.eth_call(base_fee_call, BlockId::Latest).ok();
    assert_eq!(
        l, r,
        "getMultiGasBaseFee diverged: nitro={l:?} arbreth={r:?}"
    );

    let fees = decode_uint256_array(&r.expect("base fee bytes"));
    let floor = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBGASINFO),
                data: Some(Bytes::from(selector4("getMinimumGasPrice()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b))
        .unwrap_or(U256::ZERO);
    let max_fee = fees.iter().copied().max().unwrap_or(U256::ZERO);
    eprintln!("[mgas] floor={floor} max_fee={max_fee} fees={fees:?}");
    assert!(
        max_fee > floor,
        "no resource base fee escalated above floor {floor}; the write constraint \
         never bit, so the byte-for-byte comparison would be insensitive"
    );
}

const ARBOWNERPUBLIC: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6b,
]);
const ARBNATIVETOKENMANAGER: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x73,
]);

fn word_addr(a: Address) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[12..].copy_from_slice(a.as_slice());
    w
}

fn word_u256(v: U256) -> [u8; 32] {
    v.to_be_bytes()
}

/// Tip-collection routing parity. With collectTips enabled (v60), a transaction
/// carrying a priority fee pays it (rather than the tip being dropped), and the
/// tip is routed to the network fee account. The resulting balances — hence the
/// state root — must match Nitro.
#[test]
#[ignore]
fn collect_tips_routing_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let sender_key = B256::repeat_byte(0x43);
    let sender = derive_address(sender_key);
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    const T0: u64 = 1_700_000_000;
    let amount = U256::from(10u128).pow(U256::from(20u64));
    for to in [owner, sender] {
        let i = idx.next();
        let dep = DepositBuilder {
            from: FUNDER,
            to,
            amount,
            l1_block_number: 1,
            timestamp: T0,
            request_seq: i,
            base_fee_l1: 0,
        }
        .build()
        .expect("deposit");
        steps.push(msg_step(i, dep, 1));
    }

    let mut enable = selector4("setCollectTips(bool)").to_vec();
    enable.extend_from_slice(&word(1));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), enable, 2_000_000, 0)
            .build()
            .expect("enable"),
        1,
    ));

    // A tx from a separate sender carrying a priority fee, in a later block so
    // tip collection is already enabled.
    let recipient = address!("00000000000000000000000000000000deadbeef");
    let tipped = SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce: 0,
        to: Some(recipient),
        value: U256::from(1u64),
        data: Bytes::new(),
        gas_limit: 1_000_000,
        gas_price: 2_000_000_000,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 1_000_000_000,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: sender_key,
        l1_block_number: 1,
        timestamp: T0 + 10,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: 0,
    }
    .build()
    .expect("tipped tx");
    let tip_idx = idx.next();
    let tip_hash = arb_test_harness::messaging::signed_l2_tx_hash(&tipped);
    steps.push(msg_step(tip_idx, tipped, 1));

    let scenario = Scenario {
        name: "collect_tips_routing".into(),
        description: "tip routed to the network fee account under collectTips".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "tip routing diverged from Nitro: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // Sensitivity: tip collection must be enabled, and the tx must have actually
    // paid a tip (effective gas price well above the base fee) so the routing
    // path was exercised.
    let collect = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBOWNERPUBLIC),
                data: Some(Bytes::from(selector4("getCollectTips()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b) == U256::from(1u64))
        .unwrap_or(false);
    assert!(collect, "collectTips was not enabled");
    let r = rig
        .dual
        .right
        .receipt(tip_hash.expect("tip hash"))
        .expect("tip receipt");
    eprintln!(
        "[tips] status={} effective_gas_price={}",
        r.status, r.effective_gas_price
    );
    assert_eq!(r.status, 1, "tipped tx did not succeed");
    assert!(
        r.effective_gas_price > 500_000_000,
        "no tip was paid (effective gas price {} not above the base fee)",
        r.effective_gas_price
    );
}

/// Native-token mint/burn parity, including the over-burn path: minting and a
/// within-balance burn must match Nitro, and a burn exceeding the balance must
/// revert having paid the membership read and the mint/burn charge (not the
/// whole gas limit), byte-for-byte with Nitro.
#[test]
#[ignore]
fn native_token_mint_burn_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    // Enabling native-token management requires a one-week delay (FeatureEnableDelay),
    // so the feature must be scheduled a week out and enrolled only once a block at
    // or after that time is produced.
    const T0: u64 = 1_700_000_000;
    const FEATURE_DELAY: u64 = 7 * 24 * 60 * 60;
    const ENABLE_TIME: u64 = T0 + FEATURE_DELAY;

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: T0,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let owner_call = |nonce: u64, to: Address, data: Vec<u8>, ts: u64| {
        let mut b = owner_tx_l1(nonce, Some(to), data, 2_000_000, 0);
        b.timestamp = ts;
        b.build().expect("owner call")
    };

    // Zero the L1 price so the receipts reflect L2 execution (the precompile
    // gas), not poster cost.
    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(i, owner_call(0, ARBOWNER, set_price, T0), 1));

    let mut enable = selector4("setNativeTokenManagementFrom(uint64)").to_vec();
    enable.extend_from_slice(&word(ENABLE_TIME));
    let i = idx.next();
    steps.push(msg_step(i, owner_call(1, ARBOWNER, enable, T0), 1));

    let mut add = selector4("addNativeTokenOwner(address)").to_vec();
    add.extend_from_slice(&word_addr(owner));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_call(2, ARBOWNER, add, ENABLE_TIME + 5),
        1,
    ));

    let mut mint = selector4("mintNativeToken(uint256)").to_vec();
    mint.extend_from_slice(&word_u256(U256::from(10u128).pow(U256::from(18u64))));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_call(3, ARBNATIVETOKENMANAGER, mint, ENABLE_TIME + 15),
        1,
    ));

    let mut burn = selector4("burnNativeToken(uint256)").to_vec();
    burn.extend_from_slice(&word_u256(U256::from(10u128).pow(U256::from(17u64))));
    let i = idx.next();
    let burn_ok = owner_call(4, ARBNATIVETOKENMANAGER, burn, ENABLE_TIME + 25);
    let burn_ok_hash = arb_test_harness::messaging::signed_l2_tx_hash(&burn_ok);
    steps.push(msg_step(i, burn_ok, 1));

    // Over-burn: amount far exceeds the balance, so the precompile reverts after
    // the membership read and mint/burn charge.
    let mut over = selector4("burnNativeToken(uint256)").to_vec();
    over.extend_from_slice(&word_u256(U256::from(10u128).pow(U256::from(30u64))));
    let i = idx.next();
    let over_tx = owner_call(5, ARBNATIVETOKENMANAGER, over, ENABLE_TIME + 35);
    let over_hash = arb_test_harness::messaging::signed_l2_tx_hash(&over_tx);
    steps.push(msg_step(i, over_tx, 1));

    let scenario = Scenario {
        name: "native_token_mint_burn".into(),
        description: "mint/burn parity incl. over-burn revert".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "native-token mint/burn diverged from Nitro: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // Sensitivity: the owner must actually be enrolled (else mint/burn take the
    // unauthorized burn-out path and the comparison is meaningless), the
    // within-balance burn must succeed, and the over-burn must revert with a
    // small gas charge (the soft revert), not consume the whole 2M limit.
    let is_owner = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBOWNERPUBLIC),
                data: Some(Bytes::from(
                    [
                        selector4("isNativeTokenOwner(address)").as_slice(),
                        &word_addr(owner),
                    ]
                    .concat(),
                )),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b) == U256::from(1u64))
        .unwrap_or(false);
    assert!(is_owner, "owner was not enrolled as a native-token owner");

    let ok_r = rig
        .dual
        .right
        .receipt(burn_ok_hash.expect("burn hash"))
        .expect("burn receipt");
    assert_eq!(ok_r.status, 1, "within-balance burn did not succeed");
    let over_r = rig
        .dual
        .right
        .receipt(over_hash.expect("over hash"))
        .expect("over receipt");
    eprintln!(
        "[native] burn_ok_gas={} over_burn_gas={}",
        ok_r.gas_used, over_r.gas_used
    );
    assert_eq!(over_r.status, 0, "over-burn did not revert");
    // The soft revert (membership read + mint/burn charge, then the balance
    // check fails before the event) costs strictly less than the full
    // successful burn. A burn-out would instead consume the whole gas limit,
    // far exceeding the successful burn — so this bounds out that path.
    assert!(
        over_r.gas_used < ok_r.gas_used,
        "over-burn ({}) did not cost less than the successful burn ({}) — \
         it took the burn-out path, not the soft revert",
        over_r.gas_used,
        ok_r.gas_used,
    );
}

/// Decode an ABI `uint256[]` return (offset, length, then words).
fn decode_uint256_array(b: &[u8]) -> Vec<U256> {
    if b.len() < 64 {
        return Vec::new();
    }
    let len = U256::from_be_slice(&b[32..64]).to::<usize>();
    (0..len)
        .filter_map(|i| {
            let s = 64 + i * 32;
            b.get(s..s + 32).map(U256::from_be_slice)
        })
        .collect()
}

/// EIP-7623 calldata floor parity. With the calldata-price increase enabled, a
/// data-heavy / compute-light transaction pays `floorDataGas` rather than its
/// (lower) execution gas. The receipt gas must match Nitro and equal the floor.
#[test]
#[ignore]
fn eip7623_calldata_floor_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    // Enable the calldata-price increase (owner-only). base_fee_l1 = 0 on every
    // message keeps the L1 price ~0 so the floor tx incurs no poster gas to
    // lift execution above the floor.
    let mut enable = selector4("setCalldataPriceIncrease(bool)").to_vec();
    enable.extend_from_slice(&word(1));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), enable, 2_000_000, 0)
            .build()
            .expect("enable"),
        1,
    ));

    // Zero the L1 price so the floor tx pays no poster gas, letting the calldata
    // floor (rather than the L1 cost) decide gas.
    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(1, Some(ARBOWNER), set_price, 2_000_000, 0)
            .build()
            .expect("set price"),
        1,
    ));

    // Floor-binding tx: many zero calldata bytes to an empty account, with no
    // L1 base fee so there is no poster gas to lift execution above the floor.
    // intrinsic = 21000 + 1000*4 = 25000; floor = 21000 + 1000*10 = 31000.
    const ZERO_BYTES: usize = 1000;
    let eoa = address!("00000000000000000000000000000000deadbeef");
    let expected_floor = 21_000u64 + (ZERO_BYTES as u64) * 10;
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(2, Some(eoa), vec![0u8; ZERO_BYTES], 2_000_000, 0)
            .build()
            .expect("floor tx"),
        1,
    ));

    let scenario = Scenario {
        name: "eip7623_calldata_floor".into(),
        description: "calldata floor parity with the price increase enabled".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "calldata floor diverged from Nitro: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // The floor tx is the latest block's last tx. Its gas must equal the floor,
    // proving the floor bound (otherwise the test is insensitive).
    let n = rig
        .dual
        .right
        .block(BlockId::Latest)
        .expect("latest")
        .number;
    let b = rig.dual.right.block(BlockId::Number(n)).expect("block");
    let hash = *b.tx_hashes.last().expect("floor tx");
    let gas = rig.dual.right.receipt(hash).expect("receipt").gas_used;
    eprintln!("[floor] block={n} floor_tx_gas={gas} expected_floor={expected_floor}");
    assert_eq!(
        gas, expected_floor,
        "floor did not bind: gas_used {gas} != floor {expected_floor}"
    );
}

const ARBWASM: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);

/// A real classic Stylus program, used to build a single-fragment root.
const ERC1155_STYLUS: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_253170068_assets/stylus_erc1155.hex"
);

/// Stylus merge-on-activate: deploy a fragment holding a program's compressed
/// WASM plus a root that references it, then activate the root. arbreth must
/// reconstruct the WASM from the fragment, charge the fragment read, and
/// produce activation gas and state matching Nitro.
#[test]
#[ignore]
fn stylus_root_activation_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(21u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    // Zero the L1 price so the large deploys carry no poster gas noise.
    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), set_price, 2_000_000, 0)
            .build()
            .expect("set price"),
        1,
    ));

    // Build a single-fragment root from the classic program.
    let classic = alloy_primitives::hex::decode(ERC1155_STYLUS.trim()).expect("hex");
    let dict = classic[3];
    let compressed = &classic[4..];
    let decompressed_len = arb_stylus::decompress_wasm(&classic)
        .expect("decompress")
        .len() as u32;

    let mut fragment = vec![0xEFu8, 0xF0, 0x01];
    fragment.extend_from_slice(compressed);
    let frag_addr = create_address(owner, 1);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(1, None, wrap_init_code(&fragment), 500_000_000, 0)
            .build()
            .expect("deploy fragment"),
        1,
    ));

    let mut root = vec![0xEFu8, 0xF0, 0x02, dict];
    root.extend_from_slice(&decompressed_len.to_be_bytes());
    root.extend_from_slice(frag_addr.as_slice());
    let root_addr = create_address(owner, 2);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(2, None, wrap_init_code(&root), 100_000_000, 0)
            .build()
            .expect("deploy root"),
        1,
    ));

    // Activate the root program (carries a data fee as value).
    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(root_addr.as_slice());
    act.extend_from_slice(&arg);
    let activate = SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce: 3,
        to: Some(ARBWASM),
        value: U256::from(10u128).pow(U256::from(18u64)),
        data: Bytes::from(act),
        gas_limit: 200_000_000,
        gas_price: 1_000_000_000,
        max_fee_per_gas: 1_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1,
        // Later timestamp so this lands in a fresh block; several messages
        // sharing one timestamp do not all get sequenced.
        timestamp: 1_700_000_010,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: 0,
    }
    .build()
    .expect("activate");
    let act_idx = idx.next();
    let act_hash = arb_test_harness::messaging::signed_l2_tx_hash(&activate);
    steps.push(msg_step(act_idx, activate, 1));

    // Call the activated root program so it dispatches to Stylus and runs the
    // reconstructed WASM (an unknown selector reverts in-program; either way the
    // call must route to Stylus and consume matching gas on both nodes).
    let mut call_b = owner_tx_l1(
        4,
        Some(root_addr),
        vec![0x00, 0x00, 0x00, 0x00],
        50_000_000,
        0,
    );
    call_b.timestamp = 1_700_000_020;
    let call = call_b.build().expect("call root");
    let call_idx = idx.next();
    let call_hash = arb_test_harness::messaging::signed_l2_tx_hash(&call);
    steps.push(msg_step(call_idx, call, 1));

    let scenario = Scenario {
        name: "stylus_root_activation".into(),
        description: "merge-on-activate fragment reconstruction parity".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "root activation diverged from Nitro: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // Sensitivity: the activation must have actually run (a root that failed to
    // reconstruct would revert), so it must have consumed real gas on both nodes.
    // The fragment and root must have deployed, and the activation must have
    // executed — a failed reconstruction would leave no code or revert.
    let frag_deployed = rig
        .dual
        .right
        .code(frag_addr, BlockId::Latest)
        .is_ok_and(|c| !c.is_empty());
    let root_deployed = rig
        .dual
        .right
        .code(root_addr, BlockId::Latest)
        .is_ok_and(|c| !c.is_empty());
    assert!(
        frag_deployed && root_deployed,
        "fragment/root did not deploy"
    );
    let hash = act_hash.expect("activate tx hash");
    let gas = rig
        .dual
        .right
        .receipt(hash)
        .map(|r| r.gas_used)
        .unwrap_or(0);
    let chash = call_hash.expect("call tx hash");
    let cgas = rig
        .dual
        .right
        .receipt(chash)
        .map(|r| r.gas_used)
        .unwrap_or(0);
    eprintln!("[root] activation gas = {gas}, call gas = {cgas}");
    assert!(gas > 100_000, "root activation did not execute (gas {gas})");
    assert!(
        cgas > 21_000,
        "call did not dispatch to the Stylus program (gas {cgas})"
    );
}

/// A root whose declared decompressed length does not match what its fragments
/// actually decompress to must be rejected at activation, byte-for-byte with
/// Nitro — the activation reverts on both nodes rather than producing an
/// activated program on one and a revert on the other.
#[test]
#[ignore]
fn stylus_root_length_mismatch_reverts_on_both() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(21u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), set_price, 2_000_000, 0)
            .build()
            .expect("set price"),
        1,
    ));

    let classic = alloy_primitives::hex::decode(ERC1155_STYLUS.trim()).expect("hex");
    let dict = classic[3];
    let compressed = &classic[4..];
    let decompressed_len = arb_stylus::decompress_wasm(&classic)
        .expect("decompress")
        .len() as u32;

    let mut fragment = vec![0xEFu8, 0xF0, 0x01];
    fragment.extend_from_slice(compressed);
    let frag_addr = create_address(owner, 1);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(1, None, wrap_init_code(&fragment), 500_000_000, 0)
            .build()
            .expect("deploy fragment"),
        1,
    ));

    // Root declares a decompressed length one byte longer than the fragment
    // actually yields, so reconstruction must reject it.
    let mut root = vec![0xEFu8, 0xF0, 0x02, dict];
    root.extend_from_slice(&(decompressed_len + 1).to_be_bytes());
    root.extend_from_slice(frag_addr.as_slice());
    let root_addr = create_address(owner, 2);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(2, None, wrap_init_code(&root), 100_000_000, 0)
            .build()
            .expect("deploy root"),
        1,
    ));

    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(root_addr.as_slice());
    act.extend_from_slice(&arg);
    let activate = SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce: 3,
        to: Some(ARBWASM),
        value: U256::from(10u128).pow(U256::from(18u64)),
        data: Bytes::from(act),
        gas_limit: 200_000_000,
        gas_price: 1_000_000_000,
        max_fee_per_gas: 1_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1,
        timestamp: 1_700_000_010,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: 0,
    }
    .build()
    .expect("activate");
    let act_idx = idx.next();
    let act_hash = arb_test_harness::messaging::signed_l2_tx_hash(&activate);
    steps.push(msg_step(act_idx, activate, 1));

    let scenario = Scenario {
        name: "stylus_root_length_mismatch".into(),
        description: "mismatched root length reverts identically on both nodes".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "mismatched root activation diverged from Nitro: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    // The bad root must deploy (deploy-time only checks the prefix) and the
    // activation must revert — proving both nodes reject the mismatch rather
    // than one activating it.
    let root_deployed = rig
        .dual
        .right
        .code(root_addr, BlockId::Latest)
        .is_ok_and(|c| !c.is_empty());
    assert!(root_deployed, "bad root did not deploy");
    let r = rig
        .dual
        .right
        .receipt(act_hash.expect("activate hash"))
        .expect("activate receipt");
    eprintln!(
        "[mismatch] activation status={} gas={}",
        r.status, r.gas_used
    );
    assert_eq!(
        r.status, 0,
        "activation of a length-mismatched root must revert"
    );
}

/// WasmComputation resource kind (`ResourceKind` discriminant).
const KIND_WASM_COMPUTATION: u8 = 8;

/// A Stylus program's execution gas must reach the WasmComputation dimension, so
/// a constraint weighting it escalates the base fee identically on both nodes.
/// Lumping Stylus gas into computation would leave the WasmComputation backlog
/// flat and diverge.
#[test]
#[ignore]
fn stylus_wasm_computation_constraint_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(21u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), set_price, 2_000_000, 0)
            .build()
            .expect("set price"),
        1,
    ));

    let cons = set_constraint_calldata(60, 100_000, 0, KIND_WASM_COMPUTATION, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(1, Some(ARBOWNER), cons, 2_000_000, 0)
            .build()
            .expect("set constraint"),
        1,
    ));

    let classic = alloy_primitives::hex::decode(ERC1155_STYLUS.trim()).expect("hex");
    let dict = classic[3];
    let compressed = &classic[4..];
    let decompressed_len = arb_stylus::decompress_wasm(&classic)
        .expect("decompress")
        .len() as u32;

    let mut fragment = vec![0xEFu8, 0xF0, 0x01];
    fragment.extend_from_slice(compressed);
    let frag_addr = create_address(owner, 2);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(2, None, wrap_init_code(&fragment), 500_000_000, 0)
            .build()
            .expect("deploy fragment"),
        1,
    ));

    let mut root = vec![0xEFu8, 0xF0, 0x02, dict];
    root.extend_from_slice(&decompressed_len.to_be_bytes());
    root.extend_from_slice(frag_addr.as_slice());
    let root_addr = create_address(owner, 3);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(3, None, wrap_init_code(&root), 100_000_000, 0)
            .build()
            .expect("deploy root"),
        1,
    ));

    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(root_addr.as_slice());
    act.extend_from_slice(&arg);
    let mut activate_b = owner_tx_l1(4, Some(ARBWASM), act, 200_000_000, 0);
    activate_b.value = U256::from(10u128).pow(U256::from(18u64));
    activate_b.timestamp = 1_700_000_010;
    let act_idx = idx.next();
    steps.push(msg_step(act_idx, activate_b.build().expect("activate"), 1));

    // Repeated calls dispatch to the Stylus program, each consuming WASM gas that
    // must grow the WasmComputation backlog.
    let mut nonce = 5u64;
    let mut ts = 1_700_000_020u64;
    for _ in 0..8 {
        let mut call_b = owner_tx_l1(
            nonce,
            Some(root_addr),
            vec![0x00, 0x00, 0x00, 0x00],
            50_000_000,
            0,
        );
        call_b.timestamp = ts;
        let i = idx.next();
        steps.push(msg_step(i, call_b.build().expect("call root"), 1));
        nonce += 1;
        ts += 10;
    }

    let scenario = Scenario {
        name: "stylus_wasm_computation_constraint".into(),
        description: "Stylus execution gas grows the WasmComputation backlog".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "Stylus diverged under active WasmComputation constraint: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    let root_deployed = rig
        .dual
        .right
        .code(root_addr, BlockId::Latest)
        .is_ok_and(|c| !c.is_empty());
    assert!(root_deployed, "root did not deploy");

    let base_fee_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let l = rig
        .dual
        .left
        .eth_call(base_fee_call.clone(), BlockId::Latest)
        .ok();
    let r = rig.dual.right.eth_call(base_fee_call, BlockId::Latest).ok();
    assert_eq!(
        l, r,
        "getMultiGasBaseFee diverged: nitro={l:?} arbreth={r:?}"
    );

    let fees = decode_uint256_array(&r.expect("base fee bytes"));
    let floor = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBGASINFO),
                data: Some(Bytes::from(selector4("getMinimumGasPrice()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b))
        .unwrap_or(U256::ZERO);
    // Only WasmComputation is weighted, so it is the sole dimension whose fee can
    // escalate; the max rising above the floor proves Stylus gas reached it.
    let max_fee = fees.iter().copied().max().unwrap_or(U256::ZERO);
    eprintln!("[stylus-mgas] floor={floor} max_fee={max_fee} fees={fees:?}");
    assert!(
        max_fee > floor,
        "WasmComputation base fee did not escalate above floor {floor}; Stylus gas \
         never reached the WasmComputation dimension"
    );
}

/// `forward(address,bytes)` calldata for the `sol_caller` Stylus program: it
/// `CALL`s `target` with `inner`.
fn forward_calldata(target: Address, inner: &[u8]) -> Vec<u8> {
    let mut out = selector4("forward(address,bytes)").to_vec();
    out.extend_from_slice(&word_addr(target));
    out.extend_from_slice(&word(0x40));
    out.extend_from_slice(&word(inner.len() as u64));
    out.extend_from_slice(inner);
    while !out.len().is_multiple_of(32) {
        out.push(0);
    }
    out
}

/// A Stylus program calling a Solidity contract that writes storage must
/// attribute the callee's SSTORE gas to StorageAccessWrite, identically to
/// Nitro. Without per-frame propagation the sub-call gas falls into computation
/// and the write backlog diverges.
#[test]
#[ignore]
fn stylus_to_solidity_write_constraint_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(21u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), set_price, 2_000_000, 0)
            .build()
            .expect("set price"),
        1,
    ));

    let cons = set_constraint_calldata(60, 100_000, 0, KIND_STORAGE_WRITE, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(1, Some(ARBOWNER), cons, 2_000_000, 0)
            .build()
            .expect("set constraint"),
        1,
    ));

    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(2, None, WhichProgram::SolCaller.initcode(), 500_000_000, 0)
            .build()
            .expect("deploy sol_caller"),
        1,
    ));
    let sol_caller = create_address(owner, 2);

    let mut act = selector4("activateProgram(address)").to_vec();
    act.extend_from_slice(&word_addr(sol_caller));
    let mut activate_b = owner_tx_l1(3, Some(ARBWASM), act, 200_000_000, 0);
    activate_b.value = U256::from(10u128).pow(U256::from(18u64));
    activate_b.timestamp = 1_700_000_010;
    let act_idx = idx.next();
    steps.push(msg_step(act_idx, activate_b.build().expect("activate"), 1));

    let mut helper_b = owner_tx_l1(4, None, wrap_init_code(&SSTORE_RUNTIME), 5_000_000, 0);
    helper_b.timestamp = 1_700_000_020;
    let i = idx.next();
    steps.push(msg_step(i, helper_b.build().expect("deploy helper"), 1));
    let helper = create_address(owner, 4);

    // First write creates slot 0; the rest are resets pricing into
    // StorageAccessWrite, reached through the Stylus -> Solidity sub-call.
    let mut nonce = 5u64;
    let mut ts = 1_700_000_030u64;
    for v in [1u64, 2, 3, 4] {
        let cd = forward_calldata(helper, &word(v));
        let mut b = owner_tx_l1(nonce, Some(sol_caller), cd, 50_000_000, 0);
        b.timestamp = ts;
        let i = idx.next();
        steps.push(msg_step(i, b.build().expect("forward"), 1));
        nonce += 1;
        ts += 10;
    }

    let scenario = Scenario {
        name: "stylus_to_solidity_write".into(),
        description: "Stylus -> Solidity SSTORE grows the write backlog".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "Stylus -> Solidity diverged under active write constraint: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    let sc_code = rig
        .dual
        .right
        .code(sol_caller, BlockId::Latest)
        .expect("code");
    assert!(!sc_code.is_empty(), "sol_caller did not deploy");
    let helper_code = rig.dual.right.code(helper, BlockId::Latest).expect("code");
    assert!(!helper_code.is_empty(), "helper did not deploy");

    let base_fee_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let l = rig
        .dual
        .left
        .eth_call(base_fee_call.clone(), BlockId::Latest)
        .ok();
    let r = rig.dual.right.eth_call(base_fee_call, BlockId::Latest).ok();
    assert_eq!(
        l, r,
        "getMultiGasBaseFee diverged: nitro={l:?} arbreth={r:?}"
    );

    let fees = decode_uint256_array(&r.expect("base fee bytes"));
    let floor = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBGASINFO),
                data: Some(Bytes::from(selector4("getMinimumGasPrice()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b))
        .unwrap_or(U256::ZERO);
    let max_fee = fees.iter().copied().max().unwrap_or(U256::ZERO);
    eprintln!("[stylus->sol] floor={floor} max_fee={max_fee} fees={fees:?}");
    assert!(
        max_fee > floor,
        "write base fee did not escalate; the Stylus -> Solidity SSTORE gas never \
         reached StorageAccessWrite"
    );
}

/// An ArbOwner setter executed repeatedly under a write-weighted constraint must
/// track the reference: its ArbOS state writes do not grow the per-resource write
/// backlog, so the base fee stays at the floor and every call remains affordable.
#[test]
#[ignore]
fn precompile_write_constraint_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let cons = set_constraint_calldata(60, 100_000, 0, KIND_STORAGE_WRITE, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), cons, 2_000_000, 0)
            .build()
            .expect("set constraint"),
        1,
    ));

    let mut nonce = 1u64;
    let mut ts = 1_700_000_010u64;
    for v in [1u64, 2, 3, 4, 5, 6, 7, 8] {
        let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
        set_price.extend_from_slice(&word(v));
        let mut b = owner_tx_l1(nonce, Some(ARBOWNER), set_price, 2_000_000, 0);
        b.timestamp = ts;
        let i = idx.next();
        steps.push(msg_step(i, b.build().expect("set price"), 1));
        nonce += 1;
        ts += 10;
    }

    let scenario = Scenario {
        name: "precompile_write_constraint".into(),
        description: "ArbOwner setter SSTORE grows the write backlog".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "precompile write diverged under active write constraint: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    let nonce_after = rig.dual.right.nonce(owner, BlockId::Latest).expect("nonce");
    assert!(
        nonce_after >= 9,
        "owner setter txs did not all execute (nonce {nonce_after})"
    );

    let base_fee_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let l = rig
        .dual
        .left
        .eth_call(base_fee_call.clone(), BlockId::Latest)
        .ok();
    let r = rig.dual.right.eth_call(base_fee_call, BlockId::Latest).ok();
    assert_eq!(
        l, r,
        "getMultiGasBaseFee diverged: nitro={l:?} arbreth={r:?}"
    );
    let fees = decode_uint256_array(&r.expect("base fee bytes"));
    let floor = fees.first().copied().unwrap_or(U256::ZERO);
    assert!(
        fees.iter().all(|f| *f == floor),
        "a resource base fee escalated; owner setter writes must not grow the backlog: {fees:?}"
    );

    let price_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getL1BaseFeeEstimate()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let price = rig
        .dual
        .right
        .eth_call(price_call, BlockId::Latest)
        .expect("price call");
    let mut buf = [0u8; 32];
    buf.copy_from_slice(&price[..32]);
    let stored = U256::from_be_bytes(buf);
    assert_eq!(
        stored,
        U256::from(8u64),
        "setL1PricePerUnit setters did not take effect (stored={stored}); the test is hollow"
    );
}

/// A non-owner state-reading precompile (`ArbAddressTable.addressExists`)
/// charges two storage reads per call; under a read-weighted constraint the
/// reference grows the per-resource read backlog and the read base fee
/// escalates. arbreth must do the same — lumping precompile reads into
/// computation makes the read backlog stay flat and diverge from the
/// reference's escalation.
#[test]
#[ignore]
fn precompile_read_constraint_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    let cons = set_constraint_calldata(60, 100_000, 0, KIND_STORAGE_READ, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx_l1(0, Some(ARBOWNER), cons, 2_000_000, 0)
            .build()
            .expect("set constraint"),
        1,
    ));

    let probe = Address::new([0xbe; 20]);
    let mut probe_word = [0u8; 32];
    probe_word[12..].copy_from_slice(probe.as_slice());

    let mut nonce = 1u64;
    let mut ts = 1_700_000_010u64;
    for _ in 0..8u32 {
        let mut data = selector4("addressExists(address)").to_vec();
        data.extend_from_slice(&probe_word);
        let mut b = owner_tx_l1(nonce, Some(ARBADDRESSTABLE), data, 2_000_000, 0);
        b.timestamp = ts;
        let i = idx.next();
        steps.push(msg_step(i, b.build().expect("addressExists"), 1));
        nonce += 1;
        ts += 10;
    }

    let scenario = Scenario {
        name: "precompile_read_constraint".into(),
        description: "ArbAddressTable.addressExists feeds the read backlog".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "non-owner precompile reads diverged under active read constraint: blocks={:?} txs={:?}",
        report.block_diffs,
        report.tx_diffs,
    );

    let nonce_after = rig.dual.right.nonce(owner, BlockId::Latest).expect("nonce");
    assert!(
        nonce_after >= 2,
        "no addressExists tx executed (nonce {nonce_after}); the test is hollow"
    );

    let base_fee_call = TxRequest {
        from: Some(owner),
        to: Some(ARBGASINFO),
        data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
        value: Some(U256::ZERO),
        gas: Some(3_000_000),
    };
    let l = rig
        .dual
        .left
        .eth_call(base_fee_call.clone(), BlockId::Latest)
        .ok();
    let r = rig.dual.right.eth_call(base_fee_call, BlockId::Latest).ok();
    assert_eq!(
        l, r,
        "getMultiGasBaseFee diverged: nitro={l:?} arbreth={r:?}"
    );
    let fees = decode_uint256_array(&r.expect("base fee bytes"));
    let floor = fees.first().copied().unwrap_or(U256::ZERO);
    let read_fee = fees
        .get(KIND_STORAGE_READ as usize)
        .copied()
        .unwrap_or(U256::ZERO);
    assert!(
        read_fee > floor,
        "read base fee did not escalate; non-owner precompile reads must feed StorageAccessRead: {fees:?}"
    );
}
