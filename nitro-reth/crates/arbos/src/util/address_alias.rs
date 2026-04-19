use alloy_primitives::Address;

/// The offset applied to L1 addresses when they appear on L2.
/// This prevents a contract on L1 from impersonating an L2 address.
pub const ADDRESS_ALIAS_OFFSET: Address = {
    let bytes: [u8; 20] = [
        0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x11, 0x11,
    ];
    Address::new(bytes)
};

/// The inverse offset for remapping L2 aliased addresses back to L1.
///
/// Equals 2^160 - ADDRESS_ALIAS_OFFSET so that
/// `(x + ADDRESS_ALIAS_OFFSET + INVERSE_ADDRESS_ALIAS_OFFSET) mod 2^160 == x`.
pub const INVERSE_ADDRESS_ALIAS_OFFSET: Address = {
    let bytes: [u8; 20] = [
        0xee, 0xee, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
        0xff, 0xff, 0xff, 0xee, 0xef,
    ];
    Address::new(bytes)
};

/// Applies the L1→L2 address alias offset.
pub fn remap_l1_address(l1_address: Address) -> Address {
    address_add(l1_address, ADDRESS_ALIAS_OFFSET)
}

/// Removes the L1→L2 address alias offset.
pub fn inverse_remap_l1_address(l2_address: Address) -> Address {
    address_add(l2_address, INVERSE_ADDRESS_ALIAS_OFFSET)
}

/// Wrapping addition of two addresses (treated as 160-bit integers).
fn address_add(a: Address, b: Address) -> Address {
    let a_bytes = a.0 .0;
    let b_bytes = b.0 .0;
    let mut result = [0u8; 20];
    let mut carry: u16 = 0;
    for i in (0..20).rev() {
        let sum = a_bytes[i] as u16 + b_bytes[i] as u16 + carry;
        result[i] = sum as u8;
        carry = sum >> 8;
    }
    Address::new(result)
}

/// Whether a transaction type uses address aliasing.
pub fn does_tx_type_alias(tx_type: u8) -> bool {
    // ArbitrumUnsignedTx = 0x65, ArbitrumContractTx = 0x66, ArbitrumRetryTx = 0x68
    matches!(tx_type, 0x65 | 0x66 | 0x68)
}

/// Whether a transaction type incurs L1 poster costs and standard fee distribution.
///
/// Matches Nitro's util.TxTypeHasPosterCosts: only standard EOA-signed tx types
/// (Legacy / EIP-2930 / EIP-1559) pay poster costs. All Arbitrum-specific tx
/// types (deposit, unsigned, contract, retry, submit-retryable, internal) skip
/// L1 charging.
pub fn tx_type_has_poster_costs(tx_type: u8) -> bool {
    !matches!(
        tx_type,
        0x64  // ArbitrumDepositTx
        | 0x65 // ArbitrumUnsignedTx
        | 0x66 // ArbitrumContractTx
        | 0x68 // ArbitrumRetryTx
        | 0x69 // ArbitrumSubmitRetryableTx
        | 0x6a // ArbitrumInternalTx
    )
}
