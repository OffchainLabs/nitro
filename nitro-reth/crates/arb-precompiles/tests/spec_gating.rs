//! Verifies that the precompile set we register for each ArbOS version matches
//! Nitro's `gethhook/geth-hook.go`:
//!
//! - ArbOS  < 30:  Berlin precompiles  (0x01..=0x09)              — no KZG, no BLS, no P256
//! - ArbOS >= 30:  Cancun precompiles + RIP-7212 P256VERIFY       — adds 0x0a, 0x100
//! - ArbOS >= 50:  Cancun + P256 + Osaka (BLS + Osaka modexp)     — adds 0x0b..=0x11

use alloy_evm::precompiles::PrecompilesMap;
use alloy_primitives::{address, Address};
use arb_precompiles::register_arb_precompiles;
use revm::{handler::EthPrecompiles, precompile::Precompiles, primitives::hardfork::SpecId};

fn enabled_eth_addresses(map: &PrecompilesMap) -> Vec<Address> {
    let mut out: Vec<Address> = map
        .addresses()
        .copied()
        .filter(|a| {
            let last4 = u32::from_be_bytes([a.0[16], a.0[17], a.0[18], a.0[19]]);
            let is_arb_range = (0x40..=0xff).contains(&last4);
            !is_arb_range && a.0[0..16].iter().all(|b| *b == 0)
        })
        .collect();
    out.sort();
    out
}

fn build_map(spec: SpecId, arbos_version: u64) -> PrecompilesMap {
    let mut precompiles = PrecompilesMap::from(EthPrecompiles {
        precompiles: Precompiles::new(spec.into()),
        spec,
    });
    register_arb_precompiles(&mut precompiles, arbos_version);
    precompiles
}

const KZG: Address = address!("000000000000000000000000000000000000000a");
const BLS_G1_ADD: Address = address!("000000000000000000000000000000000000000b");
const BLS_G1_MSM: Address = address!("000000000000000000000000000000000000000c");
const BLS_G2_ADD: Address = address!("000000000000000000000000000000000000000d");
const BLS_G2_MSM: Address = address!("000000000000000000000000000000000000000e");
const BLS_PAIRING: Address = address!("000000000000000000000000000000000000000f");
const BLS_MAP_FP: Address = address!("0000000000000000000000000000000000000010");
const BLS_MAP_FP2: Address = address!("0000000000000000000000000000000000000011");
const P256VERIFY: Address = address!("0000000000000000000000000000000000000100");

#[test]
fn pre_arbos_30_excludes_kzg_and_p256_and_bls() {
    // Use SpecId::CANCUN so the underlying precompile set has KZG; we expect the
    // Arbitrum registration to remove it for ArbOS < 30.
    let map = build_map(SpecId::CANCUN, 11);
    let addrs = enabled_eth_addresses(&map);
    assert!(!addrs.contains(&KZG), "KZG must not exist before ArbOS 30");
    assert!(
        !addrs.contains(&P256VERIFY),
        "P256VERIFY must not exist before ArbOS 30"
    );
    for bls in [
        BLS_G1_ADD,
        BLS_G1_MSM,
        BLS_G2_ADD,
        BLS_G2_MSM,
        BLS_PAIRING,
        BLS_MAP_FP,
        BLS_MAP_FP2,
    ] {
        assert!(
            !addrs.contains(&bls),
            "BLS {bls} must not exist before ArbOS 50"
        );
    }
}

#[test]
fn arbos_30_to_49_has_kzg_and_p256_no_bls() {
    let map = build_map(SpecId::PRAGUE, 30);
    let addrs = enabled_eth_addresses(&map);
    assert!(addrs.contains(&KZG), "KZG required from ArbOS 30");
    assert!(
        addrs.contains(&P256VERIFY),
        "P256VERIFY required from ArbOS 30"
    );
    for bls in [
        BLS_G1_ADD,
        BLS_G1_MSM,
        BLS_G2_ADD,
        BLS_G2_MSM,
        BLS_PAIRING,
        BLS_MAP_FP,
        BLS_MAP_FP2,
    ] {
        assert!(
            !addrs.contains(&bls),
            "BLS {bls} must not exist before ArbOS 50"
        );
    }
}

#[test]
fn arbos_50_plus_has_kzg_p256_and_bls() {
    let map = build_map(SpecId::PRAGUE, 51);
    let addrs = enabled_eth_addresses(&map);
    assert!(addrs.contains(&KZG));
    assert!(addrs.contains(&P256VERIFY));
    for bls in [
        BLS_G1_ADD,
        BLS_G1_MSM,
        BLS_G2_ADD,
        BLS_G2_MSM,
        BLS_PAIRING,
        BLS_MAP_FP,
        BLS_MAP_FP2,
    ] {
        assert!(addrs.contains(&bls), "BLS {bls} required from ArbOS 50");
    }
}
