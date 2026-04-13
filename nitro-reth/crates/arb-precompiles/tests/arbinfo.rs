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

#[test]
fn get_code_returns_deployed_bytecode_padded_to_32() {
    use revm::bytecode::Bytecode;
    let addr: Address = address!("00000000000000000000000000000000000000dd");
    // Pick an opcode sequence whose length isn't a multiple of 32 to also
    // exercise the right-padding ABI dynamic-bytes encoding.
    let code = vec![
        0x60, 0x00, 0x60, 0x00, 0xfd, 0x60, 0x80, 0x60, 0x40, 0x52, 0x60, 0x04, 0x36, 0x10,
    ];
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
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
    let out = run.output();
    // ABI: offset(32) | length(32) | data | pad-to-32
    let offset = U256::from_be_slice(&out[0..32]);
    let length = U256::from_be_slice(&out[32..64]);
    assert_eq!(offset, U256::from(32));
    assert_eq!(length, U256::from(code.len() as u64));
    assert_eq!(&out[64..64 + code.len()], &code[..]);
    let padded_len = 64 + code.len() + ((32 - code.len() % 32) % 32);
    assert_eq!(out.len(), padded_len);
}

#[test]
fn get_balance_works_for_zero_address() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .balance(Address::ZERO, U256::from(42))
        .call(
            &arbinfo(),
            &calldata("getBalance(address)", &[word_address(Address::ZERO)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(42));
}
