// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![no_main]

extern crate alloc;

use stylus_sdk::{
    alloy_primitives::{Address, B256, U256},
    call::RawCall,
    host::VM,
    prelude::*,
};

#[entrypoint]
fn user_main(input: Vec<u8>, vm: VM) -> Result<Vec<u8>, Vec<u8>> {
    let balance_check_addr = Address::try_from(&input[..20]).unwrap();
    let eth_precompile_addr = Address::try_from(&input[20..40]).unwrap();
    let arb_test_addr = Address::try_from(&input[40..60]).unwrap();
    let contract_addr = Address::try_from(&input[60..80]).unwrap();
    let burn_call_data = &input[80..];

    let address_balance = vm.balance(balance_check_addr);
    let eth_precompile_codehash = vm.code_hash(eth_precompile_addr);
    let arb_precompile_codehash = vm.code_hash(arb_test_addr);
    let contract_codehash = vm.code_hash(contract_addr);

    let code = vm.code(contract_addr);
    assert_eq!(code.len(), vm.code_size(contract_addr));
    assert_eq!(vm.code_size(arb_test_addr), 1);
    assert_eq!(vm.code(arb_test_addr), [0xfe]);
    assert_eq!(vm.code_size(eth_precompile_addr), 0);
    assert_eq!(vm.code(eth_precompile_addr), []);

    let basefee = vm.block_basefee();
    let chainid = vm.chain_id();
    let coinbase = vm.block_coinbase();
    let gas_limit = vm.block_gas_limit();
    let timestamp = vm.block_timestamp();
    let address = vm.contract_address();
    let sender = vm.msg_sender();
    let value = vm.msg_value();
    let origin = vm.tx_origin();
    let gas_price = vm.tx_gas_price();
    let ink_price = vm.tx_ink_price();

    let mut block_number = vm.block_number();
    block_number -= 1;

    // Call burnArbGas
    let gas_left_before = vm.evm_gas_left();
    let ink_left_before = vm.evm_ink_left();
    unsafe { RawCall::new(&vm).call(arb_test_addr, burn_call_data)? };
    let gas_left_after = vm.evm_gas_left();
    let ink_left_after = vm.evm_ink_left();

    let mut output = vec![];
    output.extend(B256::from(U256::from(block_number)));
    output.extend(B256::from(U256::from(chainid)));
    output.extend(B256::from(basefee));
    output.extend(B256::from(gas_price));
    output.extend(B256::from(U256::from(gas_limit)));
    output.extend(B256::from(value));
    output.extend(B256::from(U256::from(timestamp)));
    output.extend(B256::from(address_balance));

    output.extend(address.into_word());
    output.extend(sender.into_word());
    output.extend(origin.into_word());
    output.extend(coinbase.into_word());

    output.extend(contract_codehash);
    output.extend(arb_precompile_codehash);
    output.extend(eth_precompile_codehash);
    output.extend(code);

    output.extend(ink_price.to_be_bytes());
    output.extend(gas_left_before.to_be_bytes());
    output.extend(ink_left_before.to_be_bytes());
    output.extend(gas_left_after.to_be_bytes());
    output.extend(ink_left_after.to_be_bytes());
    Ok(output)
}
