// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{Bytes20, address, block, contract, evm, msg, tx};

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
    let block: u64 = 4;
    let blockhash = evm::blockhash(block.into());
    let basefee = block::basefee();
    let chainid = block::chainid();
    let coinbase = block::coinbase();
    let difficulty = block::difficulty();
    let gas_limit = block::gas_limit();
    let block_number = block::number();
    let timestamp = block::timestamp();
    let address = contract::address();
    let sender = msg::sender();
    let value = msg::value();
    let origin = tx::origin();
    let gas_price = tx::gas_price();
    let ink_price = tx::ink_price();
    let gas_left_before = evm::gas_left();
    let ink_left_before = evm::ink_left();

    // Call burnArbGas
    contract::call(arb_test_addr, burn_call_data, None, None)?;
    let gas_left_after = evm::gas_left();
    let ink_left_after = evm::ink_left();

    let mut output = vec![];
    output.extend(address_balance.unwrap_or_default());
    output.extend(eth_precompile_codehash.unwrap_or_default());
    output.extend(arb_precompile_codehash.unwrap_or_default());
    output.extend(contract_codehash.unwrap_or_default());
    output.extend(blockhash.unwrap_or_default());
    output.extend(basefee);
    output.extend(chainid);
    output.extend(coinbase);
    output.extend(difficulty);
    output.extend(gas_limit.to_be_bytes());
    output.extend(block_number);
    output.extend(timestamp);
    output.extend(address);
    output.extend(sender);
    output.extend(value);
    output.extend(origin);
    output.extend(gas_price);
    output.extend(ink_price.to_be_bytes());
    output.extend(gas_left_before.to_be_bytes());
    output.extend(ink_left_before.to_be_bytes());
    output.extend(gas_left_after.to_be_bytes());
    output.extend(ink_left_after.to_be_bytes());
    Ok(output)
}
