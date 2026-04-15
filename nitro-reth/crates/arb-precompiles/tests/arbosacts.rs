mod common;

use arb_precompiles::create_arbosacts_precompile;
use common::{calldata, PrecompileTest};

#[test]
fn rejects_all_calls_with_caller_not_arbos() {
    for sig in [
        "startBlock(uint256,uint64,uint64,uint64)",
        "batchPostingReport(uint256,address,uint64,uint64,uint256)",
        "batchPostingReportV2(uint256,address,uint64,uint64,uint64,uint64,uint256)",
    ] {
        let run = PrecompileTest::new()
            .arbos_version(30)
            .arbos_state()
            .call(&create_arbosacts_precompile(), &calldata(sig, &[]));
        assert!(run.result.is_err(), "{sig} must error");
    }
}
