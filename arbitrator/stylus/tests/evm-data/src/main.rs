// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use arbitrum::block;
use arbitrum::evm;
use arbitrum::msg;
use arbitrum::tx;

arbitrum::arbitrum_main!(user_main);

fn user_main(_input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let block: u64 = 4;
    let blockhash = evm::blockhash(block.into());
    let basefee = block::basefee();
    let chainid = block::chainid();
    let coinbase = block::coinbase();
    let difficulty = block::difficulty();
    let gas_limit = block::gas_limit();
    let block_number = block::number();
    let timestamp = block::timestamp();
    let sender = msg::sender();
    let value = msg::value();
    let gas_price = tx::gas_price();
    let origin = tx::origin();

    let mut output = vec![];
    match blockhash {
        Some(hash) => output.extend(hash.0),
        None => {
            let data = [0; 32];
            output.extend(data)
        }
    }

    output.extend(basefee.0);
    output.extend(chainid.0);
    output.extend(coinbase.0);
    output.extend(difficulty.0);
    output.extend(gas_limit.to_be_bytes());
    output.extend(block_number.0);
    output.extend(timestamp.0);
    output.extend(sender.0);
    output.extend(value.0);
    output.extend(gas_price.0);
    output.extend(origin.0);
    Ok(output)
}
