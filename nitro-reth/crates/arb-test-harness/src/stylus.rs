#![cfg(feature = "stylus-wat")]

use crate::{error::HarnessError, Result};

pub fn wat_to_wasm(wat_source: &str) -> Result<Vec<u8>> {
    wat::parse_str(wat_source).map_err(|e| HarnessError::Invalid(format!("wat: {e}")))
}
