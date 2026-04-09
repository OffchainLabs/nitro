mod common;

use arb_precompiles::create_arbbls_precompile;
use common::{calldata, PrecompileTest};

#[test]
fn arbbls_has_no_methods() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&create_arbbls_precompile(), &calldata("anything()", &[]));
    assert!(run.assert_ok().reverted);
}
