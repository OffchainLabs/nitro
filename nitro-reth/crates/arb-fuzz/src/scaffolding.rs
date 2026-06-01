use alloy_primitives::{keccak256, Address, Bytes, U256};
use arb_test_harness::{
    messaging::{
        signed_tx::{L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::ScenarioStep,
};

use crate::{
    arbitrary_impls::{
        interop::{create_address, interop_eoa, interop_signing_key, WhichProgram},
        message_step,
    },
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};

pub const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
pub const INVOKE_GAS_CAP: u64 = 30_000_000;
pub const DEPLOY_GAS_CAP: u64 = 1_000_000_000;
pub const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x73, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63, 0x65, 0x72,
]);
pub const ARBWASM_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);
pub const ARBSYS_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x64,
]);

pub fn selector4(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

pub fn signed(
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
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: interop_signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

pub fn wrap_two_arg(sig: &str, addr: Address, data: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 96 + data.len());
    out.extend_from_slice(&selector4(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    out.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(data);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

pub fn eoa_create_addr(nonce: u64) -> Address {
    create_address(interop_eoa(), nonce)
}

pub fn fund_interop_eoa(steps: &mut Vec<ScenarioStep>) {
    let eoa = interop_eoa();
    let idx = next_msg_idx();
    let msg = DepositBuilder {
        from: eoa,
        to: eoa,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund");
    steps.push(message_step(idx, msg, idx));
}

pub fn deploy_solcaller(steps: &mut Vec<ScenarioStep>, nonce: u64) -> Address {
    let tx = signed(
        nonce,
        None,
        Bytes::from(WhichProgram::SolCaller.initcode()),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy SolCaller");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    eoa_create_addr(nonce)
}

pub fn activate_program(steps: &mut Vec<ScenarioStep>, nonce: u64, addr: Address) {
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    data.extend_from_slice(&pad);
    let tx = signed(
        nonce,
        Some(ARBWASM_ADDR),
        Bytes::from(data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
}

pub fn deploy_solidity(steps: &mut Vec<ScenarioStep>, nonce: u64, runtime: &[u8]) -> Address {
    let init = crate::arbitrary_impls::interop::wrap_init_code(runtime);
    let tx = signed(nonce, None, Bytes::from(init), U256::ZERO, DEPLOY_GAS_CAP)
        .build()
        .expect("deploy solidity");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    eoa_create_addr(nonce)
}

pub fn baseline_stylus_plus_helper(helper_runtime: &[u8]) -> (Vec<ScenarioStep>, Address, Address) {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let stylus = deploy_solcaller(&mut steps, 0);
    activate_program(&mut steps, 1, stylus);
    let helper = deploy_solidity(&mut steps, 2, helper_runtime);
    (steps, stylus, helper)
}
