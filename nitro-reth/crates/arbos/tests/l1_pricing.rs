use alloy_primitives::{address, Address, U256};
use arb_test_utils::ArbosHarness;
use arbos::l1_pricing::BATCH_POSTER_ADDRESS;

const ONE_GWEI: u64 = 1_000_000_000;

#[test]
fn l1_price_update_initial_state() {
    let initial_price = U256::from(123u64) * U256::from(ONE_GWEI);
    let mut h = ArbosHarness::new()
        .with_l1_initial_base_fee(initial_price)
        .initialize();
    let ps = h.l1_pricing_state();

    assert_eq!(ps.last_update_time().unwrap(), 0);
    assert_eq!(ps.price_per_unit().unwrap(), initial_price);
}

#[test]
fn batch_poster_table_lifecycle() {
    let mut h = ArbosHarness::new().initialize();
    let l1 = h.l1_pricing_state();
    let bpt = l1.batch_poster_table();

    let addr1: Address = address!("0102030000000000000000000000000000000000");
    let pay1: Address = address!("0405060700000000000000000000000000000000");
    let addr2: Address = address!("0204060000000000000000000000000000000000");
    let pay2: Address = address!("080A0C0E00000000000000000000000000000000");

    let initial = bpt.all_posters().unwrap();
    assert_eq!(initial.len(), 1);
    assert_eq!(initial[0], BATCH_POSTER_ADDRESS);
    assert!(!bpt.contains_poster(addr1).unwrap());

    let bp1 = bpt.add_poster(addr1, pay1).unwrap();
    assert_eq!(bp1.pay_to().unwrap(), pay1);
    assert_eq!(bp1.funds_due().unwrap(), U256::ZERO);
    assert!(bpt.contains_poster(addr1).unwrap());

    let bp2 = bpt.add_poster(addr2, pay2).unwrap();
    assert_eq!(bp2.pay_to().unwrap(), pay2);
    assert_eq!(bp2.funds_due().unwrap(), U256::ZERO);
    assert!(bpt.contains_poster(addr2).unwrap());

    assert_eq!(bpt.all_posters().unwrap().len(), 3);

    let bp1 = bpt.open_poster(addr1, false).unwrap();
    bp1.set_pay_to(addr2).unwrap();
    assert_eq!(bp1.pay_to().unwrap(), addr2);

    bp1.set_funds_due(U256::from(13u64), &bpt.total_funds_due)
        .unwrap();
    assert_eq!(bp1.funds_due().unwrap(), U256::from(13u64));

    bp2.set_funds_due(U256::from(42u64), &bpt.total_funds_due)
        .unwrap();
    assert_eq!(bp2.funds_due().unwrap(), U256::from(42u64));

    assert_eq!(bpt.total_funds_due().unwrap(), U256::from(55u64));
}

#[test]
fn add_poster_twice_fails() {
    let mut h = ArbosHarness::new().initialize();
    let l1 = h.l1_pricing_state();
    let bpt = l1.batch_poster_table();
    let addr: Address = address!("0102030000000000000000000000000000000000");
    bpt.add_poster(addr, addr).unwrap();
    assert!(bpt.add_poster(addr, addr).is_err());
}

#[test]
fn fresh_l1_pricing_state_reports_zero_units() {
    let mut h = ArbosHarness::new().initialize();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.units_since_update().unwrap(), 0);
    assert_eq!(ps.l1_fees_available().unwrap(), U256::ZERO);
}
