//! Transaction-level intrinsic gas split, mirroring Nitro's `IntrinsicMultiGas`.
//!
//! The intrinsic cost is charged before any opcode runs, so the inspector never
//! sees it; it is classified here from the transaction fields and added to the
//! per-tx accumulator.

use arb_primitives::multigas::{MultiGas, ResourceKind};

const TX_GAS: u64 = 21_000;
const TX_GAS_CREATE: u64 = 53_000;
const TX_DATA_NONZERO: u64 = 16; // TxDataNonZeroGasEIP2028
const TX_DATA_ZERO: u64 = 4; // TxDataZeroGas
const INIT_CODE_WORD: u64 = 2; // InitCodeWordGas
const ACCESS_LIST_ADDRESS: u64 = 2_400; // TxAccessListAddressGas
const ACCESS_LIST_KEY: u64 = 1_900; // TxAccessListStorageKeyGas
const AUTH_ACCOUNT: u64 = 25_000; // CallNewAccountGas (EIP-7702)

/// Transaction fields needed to split the intrinsic gas.
#[derive(Debug, Clone, Copy)]
pub struct IntrinsicInput {
    pub is_create: bool,
    pub zero_bytes: u64,
    pub nonzero_bytes: u64,
    pub init_code_words: u64,
    pub access_list_addresses: u64,
    pub access_list_keys: u64,
    pub auth_list_len: u64,
}

/// Split the intrinsic gas across resource kinds: base and init-code words are
/// computation, calldata bytes are L2 calldata, access-list entries are storage
/// reads, and EIP-7702 authorizations are storage growth.
pub fn intrinsic_multigas(input: IntrinsicInput) -> MultiGas {
    let base = if input.is_create {
        TX_GAS_CREATE
    } else {
        TX_GAS
    };
    let init_code = if input.is_create {
        input.init_code_words * INIT_CODE_WORD
    } else {
        0
    };
    let calldata = input.nonzero_bytes * TX_DATA_NONZERO + input.zero_bytes * TX_DATA_ZERO;
    let access = input.access_list_addresses * ACCESS_LIST_ADDRESS
        + input.access_list_keys * ACCESS_LIST_KEY;
    let auth = input.auth_list_len * AUTH_ACCOUNT;

    MultiGas::from_pairs(&[
        (ResourceKind::Computation, base + init_code),
        (ResourceKind::L2Calldata, calldata),
        (ResourceKind::StorageAccessRead, access),
        (ResourceKind::StorageGrowth, auth),
    ])
}

#[cfg(test)]
mod tests {
    use super::*;
    use arb_primitives::multigas::ResourceKind::*;

    fn input() -> IntrinsicInput {
        IntrinsicInput {
            is_create: false,
            zero_bytes: 0,
            nonzero_bytes: 0,
            init_code_words: 0,
            access_list_addresses: 0,
            access_list_keys: 0,
            auth_list_len: 0,
        }
    }

    #[test]
    fn bare_call() {
        let mg = intrinsic_multigas(input());
        assert_eq!(mg.get(Computation), 21_000);
        assert_eq!(mg.single_gas(), 21_000);
    }

    #[test]
    fn calldata_goes_to_l2_calldata() {
        let mg = intrinsic_multigas(IntrinsicInput {
            zero_bytes: 5,
            nonzero_bytes: 10,
            ..input()
        });
        assert_eq!(mg.get(Computation), 21_000);
        assert_eq!(mg.get(L2Calldata), 10 * 16 + 5 * 4);
        assert_eq!(mg.single_gas(), 21_000 + 180);
    }

    #[test]
    fn creation_base_and_init_words() {
        let mg = intrinsic_multigas(IntrinsicInput {
            is_create: true,
            init_code_words: 7,
            ..input()
        });
        assert_eq!(mg.get(Computation), 53_000 + 7 * 2);
    }

    #[test]
    fn access_list_is_storage_read() {
        let mg = intrinsic_multigas(IntrinsicInput {
            access_list_addresses: 2,
            access_list_keys: 3,
            ..input()
        });
        assert_eq!(mg.get(StorageAccessRead), 2 * 2_400 + 3 * 1_900);
    }

    #[test]
    fn auth_list_is_storage_growth() {
        let mg = intrinsic_multigas(IntrinsicInput {
            auth_list_len: 2,
            ..input()
        });
        assert_eq!(mg.get(StorageGrowth), 2 * 25_000);
    }
}
