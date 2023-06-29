// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{address, block, call::Call, contract, evm, msg, tx, Bytes20, Bytes32};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let balance_check_addr = Bytes20::from_slice(&input[..20]).unwrap();
    let eth_precompile_addr = Bytes20::from_slice(&input[20..40]).unwrap();
    let arb_test_addr = Bytes20::from_slice(&input[40..60]).unwrap();
    let contract_addr = Bytes20::from_slice(&input[60..80]).unwrap();
    let burn_call_data = &input[80..];

    let address_balance = address::balance(balance_check_addr);
    let eth_precompile_codehash = address::codehash(eth_precompile_addr);
    let arb_precompile_codehash = address::codehash(arb_test_addr);
    let contract_codehash = address::codehash(contract_addr);
    let basefee = block::basefee();
    let chainid = block::chainid();
    let coinbase = block::coinbase();
    let difficulty = block::difficulty();
    let gas_limit = block::gas_limit();
    let timestamp = block::timestamp();
    let address = contract::address();
    let sender = msg::sender();
    let value = msg::value();
    let origin = tx::origin();
    let gas_price = tx::gas_price();
    let ink_price = tx::ink_price();

    let mut block_number = block::number();
    block_number[31] -= 1;
    let blockhash = evm::blockhash(block_number);

    // Call burnArbGas
    let gas_left_before = evm::gas_left();
    let ink_left_before = evm::ink_left();
    Call::new().call(arb_test_addr, burn_call_data)?;
    let gas_left_after = evm::gas_left();
    let ink_left_after = evm::ink_left();

    let mut output = vec![];
    output.extend(block_number);
    output.extend(blockhash.unwrap_or_default());
    output.extend(chainid);
    output.extend(basefee);
    output.extend(gas_price);
    output.extend(Bytes32::from(gas_limit));
    output.extend(value);
    output.extend(difficulty);
    output.extend(Bytes32::from(timestamp));
    output.extend(address_balance);

    output.extend(Bytes32::from(address));
    output.extend(Bytes32::from(sender));
    output.extend(Bytes32::from(origin));
    output.extend(Bytes32::from(coinbase));

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
