use anyhow::{Context, Result};

use arb_spec_tests::ExecutionFixture;

use crate::cli::VerifyArgs;

pub fn run(args: VerifyArgs) -> Result<()> {
    let fixture = ExecutionFixture::load(&args.fixture)
        .with_context(|| format!("load fixture {}", args.fixture.display()))?;

    fixture
        .run(args.rpc.as_str())
        .map_err(|e| anyhow::anyhow!("verify failed: {e}"))?;

    println!("OK {}", args.fixture.display());
    Ok(())
}
