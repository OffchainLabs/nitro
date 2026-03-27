use arb_primitives::multigas::{MultiGas, ResourceKind};

use crate::util::TracingInfo;

/// Gas burning abstraction for ArbOS operations.
///
/// Tracks multi-dimensional gas usage during ArbOS state modifications.
/// `SystemBurner` is used for internal ArbOS operations that don't have
/// a notion of remaining gas.
pub trait Burner {
    fn burn(&mut self, kind: ResourceKind, amount: u64) -> Result<(), BurnError>;
    fn burn_multi_gas(&mut self, amount: MultiGas) -> Result<(), BurnError>;
    fn burned(&self) -> u64;
    fn gas_left(&self) -> u64;
    fn burn_out(&mut self) -> Result<(), BurnError>;
    fn restrict(&mut self, err: BurnError);
    fn handle_error(&self, err: BurnError) -> Result<(), BurnError>;
    fn read_only(&self) -> bool;
    fn tracing_info(&self) -> Option<&TracingInfo>;
}

/// Error from a burn operation.
#[derive(Debug, Clone, thiserror::Error)]
pub enum BurnError {
    #[error("out of gas")]
    OutOfGas,
    #[error("restricted: {0}")]
    Restricted(String),
    #[error("{0}")]
    Other(String),
}

/// A burner for internal ArbOS system operations.
///
/// Has no concept of "gas left" — only tracks total gas burned.
/// Panics if `gas_left()` is called.
#[derive(Debug, Clone)]
pub struct SystemBurner {
    gas_burnt: MultiGas,
    tracing_info: Option<TracingInfo>,
    read_only: bool,
}

impl SystemBurner {
    pub fn new(tracing_info: Option<TracingInfo>, read_only: bool) -> Self {
        Self {
            gas_burnt: MultiGas::zero(),
            tracing_info,
            read_only,
        }
    }
}

impl Burner for SystemBurner {
    fn burn(&mut self, kind: ResourceKind, amount: u64) -> Result<(), BurnError> {
        self.gas_burnt.saturating_increment_into(kind, amount);
        Ok(())
    }

    fn burn_multi_gas(&mut self, amount: MultiGas) -> Result<(), BurnError> {
        self.gas_burnt.saturating_add_into(amount);
        Ok(())
    }

    fn burned(&self) -> u64 {
        self.gas_burnt.total()
    }

    fn gas_left(&self) -> u64 {
        unreachable!("SystemBurner has no notion of gas left")
    }

    fn burn_out(&mut self) -> Result<(), BurnError> {
        Err(BurnError::OutOfGas)
    }

    fn restrict(&mut self, _err: BurnError) {
        // SystemBurner ignores restrictions
    }

    fn handle_error(&self, err: BurnError) -> Result<(), BurnError> {
        Err(err)
    }

    fn read_only(&self) -> bool {
        self.read_only
    }

    fn tracing_info(&self) -> Option<&TracingInfo> {
        self.tracing_info.as_ref()
    }
}
