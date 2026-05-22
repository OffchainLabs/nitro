//! Error types raised by the Stylus runtime.
//!
//! All public APIs return [`StylusError`]. Host-function failures and
//! module-lifecycle failures share one enum because both flow through the
//! same surfaces (host functions trap; lifecycle errors revert the EVM
//! frame) and callers benefit from a single match site.

use thiserror::Error;
use wasmer::MemoryAccessError;

/// All failure modes raised by the Stylus runtime.
#[derive(Error, Debug)]
pub enum StylusError {
    /// WASM linear-memory access (read/write) failed, typically because a
    /// host function received an out-of-bounds pointer from the program.
    #[error("memory access failed: {0}")]
    Memory(#[from] MemoryAccessError),

    /// A host function detected an unrecoverable internal condition (e.g.
    /// the underlying journal returned an error during sload/sstore).
    /// Surfaces from the WASM trap as `UserOutcome::Failure`.
    #[error("internal stylus error: {0}")]
    Internal(String),

    /// A host function rejected its inputs on semantic grounds (e.g.
    /// `emit_log` called in a static context, malformed topic data).
    /// Surfaces as `UserOutcome::Failure`.
    #[error("stylus logical error: {0}")]
    Logical(String),

    /// The program exhausted its ink budget mid-host-call. Surfaces as
    /// `UserOutcome::OutOfInk`.
    #[error("out of ink")]
    OutOfInk,

    /// The program returned voluntarily via `exit_early(status)`. A
    /// non-zero status is reported as `UserOutcome::Revert`.
    #[error("program exited with status {0}")]
    Exit(u32),

    /// Wasmer module compilation or serialization failed. Carries the
    /// rendered wasmer error since the underlying type varies across
    /// compilation backends.
    #[error("wasm compile failed: {0}")]
    Compile(String),

    /// Wasmer instance creation failed (import binding, memory export, or
    /// start function).
    #[error("wasm instantiation failed: {0}")]
    Instantiation(String),

    /// `activate_program` failed at the prover-machine level. Wraps the
    /// rendered error from the activation pipeline.
    #[error("program activation failed: {0}")]
    Activation(String),

    /// The contract bytecode does not satisfy Stylus structural
    /// requirements (e.g. missing discriminant or unsupported dictionary).
    #[error("invalid stylus program: {0}")]
    InvalidProgram(&'static str),

    /// Brotli decompression of the Stylus payload failed.
    #[error("brotli decompression failed: {0}")]
    Decompression(String),

    /// `TypedFunction::call` returned a non-trap error that could not be
    /// downcast to [`StylusError`]. The original message is preserved.
    #[error("wasm execution failed: {0}")]
    Run(String),

    /// A required wasmer global export was missing or had the wrong type.
    #[error("missing or invalid wasm global: {0}")]
    MissingGlobal(String),
}

impl StylusError {
    /// Construct an [`Internal`](Self::Internal) failure in an `Err(_)`.
    pub fn internal<T>(error: impl Into<String>) -> Result<T, StylusError> {
        Err(Self::Internal(error.into()))
    }

    /// Construct a [`Logical`](Self::Logical) failure in an `Err(_)`.
    pub fn logical<T>(error: impl Into<String>) -> Result<T, StylusError> {
        Err(Self::Logical(error.into()))
    }

    /// Construct an [`OutOfInk`](Self::OutOfInk) failure in an `Err(_)`.
    pub fn out_of_ink<T>() -> Result<T, StylusError> {
        Err(Self::OutOfInk)
    }
}

impl From<std::io::Error> for StylusError {
    fn from(err: std::io::Error) -> Self {
        Self::Internal(err.to_string())
    }
}

/// Result alias used by host functions whose only meaningful return is
/// "success or trap".
pub type MaybeEscape = Result<(), StylusError>;
