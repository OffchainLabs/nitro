mod common;

use alloy_evm::precompiles::DynPrecompile;
use arb_precompiles::create_arbdebug_precompile;
use common::{calldata, PrecompileTest};

fn arbdebug() -> DynPrecompile {
    create_arbdebug_precompile()
}

#[test]
fn debug_methods_revert_in_production() {
    for sig in [
        "events(uint256,bool,bytes32)",
        "eventsView()",
        "becomeChainOwner()",
        "panic()",
        "legacyError()",
    ] {
        let run = PrecompileTest::new()
            .arbos_version(30)
            .arbos_state()
            .call(&arbdebug(), &calldata(sig, &[]));
        assert!(
            run.assert_ok().reverted,
            "{sig} must revert when debug precompiles are disabled"
        );
    }
}
