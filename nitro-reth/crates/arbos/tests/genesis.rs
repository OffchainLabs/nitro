use alloy_primitives::{address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::{
    arbos_state::initialize::{
        initialize_retryables, make_genesis_block, AccountInitInfo, AggregatorInitInfo,
        ContractInitInfo, GenesisBlockInfo, InitRetryableData,
    },
    retryables::retryable_escrow_address,
};

#[test]
fn make_genesis_block_returns_well_formed_struct() {
    let g: GenesisBlockInfo =
        make_genesis_block(B256::ZERO, 0, 1_700_000_000, B256::repeat_byte(0xAA), 30);
    assert_eq!(g.parent_hash, B256::ZERO);
    assert_eq!(g.block_number, 0);
    assert_eq!(g.timestamp, 1_700_000_000);
    assert_eq!(g.state_root, B256::repeat_byte(0xAA));
    assert_eq!(g.nonce, 1);
    assert_eq!(g.arbos_format_version, 30);
    assert!(g.base_fee > 0);
    assert!(g.gas_limit > 0);
}

#[test]
fn initialize_retryables_skips_expired() {
    let mut h = ArbosHarness::new().initialize();
    let rs = h.retryable_state();

    let alive = InitRetryableData {
        id: B256::repeat_byte(0x11),
        timeout: 2_000,
        from: address!("00000000000000000000000000000000000A11CE"),
        to: None,
        callvalue: U256::from(1_000u64),
        beneficiary: address!("00000000000000000000000000000000000B0B00"),
        calldata: Vec::new(),
    };
    let expired = InitRetryableData {
        id: B256::repeat_byte(0x22),
        timeout: 500,
        from: address!("00000000000000000000000000000000000A11CE"),
        to: None,
        callvalue: U256::from(99_999u64),
        beneficiary: address!("0000000000000000000000000000000000DEAD00"),
        calldata: Vec::new(),
    };

    let (balance_credits, escrow_credits) =
        initialize_retryables(&rs, vec![alive.clone(), expired.clone()], 1_000).unwrap();

    assert_eq!(balance_credits.len(), 1);
    assert_eq!(balance_credits[0], (expired.beneficiary, expired.callvalue));
    assert_eq!(escrow_credits.len(), 1);
    assert_eq!(escrow_credits[0].0, retryable_escrow_address(alive.id));
    assert_eq!(escrow_credits[0].1, alive.callvalue);
}

#[test]
fn account_init_info_round_trips_through_construction() {
    let info = AccountInitInfo {
        addr: address!("00000000000000000000000000000000000A11CE"),
        nonce: 5,
        balance: U256::from(123_456u64),
        contract_info: Some(ContractInitInfo {
            code: vec![0x60, 0x60, 0x60, 0x40],
            storage: vec![(U256::from(1u64), U256::from(2u64))],
        }),
        aggregator_info: Some(AggregatorInitInfo {
            fee_collector: address!("00000000000000000000000000000000FEEEEEEE"),
        }),
    };
    assert_eq!(info.nonce, 5);
    assert_eq!(info.balance, U256::from(123_456u64));
    assert!(info.contract_info.is_some());
    assert_eq!(
        info.contract_info.unwrap().storage[0],
        (U256::from(1u64), U256::from(2u64))
    );
}

#[test]
fn sepolia_genesis_file_parses() {
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("..")
        .join("genesis")
        .join("arbitrum-sepolia.json");
    let bytes = std::fs::read(&path).expect("read genesis file");
    let json: serde_json::Value = serde_json::from_slice(&bytes).expect("parse JSON");
    assert_eq!(json["config"]["chainId"], 421614);
    assert!(json["alloc"].is_object());
    let alloc = json["alloc"].as_object().unwrap();
    let arb_state = alloc
        .get("a4b05fffffffffffffffffffffffffffffffffff")
        .expect("ArbOS state account present");
    assert_eq!(arb_state["nonce"], "0x1");
    assert!(arb_state["storage"].is_object());
}
