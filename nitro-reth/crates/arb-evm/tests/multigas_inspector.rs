//! End-to-end checks of `MultiGasInspector` against real revm execution.
//!
//! The central invariant is that the inspector's accumulated per-dimension gas,
//! summed, equals the transaction's execution gas (`gas_used` minus the 21000
//! intrinsic, which the inspector does not observe). This holds across cold and
//! warm storage, slot creation, and nested calls, exercising the forwarded-gas
//! reconciliation.

use arb_evm::multi_gas::MultiGasInspector;
use arb_primitives::multigas::{MultiGas, ResourceKind};
use revm::{
    bytecode::{opcode, Bytecode},
    context::TxEnv,
    database::{CacheDB, EmptyDB},
    primitives::{hardfork::SpecId, Address, TxKind, U256},
    state::AccountInfo,
    Context, InspectEvm, MainBuilder, MainContext,
};

const INTRINSIC: u64 = 21_000;
const TARGET: Address = Address::with_last_byte(0xaa);
const CALLEE: Address = Address::with_last_byte(0xbb);
const CALLER: Address = Address::with_last_byte(0x01);

fn account_with_code(code: Vec<u8>) -> AccountInfo {
    AccountInfo {
        code: Some(Bytecode::new_raw(code.into())),
        ..Default::default()
    }
}

fn run(target_code: Vec<u8>, callee_code: Option<Vec<u8>>) -> (u64, MultiGas) {
    let mut db = CacheDB::new(EmptyDB::default());
    db.insert_account_info(TARGET, account_with_code(target_code));
    if let Some(code) = callee_code {
        db.insert_account_info(CALLEE, account_with_code(code));
    }
    db.insert_account_info(
        CALLER,
        AccountInfo {
            balance: U256::from(10u64).pow(U256::from(18u64)),
            ..Default::default()
        },
    );

    let mut inspector = MultiGasInspector::default();
    let mut evm = Context::mainnet()
        .with_db(db)
        .modify_cfg_chained(|cfg| cfg.spec = SpecId::CANCUN)
        .build_mainnet_with_inspector(&mut inspector);

    let result = evm
        .inspect_one_tx(
            TxEnv::builder()
                .caller(CALLER)
                .kind(TxKind::Call(TARGET))
                .gas_limit(1_000_000)
                .build()
                .unwrap(),
        )
        .unwrap();

    assert!(result.is_success(), "tx reverted: {result:?}");
    let gas_used = result.gas_used();
    drop(evm);
    (gas_used, inspector.take_multi_gas())
}

fn push1(code: &mut Vec<u8>, byte: u8) {
    code.push(opcode::PUSH1);
    code.push(byte);
}

#[test]
fn cold_then_warm_sload() {
    let mut code = Vec::new();
    push1(&mut code, 0x00);
    code.push(opcode::SLOAD);
    code.push(opcode::POP);
    push1(&mut code, 0x00);
    code.push(opcode::SLOAD);
    code.push(opcode::POP);
    code.push(opcode::STOP);

    let (gas_used, mg) = run(code, None);

    // Cold SLOAD: 2000 read + 100 computation; warm SLOAD: 100 computation.
    assert_eq!(mg.get(ResourceKind::StorageAccessRead), 2_000);
    assert_eq!(mg.single_gas() + INTRINSIC, gas_used);
}

#[test]
fn sstore_create_slot() {
    let mut code = Vec::new();
    push1(&mut code, 0x01); // value
    push1(&mut code, 0x00); // key
    code.push(opcode::SSTORE);
    code.push(opcode::STOP);

    let (gas_used, mg) = run(code, None);

    // Cold create: 2100 read + 20000 growth, no refund.
    assert_eq!(mg.get(ResourceKind::StorageAccessRead), 2_100);
    assert_eq!(mg.get(ResourceKind::StorageGrowth), 20_000);
    assert_eq!(mg.single_gas() + INTRINSIC, gas_used);
}

#[test]
fn call_into_empty_callee_reconciles_forwarded_gas() {
    let mut code = Vec::new();
    // CALL(gas, addr, value, argsOff, argsLen, retOff, retLen); push in reverse.
    push1(&mut code, 0x00); // retLen
    push1(&mut code, 0x00); // retOff
    push1(&mut code, 0x00); // argsLen
    push1(&mut code, 0x00); // argsOff
    push1(&mut code, 0x00); // value
    code.push(opcode::PUSH20);
    code.extend_from_slice(CALLEE.as_slice());
    code.push(opcode::PUSH2);
    code.extend_from_slice(&[0xff, 0xff]); // gas
    code.push(opcode::CALL);
    code.push(opcode::POP);
    code.push(opcode::STOP);

    let (gas_used, mg) = run(code, Some(vec![opcode::STOP]));

    // Cold call, no value: 2500 read surcharge; forwarded gas is returned by the
    // callee and must not inflate the accumulated total.
    assert_eq!(mg.get(ResourceKind::StorageAccessRead), 2_500);
    assert_eq!(mg.get(ResourceKind::StorageGrowth), 0);
    assert_eq!(mg.single_gas() + INTRINSIC, gas_used);
}

#[test]
fn create_deposits_code_as_storage_growth() {
    // Init code that returns a 1-byte runtime (the zero byte already in memory):
    // PUSH1 0; PUSH1 0; MSTORE; PUSH1 1; PUSH1 0; RETURN.
    let init: [u8; 10] = [
        opcode::PUSH1,
        0x00,
        opcode::PUSH1,
        0x00,
        opcode::MSTORE,
        opcode::PUSH1,
        0x01,
        opcode::PUSH1,
        0x00,
        opcode::RETURN,
    ];

    // Place init in memory (right-aligned in the first word: bytes 22..32), then
    // CREATE(value=0, offset=22, length=10).
    let mut code = Vec::new();
    code.push(opcode::PUSH10);
    code.extend_from_slice(&init);
    push1(&mut code, 0x00);
    code.push(opcode::MSTORE);
    push1(&mut code, init.len() as u8); // length
    push1(&mut code, (32 - init.len()) as u8); // offset
    push1(&mut code, 0x00); // value
    code.push(opcode::CREATE);
    code.push(opcode::POP);
    code.push(opcode::STOP);

    let (gas_used, mg) = run(code, None);

    // Deployed runtime is 1 byte: code deposit = 1 * 200, as storage growth.
    assert_eq!(mg.get(ResourceKind::StorageGrowth), 200);
    assert_eq!(mg.single_gas() + INTRINSIC, gas_used);
}
