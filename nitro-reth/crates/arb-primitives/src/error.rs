use thiserror::Error;

#[derive(Debug, Error)]
pub enum ArbError {
    #[error("invalid ArbOS version: {0}")]
    InvalidArbOSVersion(u64),
    #[error("storage error: {0}")]
    Storage(String),
    #[error("execution error: {0}")]
    Execution(String),
    #[error("state error: {0}")]
    State(String),
    #[error("pricing error: {0}")]
    Pricing(String),
    #[error("precompile error: {0}")]
    Precompile(String),
    #[error("rlp decode error: {0}")]
    RlpDecode(#[from] alloy_rlp::Error),
}
