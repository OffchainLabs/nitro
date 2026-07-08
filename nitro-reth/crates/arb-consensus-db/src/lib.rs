pub mod kv;
pub mod schema;

#[derive(Debug, thiserror::Error)]
pub enum ConsensusDbError {}
