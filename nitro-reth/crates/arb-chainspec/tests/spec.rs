use arb_chainspec::{
    arbitrum_sepolia_spec_id_by_timestamp, spec_id_by_arbos_version, ArbChainSpec,
    ArbitrumChainSpec, ARBITRUM_ONE_CHAIN_ID, ARBITRUM_SEPOLIA_CANCUN_TIMESTAMP,
    ARBITRUM_SEPOLIA_PRAGUE_TIMESTAMP, ARBITRUM_SEPOLIA_SHANGHAI_TIMESTAMP,
};
use revm::primitives::hardfork::SpecId;

#[test]
fn arbitrum_one_chain_id_constant() {
    assert_eq!(ARBITRUM_ONE_CHAIN_ID, 42161);
}

#[test]
fn arbos_version_to_spec_id_thresholds() {
    assert_eq!(spec_id_by_arbos_version(0), SpecId::MERGE);
    assert_eq!(spec_id_by_arbos_version(10), SpecId::MERGE);
    assert_eq!(spec_id_by_arbos_version(11), SpecId::SHANGHAI);
    assert_eq!(spec_id_by_arbos_version(19), SpecId::SHANGHAI);
    assert_eq!(spec_id_by_arbos_version(20), SpecId::CANCUN);
    assert_eq!(spec_id_by_arbos_version(40), SpecId::CANCUN);
    assert_eq!(spec_id_by_arbos_version(50), SpecId::OSAKA);
    assert_eq!(spec_id_by_arbos_version(60), SpecId::OSAKA);
}

#[test]
fn sepolia_timestamp_thresholds_select_correct_spec() {
    assert_eq!(arbitrum_sepolia_spec_id_by_timestamp(0), SpecId::MERGE);
    assert_eq!(
        arbitrum_sepolia_spec_id_by_timestamp(ARBITRUM_SEPOLIA_SHANGHAI_TIMESTAMP - 1),
        SpecId::MERGE
    );
    assert_eq!(
        arbitrum_sepolia_spec_id_by_timestamp(ARBITRUM_SEPOLIA_SHANGHAI_TIMESTAMP),
        SpecId::SHANGHAI
    );
    assert_eq!(
        arbitrum_sepolia_spec_id_by_timestamp(ARBITRUM_SEPOLIA_CANCUN_TIMESTAMP),
        SpecId::CANCUN
    );
    assert_eq!(
        arbitrum_sepolia_spec_id_by_timestamp(ARBITRUM_SEPOLIA_PRAGUE_TIMESTAMP),
        SpecId::PRAGUE
    );
    assert_eq!(
        arbitrum_sepolia_spec_id_by_timestamp(u64::MAX),
        SpecId::PRAGUE
    );
}

#[test]
fn arb_chain_spec_chain_id_round_trip() {
    let s = ArbChainSpec { chain_id: 421614 };
    assert_eq!(s.chain_id(), 421614);
}

#[test]
fn arb_chain_spec_delegates_spec_id_lookup() {
    let s = ArbChainSpec { chain_id: 0 };
    assert_eq!(
        s.spec_id_by_timestamp(ARBITRUM_SEPOLIA_CANCUN_TIMESTAMP),
        SpecId::CANCUN
    );
    assert_eq!(s.spec_id_by_arbos_version(30), SpecId::CANCUN);
}
