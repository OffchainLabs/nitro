use arb_test_harness::{mock_l1::MockL1, rpc::JsonRpcClient};

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
            serde_json::json!([
                "0x0000000000000000000000000000000000000000000000000000000000000000"
            ]),
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
