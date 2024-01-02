// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use c_kzg::{KzgSettings, BYTES_PER_G1_POINT, BYTES_PER_G2_POINT};
use num::BigUint;
use serde::{de::Error as _, Deserialize};

struct HexBytes;

impl<'de, const N: usize> serde_with::DeserializeAs<'de, [u8; N]> for HexBytes {
    fn deserialize_as<D>(deserializer: D) -> Result<[u8; N], D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        let mut s = s.as_str();
        if s.starts_with("0x") {
            s = &s[2..];
        }
        let mut bytes = [0; N];
        match hex::decode_to_slice(s, &mut bytes) {
            Ok(()) => Ok(bytes),
            Err(err) => Err(D::Error::custom(err.to_string())),
        }
    }
}

#[derive(Deserialize)]
struct TrustedSetup {
    #[serde(with = "serde_with::As::<Vec<HexBytes>>")]
    g1_lagrange: Vec<[u8; BYTES_PER_G1_POINT]>,
    #[serde(with = "serde_with::As::<Vec<HexBytes>>")]
    g2_monomial: Vec<[u8; BYTES_PER_G2_POINT]>,
}

const FIELD_ELEMENTS_PER_BLOB: usize = 4096;

lazy_static::lazy_static! {
    pub static ref ETHEREUM_KZG_SETTINGS: KzgSettings = {
        let trusted_setup = serde_json::from_str::<TrustedSetup>(include_str!("kzg-trusted-setup.json"))
            .expect("Failed to deserialize Ethereum trusted setup");
        KzgSettings::load_trusted_setup(&trusted_setup.g1_lagrange, &trusted_setup.g2_monomial)
            .expect("Failed to load Ethereum trusted setup")
    };

    pub static ref BLS_MODULUS: BigUint = "52435875175126190479447740508185965837690552500527637822603658699938581184513".parse().unwrap();
    pub static ref ROOT_OF_UNITY: BigUint = {
        // order 2^32
        let root: BigUint = "10238227357739495823651030575849232062558860180284477541189508159991286009131".parse().unwrap();
        let exponent = (1_u64 << 32) / (FIELD_ELEMENTS_PER_BLOB as u64);
        root.modpow(&BigUint::from(exponent), &BLS_MODULUS)
    };
}

#[cfg(test)]
#[test]
fn load_trusted_setup() {
    let _: &KzgSettings = &*ETHEREUM_KZG_SETTINGS;
}
