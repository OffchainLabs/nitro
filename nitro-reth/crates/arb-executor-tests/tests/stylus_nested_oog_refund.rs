//! A Stylus upfront-abort under a Stylus ancestor must be dimensioned (the
//! ancestor attributes its gas), so no v60 refund is due. A Stylus caller
//! forwards through an EVM forwarder into a fresh callee capped below its
//! upfront cost; the sender's net charge must stay `base_fee * gas_used`.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_consensus::{transaction::Recovered, TxLegacy};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, hex, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::{
    config::ArbEvmConfig,
    multi_gas::{MultiGasInspector, MultiGasSink},
};
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::{ArbosHarness, EmptyDb};
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::State, primitives::hardfork::SpecId};

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const BLOCK_NUMBER: u64 = 1_000;
const BLOCK_TIMESTAMP: u64 = 1_781_272_123;
const BASE_FEE: u128 = 100_000_000;
const L1_BLOCK_NUMBER: u64 = 0x100;
const PARENT_HASH: B256 = B256::ZERO;
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");
const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");
const NETWORK_FEE_ACCOUNT: Address = address!("71b61c2e250afa05dfc36304d6c91501be0965d8");

const SENDER: Address = address!("d708654494bc0aa70d85579b16471b635b38a304");
const FORWARDER: Address = address!("00000000000000000000000000000000f0f0f0f0");
const PROGRAM: Address = address!("2dc1bad4e0a3af9acf003d65dea54dc568e787d2");
const CALLER: Address = address!("00000000000000000000000000000000cafecafe");

const PROGRAM_HEX: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/stylus_nested_oog/program.hex"
));
const CALLER_HEX: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/stylus_nested_oog/sol_caller.hex"
));

/// Forwarded gas cap, well below the program's upfront cost.
const GAS_CAP: u16 = 8_000;

fn hb(s: &str) -> Vec<u8> {
    hex::decode(s.trim().trim_start_matches("0x")).unwrap()
}

/// Runtime that copies its calldata to memory, CALLs `target` with `gas_cap`
/// and a zero value, pops the result and STOPs — committing the outer tx
/// regardless of the inner outcome.
fn forwarder_runtime(target: Address, gas_cap: u16) -> Vec<u8> {
    let mut c = Vec::new();
    c.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]); // CALLDATACOPY(0,0,CALLDATASIZE)
    c.extend_from_slice(&[0x60, 0x00]); // retLen
    c.extend_from_slice(&[0x60, 0x00]); // retOff
    c.push(0x36); // argLen = CALLDATASIZE
    c.extend_from_slice(&[0x60, 0x00]); // argOff
    c.extend_from_slice(&[0x60, 0x00]); // value
    c.push(0x73); // PUSH20 addr
    c.extend_from_slice(target.as_slice());
    c.push(0x61); // PUSH2 gas_cap
    c.extend_from_slice(&gas_cap.to_be_bytes());
    c.push(0xf1); // CALL
    c.push(0x50); // POP
    c.push(0x00); // STOP
    c
}

fn seed_l1_price_zero(state: &mut State<EmptyDb>) {
    use revm::database::states::bundle_state::BundleRetention;
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: no other borrow of the backing state is live for this write.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        arb_state
            .l1_pricing_state
            .set_price_per_unit(backend, U256::ZERO)
            .expect("set l1 price");
        let l2 = &arb_state.l2_pricing_state;
        l2.set_min_base_fee_wei(backend, U256::from(BASE_FEE))
            .expect("min base fee");
        l2.set_base_fee_wei(backend, U256::from(BASE_FEE))
            .expect("base fee");
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn activate_and_seed_program(state: &mut State<EmptyDb>, code: &[u8], version: u16) {
    use revm::database::states::bundle_state::BundleRetention;
    let code_hash = alloy_primitives::keccak256(code);
    let wasm = arb_stylus::decompress_wasm(code).expect("decompress");
    let mut gas = u64::MAX;
    let activation = arb_stylus::activate_program(
        &wasm,
        code_hash.as_ref(),
        version,
        ARBOS_VERSION,
        u16::MAX,
        false,
        &mut gas,
    )
    .expect("activate");
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: no other borrow of the backing state is live for this write.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        let mut params = arb_state.programs.params(backend).expect("load params");
        while params.version < version {
            let next = params.version + 1;
            params.upgrade_to_version(next).expect("upgrade params");
        }
        arb_state
            .programs
            .save_params(backend, &params)
            .expect("save params");
        arb_state
            .programs
            .set_module_hash(backend, code_hash, activation.module_hash)
            .expect("set module hash");
        arb_state
            .programs
            .set_program(
                backend,
                code_hash,
                arbos::programs::Program {
                    version,
                    init_cost: activation.init_gas,
                    cached_cost: activation.cached_init_gas,
                    footprint: activation.footprint,
                    asm_estimate_kb: activation.asm_estimate.div_ceil(1024),
                    activated_at: arbos::programs::hours_since_arbitrum(BLOCK_TIMESTAMP),
                    age_seconds: 0,
                    cached: false,
                },
            )
            .expect("set program");
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn read_balance(state: &mut State<EmptyDb>, a: Address) -> U256 {
    state
        .cache
        .accounts
        .get(&a)
        .and_then(|c| c.account.as_ref())
        .map(|x| x.info.balance)
        .unwrap_or(U256::ZERO)
}

#[test]
fn nested_upfront_oog_under_stylus_earns_no_refund() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .with_network_fee_account(NETWORK_FEE_ACCOUNT)
        .initialize();

    let program_code = hb(PROGRAM_HEX);
    arb_storage::set_account_code(harness.state(), PROGRAM, Bytes::from(program_code.clone()));
    activate_and_seed_program(harness.state(), &program_code, 3);
    let caller_code = hb(CALLER_HEX);
    arb_storage::set_account_code(harness.state(), CALLER, Bytes::from(caller_code.clone()));
    activate_and_seed_program(harness.state(), &caller_code, 3);
    arb_executor_tests::helpers::deploy_contract(
        harness.state(),
        FORWARDER,
        forwarder_runtime(PROGRAM, GAS_CAP),
        U256::ZERO,
    );
    arb_executor_tests::helpers::fund_account(
        harness.state(),
        SENDER,
        U256::from(10u128).pow(U256::from(18u64)),
    );
    seed_l1_price_zero(harness.state());
    let sender_before = read_balance(harness.state(), SENDER);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.cfg_env.tx_gas_limit_cap = Some(u64::MAX);
    env.block_env.timestamp = U256::from(BLOCK_TIMESTAMP);
    env.block_env.basefee = BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);
    env.block_env.beneficiary = SEQUENCER;

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        L1_BLOCK_NUMBER,
        false,
    );
    evm_factory.stage_ctx(Arc::new(arb_context::ArbPrecompileCtx::with_block(
        Arc::new(block_ctx),
    )));

    let sink = MultiGasSink::default();
    let evm = evm_factory.create_evm_with_inspector(
        harness.state(),
        env,
        MultiGasInspector::with_sink(sink.clone()),
    );
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(2),
        parent_hash: PARENT_HASH,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.set_multi_gas_sink(sink);
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = L1_BLOCK_NUMBER;
    executor.apply_pre_execution_changes().expect("pre-exec");

    let sb = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: encode_start_block(U256::ZERO, L1_BLOCK_NUMBER, BLOCK_NUMBER, 0).into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let r = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(sb, ARBOS_ADDRESS))
        .expect("startblock");
    executor.commit_transaction(r).expect("commit sb");

    // forward(address target, bytes data) with the forwarder as target and
    // empty data: routes the call through the Stylus caller into the forwarder.
    let mut call_data = alloy_primitives::keccak256(b"forward(address,bytes)")[..4].to_vec();
    let mut w = [0u8; 32];
    w[12..].copy_from_slice(FORWARDER.as_slice());
    call_data.extend_from_slice(&w);
    call_data.extend_from_slice(&{
        let mut o = [0u8; 32];
        o[31] = 0x40;
        o
    });
    call_data.extend_from_slice(&[0u8; 32]);
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Legacy(TxLegacy {
            chain_id: Some(CHAIN_ID),
            nonce: 0,
            gas_price: BASE_FEE,
            gas_limit: 2_000_000,
            to: alloy_primitives::TxKind::Call(CALLER),
            value: U256::ZERO,
            input: Bytes::from(call_data),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let result = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(tx, SENDER))
        .expect("user tx");
    let gas_used = result.result.result.gas_used();
    executor.commit_transaction(result).expect("commit user tx");
    let _ = executor.finish().expect("finish");

    let sender_after = read_balance(harness.state(), SENDER);
    let paid = sender_before - sender_after;
    let expected = U256::from(gas_used) * U256::from(BASE_FEE);
    assert_eq!(
        paid, expected,
        "sender net charge must equal base_fee * gas_used (no refund); gas_used={gas_used} paid={paid} expected={expected}, over-refund={}",
        expected.saturating_sub(paid),
    );
}
