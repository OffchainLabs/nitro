mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbinfo_precompile;
use common::{calldata, decode_u256, word_address, PrecompileTest};
use revm::state::AccountInfo;

fn arbinfo() -> DynPrecompile {
    create_arbinfo_precompile()
}

#[test]
fn get_balance_returns_account_balance() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let bal = U256::from(1_234_567_890_u64);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .account(
            addr,
            AccountInfo {
                balance: bal,
                ..Default::default()
            },
        )
        .call(
            &arbinfo(),
            &calldata("getBalance(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), bal);
}

#[test]
fn get_balance_returns_zero_for_unknown_account() {
    let addr: Address = address!("00000000000000000000000000000000000000bb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbinfo(),
            &calldata("getBalance(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn get_code_returns_empty_for_account_without_code() {
    let addr: Address = address!("00000000000000000000000000000000000000cc");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbinfo(),
            &calldata("getCode(address)", &[word_address(addr)]),
        );
    let out = run.output();
    let length = U256::from_be_slice(&out[32..64]);
    assert_eq!(length, U256::ZERO);
}
