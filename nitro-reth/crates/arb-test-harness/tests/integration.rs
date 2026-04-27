use std::time::Duration;

use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    mock_l1::MockL1,
    node::{arbreth::ArbrethProcess, nitro_local::NitroProcess, BlockId, NodeStartCtx},
    rpc::JsonRpcClient,
    scenario::{Scenario, ScenarioSetup},
    ExecutionNode,
};

const TEST_L2_CHAIN_ID: u64 = 421614;
const TEST_L1_CHAIN_ID: u64 = 0xaa36a7;

#[test]
fn mock_l1_serves_chain_id() {
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let client = JsonRpcClient::new(mock.rpc_url());
    let v = client
        .call("eth_chainId", serde_json::json!([]))
        .expect("eth_chainId on mock");
    assert_eq!(v.as_str().unwrap(), format!("0x{:x}", TEST_L1_CHAIN_ID));

    let bn = client
        .call("eth_blockNumber", serde_json::json!([]))
        .expect("eth_blockNumber on mock");
    assert_eq!(bn.as_str().unwrap(), "0x1");

    let net = client
        .call("net_version", serde_json::json!([]))
        .expect("net_version on mock");
    assert_eq!(net.as_str().unwrap(), TEST_L1_CHAIN_ID.to_string());

    let call = client
        .call(
            "eth_call",
            serde_json::json!([{"to": "0x0000000000000000000000000000000000000000"}, "latest"]),
        )
        .expect("eth_call on mock");
    assert_eq!(call.as_str().unwrap(), "0x");

    let receipt = client
        .call(
            "eth_getTransactionReceipt",
            serde_json::json!(["0x0000000000000000000000000000000000000000000000000000000000000000"]),
        )
        .expect("getTransactionReceipt on mock");
    assert!(receipt.is_null());

    mock.shutdown().expect("clean mock shutdown");
}

#[test]
fn mock_l1_advances_block_number() {
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let client = JsonRpcClient::new(mock.rpc_url());
    let before = client
        .call("eth_blockNumber", serde_json::json!([]))
        .expect("before");
    let _ = mock.advance_block();
    let _ = mock.advance_block();
    let after = client
        .call("eth_blockNumber", serde_json::json!([]))
        .expect("after");
    let parse = |v: &serde_json::Value| {
        u64::from_str_radix(v.as_str().unwrap().trim_start_matches("0x"), 16).unwrap()
    };
    assert_eq!(parse(&after) - parse(&before), 2);
}

#[test]
#[ignore]
fn nitro_chain_id_round_trip() {
    if std::env::var("NITRO_REF_BINARY").is_err() {
        eprintln!("skip: NITRO_REF_BINARY unset");
        return;
    }
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let genesis = GenesisBuilder::new(TEST_L2_CHAIN_ID, 10)
        .build()
        .expect("build genesis");
    let ctx = NodeStartCtx {
        binary: None,
        l2_chain_id: TEST_L2_CHAIN_ID,
        l1_chain_id: TEST_L1_CHAIN_ID,
        mock_l1_rpc: mock.rpc_url(),
        genesis,
        jwt_hex: String::new(),
        workdir: std::path::PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    };

    let node = NitroProcess::start(&ctx).expect("nitro startup");
    let chain_id = JsonRpcClient::new(node.rpc_url())
        .with_timeout(Duration::from_secs(10))
        .call("eth_chainId", serde_json::json!([]))
        .expect("nitro eth_chainId");
    let parsed = u64::from_str_radix(
        chain_id.as_str().unwrap().trim_start_matches("0x"),
        16,
    )
    .expect("hex chain id");
    assert_eq!(parsed, TEST_L2_CHAIN_ID);

    let block = node.block(BlockId::Latest).expect("latest block");
    assert!(block.gas_limit > 0);

    Box::new(node).shutdown().expect("shutdown nitro");
    mock.shutdown().expect("shutdown mock");
}

#[test]
#[ignore]
fn arbreth_chain_id_round_trip() {
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skip: ARB_SPEC_BINARY unset");
        return;
    }
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let genesis = GenesisBuilder::new(TEST_L2_CHAIN_ID, 10)
        .build()
        .expect("build genesis");
    let ctx = NodeStartCtx {
        binary: None,
        l2_chain_id: TEST_L2_CHAIN_ID,
        l1_chain_id: TEST_L1_CHAIN_ID,
        mock_l1_rpc: mock.rpc_url(),
        genesis,
        jwt_hex: String::new(),
        workdir: std::path::PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    };

    let node = ArbrethProcess::start(&ctx).expect("arbreth startup");
    let chain_id = JsonRpcClient::new(node.rpc_url())
        .with_timeout(Duration::from_secs(10))
        .call("eth_chainId", serde_json::json!([]))
        .expect("arbreth eth_chainId");
    let parsed = u64::from_str_radix(
        chain_id.as_str().unwrap().trim_start_matches("0x"),
        16,
    )
    .expect("hex chain id");
    assert_eq!(parsed, TEST_L2_CHAIN_ID);

    let block = node.block(BlockId::Latest).expect("latest block");
    assert!(block.gas_limit > 0);

    Box::new(node).shutdown().expect("shutdown arbreth");
    mock.shutdown().expect("shutdown mock");
}

#[test]
#[ignore]
fn dual_node_chain_ids_match() {
    if std::env::var("NITRO_REF_BINARY").is_err() || std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skip: NITRO_REF_BINARY and/or ARB_SPEC_BINARY unset");
        return;
    }
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let genesis = GenesisBuilder::new(TEST_L2_CHAIN_ID, 10)
        .build()
        .expect("build genesis");
    let make_ctx = || NodeStartCtx {
        binary: None,
        l2_chain_id: TEST_L2_CHAIN_ID,
        l1_chain_id: TEST_L1_CHAIN_ID,
        mock_l1_rpc: mock.rpc_url(),
        genesis: genesis.clone(),
        jwt_hex: String::new(),
        workdir: std::path::PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    };

    let nitro = NitroProcess::start(&make_ctx()).expect("nitro startup");
    let arb = ArbrethProcess::start(&make_ctx()).expect("arbreth startup");

    let nitro_id = JsonRpcClient::new(nitro.rpc_url())
        .call("eth_chainId", serde_json::json!([]))
        .expect("nitro eth_chainId");
    let arb_id = JsonRpcClient::new(arb.rpc_url())
        .call("eth_chainId", serde_json::json!([]))
        .expect("arbreth eth_chainId");
    assert_eq!(nitro_id, arb_id);

    Box::new(nitro).shutdown().expect("shutdown nitro");
    Box::new(arb).shutdown().expect("shutdown arb");
    mock.shutdown().expect("shutdown mock");
}

#[test]
#[ignore]
fn dual_exec_runs_empty_scenario() {
    if std::env::var("NITRO_REF_BINARY").is_err() || std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skip: NITRO_REF_BINARY and/or ARB_SPEC_BINARY unset");
        return;
    }
    let mock = MockL1::start(TEST_L1_CHAIN_ID).expect("start mock L1");
    let genesis = GenesisBuilder::new(TEST_L2_CHAIN_ID, 10)
        .build()
        .expect("build genesis");
    let make_ctx = || NodeStartCtx {
        binary: None,
        l2_chain_id: TEST_L2_CHAIN_ID,
        l1_chain_id: TEST_L1_CHAIN_ID,
        mock_l1_rpc: mock.rpc_url(),
        genesis: genesis.clone(),
        jwt_hex: String::new(),
        workdir: std::path::PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    };

    let nitro = NitroProcess::start(&make_ctx()).expect("nitro startup");
    let arb = ArbrethProcess::start(&make_ctx()).expect("arbreth startup");

    let scenario = Scenario {
        name: "empty".into(),
        description: "no messages, both nodes should agree on the empty chain".into(),
        setup: ScenarioSetup {
            l2_chain_id: TEST_L2_CHAIN_ID,
            arbos_version: 10,
            genesis: None,
        },
        steps: Vec::new(),
    };

    let mut dual = DualExec::new(nitro, arb);
    let report = dual.run(&scenario).expect("dual_exec run");
    assert!(
        report.is_clean(),
        "dual_exec produced unexpected diffs: {report:?}",
    );

    Box::new(dual.left).shutdown().expect("shutdown nitro");
    Box::new(dual.right).shutdown().expect("shutdown arb");
    mock.shutdown().expect("shutdown mock");
}
