//! Per-opcode multi-gas dimension rules, mirroring Nitro's gas functions.
//!
//! Each opcode's total gas is split across resource kinds. The non-computation
//! amounts are derived from the EIP-2929/2200 constants below; whatever the
//! observed total leaves over is computation (base cost, warm access, memory
//! expansion, value transfer, forwarded call gas).

use alloy_primitives::U256;
use arb_primitives::multigas::MultiGas;

const WARM: u64 = 100; // WarmStorageReadCostEIP2929
const COLD_SLOAD: u64 = 2100; // ColdSloadCostEIP2929
const COLD_ACCOUNT: u64 = 2600; // ColdAccountAccessCostEIP2929
const SSTORE_SET: u64 = 20_000; // SstoreSetGasEIP2200
const SSTORE_RESET_WRITE: u64 = 2_900; // SstoreResetGasEIP2200 - ColdSloadCostEIP2929
const NEW_ACCOUNT: u64 = 25_000; // CallNewAccountGas / CreateBySelfdestructGas
const SELFDESTRUCT_BASE_WRITE: u64 = 4_900; // SelfdestructGasEIP150 - WarmStorageReadCostEIP2929
const COPY_WORD: u64 = 3; // CopyGas
const LOG_TOPIC_HISTORY: u64 = 256; // LogDataGas * LogTopicBytes (8 * 32)
const LOG_DATA: u64 = 8; // LogDataGas

/// Classification inputs for the gas-relevant opcodes; all other opcodes are
/// pure computation.
#[derive(Debug, Clone, Copy)]
pub enum OpKind {
    /// SLOAD.
    StorageRead { cold: bool },
    /// SSTORE.
    StorageWrite {
        cold: bool,
        original: U256,
        present: U256,
        new: U256,
    },
    /// BALANCE, EXTCODESIZE, EXTCODEHASH, DELEGATECALL, STATICCALL — cold
    /// surcharge only (their warm cost is the constant, i.e. computation).
    AccountAccess { cold: bool },
    /// EXTCODECOPY — cold surcharge plus per-word copy, both storage read.
    ExtCodeCopy { cold: bool, words: u64 },
    /// LOG0..LOG4.
    Log { topics: u8, data_len: u64 },
    /// CALL, CALLCODE — cold surcharge plus new-account growth.
    Call { cold: bool, new_account: bool },
    /// SELFDESTRUCT — base write split, cold beneficiary read, new-account growth.
    SelfDestruct { cold: bool, new_account: bool },
    /// Any opcode whose whole cost is computation.
    Other,
}

/// Split `gas` (the opcode's observed total) across resource kinds. The
/// remainder after the typed amounts falls into computation, so the result
/// always sums to `gas`.
pub fn classify(kind: OpKind, gas: u64) -> MultiGas {
    let non_comp = non_computation(kind);
    // An opcode that runs out of gas is charged only the gas that was left,
    // which can be less than its typed cost. Attribute just the consumed gas so
    // the split never exceeds what the opcode actually used.
    if non_comp.single_gas() > gas {
        return MultiGas::computation_gas(gas);
    }
    let computation = gas - non_comp.single_gas();
    non_comp.saturating_add(MultiGas::computation_gas(computation))
}

fn non_computation(kind: OpKind) -> MultiGas {
    use arb_primitives::multigas::ResourceKind::*;
    match kind {
        OpKind::StorageRead { cold } => {
            MultiGas::storage_access_read_gas(if cold { COLD_SLOAD - WARM } else { 0 })
        }
        OpKind::StorageWrite {
            cold,
            original,
            present,
            new,
        } => {
            let mut pairs = Vec::with_capacity(2);
            if cold {
                pairs.push((StorageAccessRead, COLD_SLOAD));
            }
            if present != new && original == present {
                if original.is_zero() {
                    pairs.push((StorageGrowth, SSTORE_SET));
                } else {
                    pairs.push((StorageAccessWrite, SSTORE_RESET_WRITE));
                }
            }
            MultiGas::from_pairs(&pairs)
        }
        OpKind::AccountAccess { cold } => {
            MultiGas::storage_access_read_gas(if cold { COLD_ACCOUNT - WARM } else { 0 })
        }
        OpKind::ExtCodeCopy { cold, words } => {
            let cold_part = if cold { COLD_ACCOUNT - WARM } else { 0 };
            MultiGas::storage_access_read_gas(cold_part + words * COPY_WORD)
        }
        OpKind::Log { topics, data_len } => {
            MultiGas::history_growth_gas(LOG_TOPIC_HISTORY * topics as u64 + LOG_DATA * data_len)
        }
        OpKind::Call { cold, new_account } => {
            let mut pairs = Vec::with_capacity(2);
            if cold {
                pairs.push((StorageAccessRead, COLD_ACCOUNT - WARM));
            }
            if new_account {
                pairs.push((StorageGrowth, NEW_ACCOUNT));
            }
            MultiGas::from_pairs(&pairs)
        }
        OpKind::SelfDestruct { cold, new_account } => {
            let mut pairs = vec![(StorageAccessWrite, SELFDESTRUCT_BASE_WRITE)];
            if cold {
                pairs.push((StorageAccessRead, COLD_ACCOUNT));
            }
            if new_account {
                pairs.push((StorageGrowth, NEW_ACCOUNT));
            }
            MultiGas::from_pairs(&pairs)
        }
        OpKind::Other => MultiGas::zero(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use arb_primitives::multigas::ResourceKind::*;

    fn dims(mg: &MultiGas) -> (u64, u64, u64, u64, u64) {
        (
            mg.get(Computation),
            mg.get(StorageAccessRead),
            mg.get(StorageAccessWrite),
            mg.get(StorageGrowth),
            mg.get(HistoryGrowth),
        )
    }

    #[test]
    fn sload_warm() {
        let mg = classify(OpKind::StorageRead { cold: false }, WARM);
        assert_eq!(dims(&mg), (100, 0, 0, 0, 0));
    }

    #[test]
    fn sload_cold() {
        let mg = classify(OpKind::StorageRead { cold: true }, COLD_SLOAD);
        assert_eq!(dims(&mg), (100, 2000, 0, 0, 0));
        assert_eq!(mg.single_gas(), COLD_SLOAD);
    }

    #[test]
    fn sstore_cold_create() {
        let mg = classify(
            OpKind::StorageWrite {
                cold: true,
                original: U256::ZERO,
                present: U256::ZERO,
                new: U256::from(1),
            },
            COLD_SLOAD + SSTORE_SET,
        );
        assert_eq!(dims(&mg), (0, 2100, 0, 20000, 0));
    }

    #[test]
    fn sstore_warm_reset() {
        let mg = classify(
            OpKind::StorageWrite {
                cold: false,
                original: U256::from(7),
                present: U256::from(7),
                new: U256::from(9),
            },
            SSTORE_RESET_WRITE,
        );
        assert_eq!(dims(&mg), (0, 0, 2900, 0, 0));
    }

    #[test]
    fn sstore_cold_reset() {
        let mg = classify(
            OpKind::StorageWrite {
                cold: true,
                original: U256::from(7),
                present: U256::from(7),
                new: U256::from(9),
            },
            COLD_SLOAD + SSTORE_RESET_WRITE,
        );
        assert_eq!(dims(&mg), (0, 2100, 2900, 0, 0));
    }

    #[test]
    fn sstore_warm_dirty_and_noop_are_computation() {
        let dirty = classify(
            OpKind::StorageWrite {
                cold: false,
                original: U256::from(1),
                present: U256::from(2),
                new: U256::from(3),
            },
            WARM,
        );
        assert_eq!(dims(&dirty), (100, 0, 0, 0, 0));
        let noop = classify(
            OpKind::StorageWrite {
                cold: false,
                original: U256::from(5),
                present: U256::from(5),
                new: U256::from(5),
            },
            WARM,
        );
        assert_eq!(dims(&noop), (100, 0, 0, 0, 0));
    }

    #[test]
    fn account_access_cold_warm() {
        assert_eq!(
            dims(&classify(
                OpKind::AccountAccess { cold: true },
                COLD_ACCOUNT
            )),
            (100, 2500, 0, 0, 0)
        );
        assert_eq!(
            dims(&classify(OpKind::AccountAccess { cold: false }, WARM)),
            (100, 0, 0, 0, 0)
        );
    }

    #[test]
    fn extcodecopy_cold_with_copy() {
        // cold account (2500 read) + 3 words * 3 read + memory/base in computation
        let total = WARM + (COLD_ACCOUNT - WARM) + 3 * COPY_WORD + 12;
        let mg = classify(
            OpKind::ExtCodeCopy {
                cold: true,
                words: 3,
            },
            total,
        );
        assert_eq!(dims(&mg), (112, 2509, 0, 0, 0));
    }

    #[test]
    fn log_two_topics() {
        // LOG2 base 375 + 2*119 computation; 2*256 + 8*data history
        let data_len = 10u64;
        let total = 375 + 2 * 119 + LOG_TOPIC_HISTORY * 2 + LOG_DATA * data_len;
        let mg = classify(
            OpKind::Log {
                topics: 2,
                data_len,
            },
            total,
        );
        assert_eq!(dims(&mg), (613, 0, 0, 0, 592));
    }

    #[test]
    fn call_cold_new_account_value() {
        // warm 100 + cold 2500 + value 9000 + new account 25000
        let total = WARM + (COLD_ACCOUNT - WARM) + 9000 + NEW_ACCOUNT;
        let mg = classify(
            OpKind::Call {
                cold: true,
                new_account: true,
            },
            total,
        );
        assert_eq!(dims(&mg), (9100, 2500, 0, 25000, 0));
    }

    #[test]
    fn selfdestruct_cold_new_account() {
        // base 5000 (100 comp + 4900 write) + cold 2600 read + new 25000 growth
        let total = 5000 + COLD_ACCOUNT + NEW_ACCOUNT;
        let mg = classify(
            OpKind::SelfDestruct {
                cold: true,
                new_account: true,
            },
            total,
        );
        assert_eq!(dims(&mg), (100, 2600, 4900, 25000, 0));
    }

    #[test]
    fn other_is_all_computation() {
        let mg = classify(OpKind::Other, 42);
        assert_eq!(dims(&mg), (42, 0, 0, 0, 0));
    }

    #[test]
    fn out_of_gas_opcode_attributes_only_consumed_gas() {
        // A cold SLOAD that runs out of gas consumes less than the 2100 cold
        // cost; the cold surcharge must not be attributed in full.
        let sload = classify(OpKind::StorageRead { cold: true }, 1_696);
        assert_eq!(sload.single_gas(), 1_696);
        assert_eq!(dims(&sload), (1_696, 0, 0, 0, 0));
        // A warm-reset SSTORE that runs out of gas (needs 2900, had 2707).
        let sstore = classify(
            OpKind::StorageWrite {
                cold: false,
                original: U256::from(7),
                present: U256::from(7),
                new: U256::from(9),
            },
            2_707,
        );
        assert_eq!(sstore.single_gas(), 2_707);
        // A new-account CALL that runs out before the 25000 growth is charged.
        let call = classify(
            OpKind::Call {
                cold: true,
                new_account: true,
            },
            5_000,
        );
        assert_eq!(call.single_gas(), 5_000);
    }

    #[test]
    fn single_gas_always_matches_total() {
        for (kind, gas) in [
            (OpKind::StorageRead { cold: true }, COLD_SLOAD),
            (
                OpKind::Call {
                    cold: true,
                    new_account: true,
                },
                36600,
            ),
            (
                OpKind::Log {
                    topics: 4,
                    data_len: 100,
                },
                9999,
            ),
            (OpKind::Other, 21000),
        ] {
            assert_eq!(classify(kind, gas).single_gas(), gas);
        }
    }
}
