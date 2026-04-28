use anyhow::{Context, Result};

use arb_spec_tests::ExecutionFixture;

use super::PromoteArgs;

pub fn run(args: PromoteArgs) -> Result<()> {
    let _parsed: ExecutionFixture = ExecutionFixture::load(&args.captured)
        .with_context(|| format!("validate {}", args.captured.display()))?;

    if let Some(parent) = args.committed.parent() {
        if !parent.as_os_str().is_empty() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("create parent dir {}", parent.display()))?;
        }
    }

    if let Err(e) = std::fs::rename(&args.captured, &args.committed) {
        // Cross-device rename: fall back to copy + remove.
        std::fs::copy(&args.captured, &args.committed).with_context(|| {
            format!(
                "copy {} -> {} (rename failed: {e})",
                args.captured.display(),
                args.committed.display()
            )
        })?;
        std::fs::remove_file(&args.captured)
            .with_context(|| format!("remove source {}", args.captured.display()))?;
    }

    println!(
        "promoted {} -> {}",
        args.captured.display(),
        args.committed.display()
    );

    if let Some(rpc) = args.rpc.as_deref() {
        let fixture = ExecutionFixture::load(&args.committed)
            .with_context(|| format!("reload committed {}", args.committed.display()))?;
        fixture
            .run(rpc)
            .map_err(|e| anyhow::anyhow!("post-promote verify failed: {e}"))?;
        println!("OK {} (verified against {rpc})", args.committed.display());
    }

    Ok(())
}
