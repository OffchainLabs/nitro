mod common;

use alloy_primitives::B256;
use arb_precompiles::create_arbosacts_precompile;
use common::{calldata, PrecompileTest};

#[test]
fn valid_calls_revert_with_caller_not_arbos() {
    for (sig, num_args) in [
        ("startBlock(uint256,uint64,uint64,uint64)", 4),
        (
            "batchPostingReport(uint256,address,uint64,uint64,uint256)",
            5,
        ),
        (
            "batchPostingReportV2(uint256,address,uint64,uint64,uint64,uint64,uint256)",
            7,
        ),
    ] {
        let args = vec![B256::ZERO; num_args];
        let run = PrecompileTest::new()
            .arbos_version(30)
            .arbos_state()
            .call(create_arbosacts_precompile, &calldata(sig, &args));
        let out = run.assert_ok();
        assert!(out.reverted, "{sig} must revert");
        assert_eq!(
            out.bytes.as_ref(),
            [0xf8u8, 0x12, 0xe6, 0x56].as_slice(),
            "{sig} must revert with CallerNotArbOS()"
        );
    }
}
