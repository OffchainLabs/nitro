// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::{Address, B256, U256},
    block,
    contract::{self, Call},
    evm, msg, tx,
    types::AddressVM,
};

stylus_sdk::entrypoint!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let balance_check_addr = Address::try_from(&input[..20]).unwrap();
    let eth_precompile_addr = Address::try_from(&input[20..40]).unwrap();
    let arb_test_addr = Address::try_from(&input[40..60]).unwrap();
    let contract_addr = Address::try_from(&input[60..80]).unwrap();
    let burn_call_data = &input[80..];

    let address_balance = balance_check_addr.balance();
    let eth_precompile_codehash = eth_precompile_addr.codehash();
    let arb_precompile_codehash = arb_test_addr.codehash();
    let contract_codehash = contract_addr.codehash();
    let basefee = block::basefee();
    let chainid = block::chainid();
    let coinbase = block::coinbase();
    let gas_limit = block::gas_limit();
    let timestamp = block::timestamp();
    let address = contract::address();
    let sender = msg::sender();
    let value = msg::value();
    let origin = tx::origin();
    let gas_price = tx::gas_price();
    let ink_price = tx::ink_price();

    let mut block_number = block::number();
    block_number -= 1;

    // Call burnArbGas
    let gas_left_before = evm::gas_left();
    let ink_left_before = evm::ink_left();
    Call::new().call(arb_test_addr, burn_call_data)?;
    let gas_left_after = evm::gas_left();
    let ink_left_after = evm::ink_left();

    let mut output = vec![];
    output.extend(B256::from(U256::from(block_number)));
    output.extend(B256::from(U256::from(chainid)));
    output.extend(basefee);
    output.extend(gas_price);
    output.extend(B256::from(U256::from(gas_limit)));
    output.extend(value);
    output.extend(B256::from(U256::from(timestamp)));
    output.extend(address_balance);

    output.extend(address.into_word());
    output.extend(sender.into_word());
    output.extend(origin.into_word());
    output.extend(coinbase.into_word());

    output.extend(contract_codehash.unwrap_or_default());
    output.extend(arb_precompile_codehash.unwrap_or_default());
    output.extend(eth_precompile_codehash.unwrap_or_default());

    output.extend(ink_price.to_be_bytes());
    output.extend(gas_left_before.to_be_bytes());
    output.extend(ink_left_before.to_be_bytes());
    output.extend(gas_left_after.to_be_bytes());
    output.extend(ink_left_after.to_be_bytes());
    Ok(output)
}
