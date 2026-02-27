use thiserror::Error;
use wasmer::MemoryAccessError;

/// Error type for host I/O operations during WASM execution.
#[derive(Error, Debug)]
pub enum Escape {
    #[error("failed to access memory: {0}")]
    Memory(#[from] MemoryAccessError),

    #[error("internal error: {0}")]
    Internal(String),

    #[error("logic error: {0}")]
    Logical(String),

    #[error("out of ink")]
    OutOfInk,

    #[error("exit early: {0}")]
    Exit(u32),
}

impl Escape {
    pub fn internal<T>(error: impl Into<String>) -> Result<T, Escape> {
        Err(Self::Internal(error.into()))
    }

    pub fn logical<T>(error: impl Into<String>) -> Result<T, Escape> {
        Err(Self::Logical(error.into()))
    }

    pub fn out_of_ink<T>() -> Result<T, Escape> {
        Err(Self::OutOfInk)
    }
}

impl From<std::io::Error> for Escape {
    fn from(err: std::io::Error) -> Self {
        Self::Internal(err.to_string())
    }
}

pub type MaybeEscape = Result<(), Escape>;
