mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbfunctiontable_precompile;
use common::{calldata, decode_u256, word_address, word_u256, PrecompileTest};

fn arbfunctiontable() -> DynPrecompile {
    create_arbfunctiontable_precompile()
}

#[test]
fn upload_is_noop() {
    let payload = vec![0u8; 64];
    let mut data = vec![0xce, 0x2a, 0xe1, 0x59];
    data.extend_from_slice(&payload);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&arbfunctiontable(), &data.into());
    let out = run.assert_ok();
    assert!(out.bytes.is_empty());
}

#[test]
fn size_returns_zero() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbfunctiontable(),
        &calldata("size(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn get_reverts_table_empty() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbfunctiontable(),
        &calldata(
            "get(address,uint256)",
            &[word_address(probe), word_u256(U256::ZERO)],
        ),
    );
    assert!(run.assert_ok().reverted);
}
