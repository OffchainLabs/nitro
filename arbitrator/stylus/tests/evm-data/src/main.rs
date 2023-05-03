// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::{Bytes20, Bytes32, address, block, contract, evm, msg, tx};

arbitrum::arbitrum_main!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let balance_check_addr = Bytes20::from_slice(&input[0..20]).expect("incorrect slice size for Bytes20");
    let arb_test_addr = Bytes20::from_slice(&input[20..40]).expect("incorrect slice size for Bytes20");
    let burn_call_data = &input[40..];

    let address_balance = address::balance(balance_check_addr);
    let address_codehash = address::codehash(arb_test_addr);
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
    let mut extend_optional = |a: Option<Bytes32>| {
        match a {
            Some(data) => output.extend(data.0),
            None => {
                let data = [0; 32];
                output.extend(data)
            }
        }
    };
    extend_optional(address_balance);
    extend_optional(address_codehash);
    extend_optional(blockhash);
    output.extend(basefee.0);
    output.extend(chainid.0);
    output.extend(coinbase.0);
    output.extend(difficulty.0);
    output.extend(gas_limit.to_be_bytes());
    output.extend(block_number.0);
    output.extend(timestamp.0);
    output.extend(address.0);
    output.extend(sender.0);
    output.extend(value.0);
    output.extend(origin.0);
    output.extend(gas_price.0);
    output.extend(ink_price.to_be_bytes());
    output.extend(gas_left_before.to_be_bytes());
    output.extend(ink_left_before.to_be_bytes());
    output.extend(gas_left_after.to_be_bytes());
    output.extend(ink_left_after.to_be_bytes());
    Ok(output)
}
