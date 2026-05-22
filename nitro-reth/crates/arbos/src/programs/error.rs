use arb_storage::StorageError;

/// Errors raised by the Stylus programs subsystem.
#[derive(thiserror::Error, Debug)]
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
}
