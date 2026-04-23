//! Arbitrum EVM execution layer.
//!
//! Block executor, custom opcode handlers, EVM configuration, and receipt
//! building for Arbitrum's modified execution environment.

extern crate alloc;

pub mod assembler;
pub mod build;
pub mod config;
pub mod context;
pub mod evm;
pub mod executor;
pub mod hooks;
pub mod receipt;
pub mod state_overlay;
pub mod transaction;

pub use assembler::ArbBlockAssembler;
pub use build::{
    ArbBlockExecutor, ArbBlockExecutorFactory, ArbScheduledTxDrain, ArbTransactionEnv,
};
pub use config::ArbEvmConfig;
pub use context::{
    ActivatedWasm, ArbBlockExecutionCtx, ArbNextBlockEnvCtx, ArbitrumExtraData, RecentWasms,
};
pub use evm::{ArbEvm, ArbEvmFactory};
pub use executor::DefaultArbOsHooks;
pub use hooks::{ArbOsHooks, NoopArbOsHooks};
pub use receipt::ArbReceiptBuilder;
pub use transaction::ArbTransaction;
