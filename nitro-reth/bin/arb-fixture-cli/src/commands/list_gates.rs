use std::path::PathBuf;

use anyhow::Result;

use crate::cli::ListGatesArgs;

const DEFAULT_INVENTORY: &str = ".plan/gate-inventory.md";

pub fn run(args: ListGatesArgs) -> Result<()> {
    let path: PathBuf = args
        .inventory
        .unwrap_or_else(|| PathBuf::from(DEFAULT_INVENTORY));

    match std::fs::read_to_string(&path) {
        Ok(body) => {
            print!("{body}");
            if !body.ends_with('\n') {
                println!();
            }
            Ok(())
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            println!(
                "no inventory at {}; run xtask regen-inventory",
                path.display()
            );
            Ok(())
        }
        Err(e) => Err(anyhow::anyhow!("read {}: {e}", path.display())),
    }
}
