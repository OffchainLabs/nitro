use arb_precompiles::storage_slot::*;

#[test]
fn print_key_slots() {
    println!("=== L1 Pricing (correct offsets) ===");
    for (name, offset) in [
        ("pay_rewards_to", 0u64),
        ("equilibration_units", 1),
        ("inertia", 2),
        ("per_unit_reward", 3),
        ("last_update_time", 4),
        ("funds_due_for_rewards", 5),
        ("units_since", 6),
        ("price_per_unit", 7),
        ("last_surplus", 8),
        ("per_batch_gas_cost", 9),
        ("amortized_cost_cap_bips", 10),
        ("l1_fees_available", 11),
    ] {
        let slot = subspace_slot(L1_PRICING_SUBSPACE, offset);
        println!("{}: 0x{:064x}", name, slot);
    }
}
