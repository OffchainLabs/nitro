//! Stylus contract fixtures: deploys real WASM at genesis, activated in
//! block 1 via `ArbWasm.activateProgram`.

use alloy_primitives::{Address, Bytes, B256, U256};
use tiny_keccak::{Hasher, Keccak};

use crate::runner::DeployedContract;

pub const ARB_WASM_ADDRESS: Address =
    alloy_primitives::address!("0000000000000000000000000000000000000071");

const STYLUS_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x00];
const BROTLI_DICT_TAG: u8 = 0;

use super::stylus_modules::StylusModule;

fn default_module() -> StylusModule {
    StylusModule::Noop
}

pub const STYLUS_FIXTURE_ADDRESS: Address =
    alloy_primitives::address!("000000000000000000000000000000005719cc01");

pub fn stylus_runtime_code_for(module: StylusModule) -> eyre::Result<Vec<u8>> {
    let wasm = wat::parse_bytes(module.wat().as_bytes())?.into_owned();
    let mut compressed = Vec::new();
    let mut encoder = brotli::CompressorWriter::new(&mut compressed, 4096, 11, 22);
    use std::io::Write as _;
    encoder.write_all(&wasm)?;
    drop(encoder);
    let mut code = Vec::with_capacity(STYLUS_DISCRIMINANT.len() + 1 + compressed.len());
    code.extend_from_slice(&STYLUS_DISCRIMINANT);
    code.push(BROTLI_DICT_TAG);
    code.extend_from_slice(&compressed);
    Ok(code)
}

pub fn stylus_runtime_code() -> eyre::Result<Vec<u8>> {
    stylus_runtime_code_for(default_module())
}

pub fn stylus_fixture_for(module: StylusModule) -> eyre::Result<DeployedContract> {
    Ok(DeployedContract {
        address: module.deploy_address(),
        runtime_code: stylus_runtime_code_for(module)?,
        balance: U256::ZERO,
    })
}

pub fn stylus_fixture() -> eyre::Result<DeployedContract> {
    stylus_fixture_for(default_module())
}

pub fn stylus_call_selector() -> [u8; 4] {
    let mut out = [0u8; 32];
    let mut h = Keccak::v256();
    h.update(b"run()");
    h.finalize(&mut out);
    let mut sel = [0u8; 4];
    sel.copy_from_slice(&out[..4]);
    sel
}

pub fn stylus_runtime_code_hash() -> eyre::Result<B256> {
    let code = stylus_runtime_code()?;
    let mut out = [0u8; 32];
    let mut h = Keccak::v256();
    h.update(&code);
    h.finalize(&mut out);
    Ok(B256::from(out))
}

pub fn activate_program_calldata(program: Address) -> Bytes {
    let selector = activate_program_selector();
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&selector);
    data.extend_from_slice(&[0u8; 12]);
    data.extend_from_slice(program.as_slice());
    Bytes::from(data)
}

fn activate_program_selector() -> [u8; 4] {
    let mut out = [0u8; 32];
    let mut h = Keccak::v256();
    h.update(b"activateProgram(address)");
    h.finalize(&mut out);
    let mut sel = [0u8; 4];
    sel.copy_from_slice(&out[..4]);
    sel
}

pub fn activate_program_value() -> U256 {
    U256::from(10_000_000_000_000_000u128)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn runtime_code_has_stylus_prefix() {
        let code = stylus_runtime_code().unwrap();
        assert_eq!(&code[..3], &STYLUS_DISCRIMINANT);
        assert!(code.len() > 10);
    }

    #[test]
    fn fixture_addresses_are_deterministic() {
        let a = stylus_runtime_code_hash().unwrap();
        let b = stylus_runtime_code_hash().unwrap();
        assert_eq!(a, b);
    }

    #[test]
    fn call_selector_is_known() {
        let sel = stylus_call_selector();
        // keccak256("run()") = 0xc0406226...
        assert_eq!(sel, [0xc0, 0x40, 0x62, 0x26]);
    }

    #[test]
    fn activate_program_calldata_well_formed() {
        let data = activate_program_calldata(STYLUS_FIXTURE_ADDRESS);
        assert_eq!(data.len(), 36);
        // Selector + 12 zero bytes + 20-byte address.
        assert_eq!(&data[4..16], &[0u8; 12]);
        assert_eq!(&data[16..36], STYLUS_FIXTURE_ADDRESS.as_slice());
    }

    #[test]
    fn activate_program_selector_matches_keccak256_signature() {
        let sel = activate_program_selector();
        // keccak256("activateProgram(address)") starts with 0x58c780c2.
        assert_eq!(sel, [0x58, 0xc7, 0x80, 0xc2]);
    }
}
