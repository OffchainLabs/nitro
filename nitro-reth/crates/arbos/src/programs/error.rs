use arb_storage::StorageError;

/// Errors raised by the Stylus programs subsystem.
#[derive(Clone, thiserror::Error, Debug)]
pub enum ProgramsError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// `open_program_at` was called for a code hash that has not been
    /// activated (its stored version is `0`).
    #[error("program is not activated")]
    NotActivated,

    /// The program's stored version does not match the version recorded in
    /// the current Stylus params.
    #[error("program version mismatch: program={program} params={params}")]
    VersionMismatch {
        /// Version recorded on the program itself.
        program: u64,
        /// Version expected by the active Stylus params.
        params: u64,
    },

    /// The program's age exceeds the configured expiry window.
    #[error("program has expired")]
    Expired,

    /// `upgrade_to_version` / `upgrade_to_arbos_version` was called from a
    /// state that does not satisfy the required precondition (e.g. trying to
    /// upgrade past a version that hasn't been reached).
    #[error("invalid stylus params upgrade: {0}")]
    InvalidParamsUpgrade(&'static str),

    /// `activate_program` was invoked for a program that is already at the
    /// current Stylus version and within its expiry window.
    #[error("program is already up to date")]
    UpToDate,

    /// `program_keepalive` was invoked before enough time has elapsed since
    /// the last activation/keepalive.
    #[error("program keepalive called too soon")]
    KeepaliveTooSoon,

    /// The program's stored version is older than the active Stylus version
    /// and an upgrade is required before this operation.
    #[error("program needs upgrade before this operation")]
    NeedsUpgrade,

    /// A Stylus call was attempted with insufficient gas to cover the call
    /// cost (memory + init/cached gas).
    #[error("insufficient gas for stylus call: need {needed}, have {have}")]
    InsufficientGas {
        /// Gas required to cover the call cost.
        needed: u64,
        /// Gas available at the call site.
        have: u64,
    },

    /// The Stylus call returned data whose EVM-equivalent memory cost
    /// exceeds the gas that was available at call entry. All remaining gas
    /// is consumed.
    #[error("stylus call out of gas (return data exceeds available)")]
    ReturnDataOutOfGas,

    /// The prover-machine activation step (WASM compile/validate) failed
    /// with a runtime-supplied diagnostic.
    #[error("stylus activation failed: {reason}")]
    Activation {
        /// Diagnostic message preserved from the activation backend.
        reason: String,
    },

    /// The user-supplied call dispatcher reported a runtime error (e.g. a
    /// trap or non-success outcome).
    #[error("stylus call execution failed: {reason}")]
    Execution {
        /// Diagnostic message preserved from the Stylus runtime.
        reason: String,
    },
}
