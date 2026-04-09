//! Asserts that `register_arb_precompiles` produces the right precompile address
//! set for each ArbOS version, matching Nitro's gethhook precompile selection.

use alloy_evm::{eth::EthEvmContext, precompiles::PrecompilesMap};
use alloy_primitives::{address, Address};
use arb_precompiles::register_arb_precompiles;
use revm::{
    database::EmptyDB,
    handler::{EthPrecompiles, PrecompileProvider},
    primitives::hardfork::SpecId,
};

const ECRECOVER: Address = address!("0000000000000000000000000000000000000001");
const SHA256: Address = address!("0000000000000000000000000000000000000002");
const RIPEMD: Address = address!("0000000000000000000000000000000000000003");
const IDENTITY: Address = address!("0000000000000000000000000000000000000004");
const MODEXP: Address = address!("0000000000000000000000000000000000000005");
const BN_ADD: Address = address!("0000000000000000000000000000000000000006");
const BN_MUL: Address = address!("0000000000000000000000000000000000000007");
const BN_PAIR: Address = address!("0000000000000000000000000000000000000008");
const BLAKE2F: Address = address!("0000000000000000000000000000000000000009");
const KZG: Address = address!("000000000000000000000000000000000000000a");
const BLS_G1_ADD: Address = address!("000000000000000000000000000000000000000b");
const BLS_G1_MSM: Address = address!("000000000000000000000000000000000000000c");
const BLS_G2_ADD: Address = address!("000000000000000000000000000000000000000d");
const BLS_G2_MSM: Address = address!("000000000000000000000000000000000000000e");
const BLS_PAIRING: Address = address!("000000000000000000000000000000000000000f");
const BLS_MAP_FP: Address = address!("0000000000000000000000000000000000000010");
const BLS_MAP_FP2: Address = address!("0000000000000000000000000000000000000011");
const P256VERIFY: Address = address!("0000000000000000000000000000000000000100");

fn build(spec: SpecId, arbos_version: u64) -> PrecompilesMap {
    let mut map = PrecompilesMap::from(EthPrecompiles::new(spec));
    register_arb_precompiles(&mut map, arbos_version);
    map
}

fn contains(map: &PrecompilesMap, addr: &Address) -> bool {
    <PrecompilesMap as PrecompileProvider<EthEvmContext<EmptyDB>>>::contains(map, addr)
}

#[test]
fn arbos_29_excludes_bls_kzg_p256() {
    // Pre-Stylus: Berlin precompiles (0x01..0x09) only.
    let map = build(SpecId::SHANGHAI, 29);
    for addr in [ECRECOVER, SHA256, RIPEMD, IDENTITY, MODEXP, BN_ADD, BN_MUL, BN_PAIR, BLAKE2F] {
        assert!(contains(&map, &addr), "expected {addr} for ArbOS 29");
    }
    for addr in [
        KZG, BLS_G1_ADD, BLS_G1_MSM, BLS_G2_ADD, BLS_G2_MSM, BLS_PAIRING, BLS_MAP_FP, BLS_MAP_FP2,
        P256VERIFY,
    ] {
        assert!(!contains(&map, &addr), "did not expect {addr} for ArbOS 29");
    }
}

#[test]
fn arbos_30_includes_p256_and_kzg_excludes_bls() {
    // Stylus (Cancun + P256VERIFY).
    let map = build(SpecId::CANCUN, 30);
    for addr in [
        ECRECOVER, SHA256, RIPEMD, IDENTITY, MODEXP, BN_ADD, BN_MUL, BN_PAIR, BLAKE2F, KZG,
        P256VERIFY,
    ] {
        assert!(contains(&map, &addr), "expected {addr} for ArbOS 30");
    }
    for addr in [
        BLS_G1_ADD, BLS_G1_MSM, BLS_G2_ADD, BLS_G2_MSM, BLS_PAIRING, BLS_MAP_FP, BLS_MAP_FP2,
    ] {
        assert!(!contains(&map, &addr), "did not expect {addr} for ArbOS 30");
    }
}

#[test]
fn arbos_50_includes_bls_and_p256() {
    // Dia: Cancun + P256VERIFY + Osaka (BLS12-381 + Osaka modexp/P256).
    // The OSAKA P256 (6900 gas) is overridden with the 3450-gas version by
    // register_arb_precompiles, so the address still resolves but our handler is in
    // place. Address-set membership is what we assert here.
    let map = build(SpecId::OSAKA, 50);
    for addr in [
        ECRECOVER,
        SHA256,
        RIPEMD,
        IDENTITY,
        MODEXP,
        BN_ADD,
        BN_MUL,
        BN_PAIR,
        BLAKE2F,
        KZG,
        BLS_G1_ADD,
        BLS_G1_MSM,
        BLS_G2_ADD,
        BLS_G2_MSM,
        BLS_PAIRING,
        BLS_MAP_FP,
        BLS_MAP_FP2,
        P256VERIFY,
    ] {
        assert!(contains(&map, &addr), "expected {addr} for ArbOS 50");
    }
}
