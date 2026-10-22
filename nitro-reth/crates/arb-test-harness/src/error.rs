use thiserror::Error;

pub type Result<T> = std::result::Result<T, HarnessError>;

#[derive(Debug, Error)]
pub enum HarnessError {
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),

    #[error("json error: {0}")]
    Json(#[from] serde_json::Error),

    #[error("rpc error: {0}")]
    Rpc(String),

    #[error("node startup timeout: {what}")]
    StartupTimeout { what: &'static str },

    #[error("node exited unexpectedly: {kind} (code={code:?})")]
    NodeExited {
        kind: &'static str,
        code: Option<i32>,
    },

    #[error("missing env var: {name}")]
    MissingEnv { name: &'static str },

    #[error("unsupported message kind: {kind}")]
    UnsupportedKind { kind: u8 },

    #[error("scenario assertion failed: {0}")]
    Assertion(String),

    #[error("invalid input: {0}")]
    Invalid(String),

    #[error("not implemented: {what}")]
    NotImplemented { what: &'static str },
}
