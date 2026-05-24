//! Per-selector gas pins for ArbInfo (0x65).
//!
//! Locks the exact `PrecompileOutput::gas_used` returned for every selector
//! so any future refactor that drops a charge or changes the per-method
//! schedule fails a named test instead of going unnoticed.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbinfo_precompile;
use common::{calldata, word_address, PrecompileTest};
use revm::{bytecode::Bytecode, state::AccountInfo};

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;
const BALANCE_GAS_EIP1884: u64 = 700;
const COLD_SLOAD_COST_EIP2929: u64 = 2100;

fn arbinfo() -> DynPrecompile {
    create_arbinfo_precompile()
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(30).arbos_state()
}

#[test]
fn get_balance_v30_gas_pin() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().balance(addr, U256::from(1_234u64)).call(
        &arbinfo(),
        &calldata("getBalance(address)", &[word_address(addr)]),
    );
    // OpenArbosState(800) + argsCost(3) + BalanceGasEIP1884(700) + resultCost(3) = 1506
    assert_eq!(
        run.gas_used(),
        SLOAD_GAS + COPY_GAS + BALANCE_GAS_EIP1884 + COPY_GAS
    );
}

#[test]
fn get_code_empty_v30_gas_pin() {
    let addr: Address = address!("00000000000000000000000000000000000000bb");
    let run = fixture().call(
        &arbinfo(),
        &calldata("getCode(address)", &[word_address(addr)]),
    );
    // OpenArbosState(800) + argsCost(3) + ColdSloadCostEIP2929(2100)
    //   + 0 code words + 2 result-head words (offset+length) = 2909
    assert_eq!(
        run.gas_used(),
        SLOAD_GAS + COPY_GAS + COLD_SLOAD_COST_EIP2929 + 2 * COPY_GAS,
    );
}

#[test]
fn get_code_with_14_bytes_v30_gas_pin() {
    let addr: Address = address!("00000000000000000000000000000000000000cc");
    let code = vec![
        0x60, 0x00, 0x60, 0x00, 0xfd, 0x60, 0x80, 0x60, 0x40, 0x52, 0x60, 0x04, 0x36, 0x10,
    ];
    let run = fixture()
        .account(
            addr,
            AccountInfo {
                code: Some(Bytecode::new_raw(code.clone().into())),
                code_hash: alloy_primitives::keccak256(&code),
                ..Default::default()
            },
        )
        .call(
            &arbinfo(),
            &calldata("getCode(address)", &[word_address(addr)]),
        );
    // 14 bytes → 1 code word; output is 64 + 14 + 18 pad = 96 bytes = 3 result words.
    assert_eq!(
        run.gas_used(),
        SLOAD_GAS + COPY_GAS + COLD_SLOAD_COST_EIP2929 + COPY_GAS + 3 * COPY_GAS,
    );
}
