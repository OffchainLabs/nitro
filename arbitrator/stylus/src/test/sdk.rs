// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::field_reassign_with_default)]

use crate::test::{random_bytes20, random_bytes32, run_native, test_configs, TestInstance};
use eyre::Result;
use num_bigint::BigUint;

#[test]
fn test_sdk_routes() -> Result<()> {
    let filename = "tests/erc20/target/wasm32-unknown-unknown/release/erc20.wasm";

    macro_rules! hex {
        ($($hex:expr),+) => {
            hex::decode(&format!($($hex),+))?
        };
    }

    let (compile, config, ink) = test_configs();
    let (mut native, mut evm) = TestInstance::new_with_evm(filename, &compile, config)?;

    // deploy a copy to another address
    let imath = random_bytes20();
    evm.deploy(imath, config, "erc20")?;

    // call balanceOf(0x000..000)
    let calldata = hex!("70a082310000000000000000000000000000000000000000000000000000000000000000");
    let output = run_native(&mut native, &calldata, ink)?;
    assert_eq!(output, [0; 32]);

    // call mint()
    let calldata = hex!("1249c58b");
    let output = run_native(&mut native, &calldata, ink)?;
    assert!(output.is_empty());

    macro_rules! big {
        ($int:expr) => {
            &format!("{:0>64}", $int.to_str_radix(16))
        };
    }

    // sumWithHelper(imath, values)
    let imath = BigUint::from_bytes_be(&imath.0);
    let count = 10_u8;
    let method = "168261a9"; // sumWithHelper
    let mut calldata = hex!(
        "{method}{}{}{}",
        big!(imath),
        big!(BigUint::from(64_u8)),
        big!(BigUint::from(count))
    );

    let mut sum = BigUint::default();
    for _ in 0..count {
        let value = BigUint::from_bytes_be(&random_bytes32().0);
        calldata.extend(hex!("{}", big!(value)));
        sum += value;
    }
    sum %= BigUint::from(2_u8).pow(256);

    let output = run_native(&mut native, &calldata, ink)?;
    assert_eq!(&hex::encode(output), big!(sum));
    Ok(())
}
