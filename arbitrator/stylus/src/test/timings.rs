// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::test::{run_native, test_configs, TestInstance};
use arbutil::{color::Color, format};
use eyre::Result;
use std::time::Instant;

#[test]
fn test_timings() -> Result<()> {
    let (mut compile, config, ink) = test_configs();
    compile.debug.count_ops = false;

    #[rustfmt::skip]
    let basic = [
        // simple returns
        "null_host", "return_data_size", "block_gas_limit", "block_timestamp", "tx_ink_price",

        // gas left
        "evm_gas_left", "evm_ink_left",

        // evm data
        "block_basefee", "chainid", "block_coinbase", "block_number", "contract_address",
        "msg_sender", "msg_value", "tx_gas_price", "tx_origin",
    ];

    let loops = ["read_args", "write_result", "keccak"];

    macro_rules! run {
        ($rounds:expr, $args:expr, $file:expr) => {{
            let mut native = TestInstance::new_linked(&$file, &compile, config)?;
            let before = Instant::now();
            run_native(&mut native, &$args, ink)?;
            let time = before.elapsed() / $rounds;
            let cost = time.as_nanos() as f64 / 10.39; // 10.39 from Rachel's desktop
            let ink = format!("{}", (cost * 10000.).ceil() as usize).grey();
            (format::time(time), format!("{cost:.4}").grey(), ink)
        }};
    }

    macro_rules! args {
        ($rounds:expr, $len:expr) => {{
            let mut args = $rounds.to_le_bytes().to_vec();
            args.extend(vec![1; $len - 4]);
            args
        }};
    }

    println!("Timings hostios. Please note the values derived are machine dependent.\n");

    println!("\n{}", format!("Hostio timings").pink());
    for name in basic {
        let file = format!("tests/timings/{name}.wat");
        let rounds: u32 = 50_000_000;
        let (time, cost, ink) = run!(rounds, rounds.to_le_bytes(), file);
        println!("{} {time} {cost} {ink}", format!("{name:16}").grey());
    }

    for name in loops {
        println!("\n{}", format!("{name} timings").pink());
        for i in 2..10 {
            let file = format!("tests/timings/{name}.wat");
            let rounds: u32 = 10_000_000;
            let size = 1 << i;
            let args = args!(rounds, size);

            let (time, cost, ink) = run!(rounds, args, file);
            let name = format!("{name}({size:03})").grey();
            println!("{name} {time} {cost} {ink}",);
        }
    }
    Ok(())
}
