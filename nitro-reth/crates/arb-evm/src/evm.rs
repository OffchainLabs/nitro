use alloy_evm::{
    eth::EthEvmContext, precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory,
};
use alloy_primitives::{Address, Bytes};
use core::fmt::Debug;
use revm::context::result::EVMError;
use revm::context_interface::result::{HaltReason, ResultAndState};
use revm::inspector::NoOpInspector;
use revm::primitives::hardfork::SpecId;

use crate::transaction::ArbTransaction;

/// Arbitrum EVM wrapper that registers custom precompiles.
pub struct ArbEvm<DB: Database + Debug, I> {
    inner: alloy_evm::EthEvm<DB, I, PrecompilesMap>,
}

impl<DB, I> ArbEvm<DB, I>
where
    DB: Database + Debug,
{
    pub fn new(inner: alloy_evm::EthEvm<DB, I, PrecompilesMap>) -> Self {
        Self { inner }
    }

    pub fn into_inner(self) -> alloy_evm::EthEvm<DB, I, PrecompilesMap> {
        self.inner
    }
}

impl<DB, I> Evm for ArbEvm<DB, I>
where
    DB: Database + Debug,
    I: revm::inspector::Inspector<EthEvmContext<DB>>,
{
    type DB = DB;
    type Tx = ArbTransaction;
    type Error = EVMError<<DB as revm::Database>::Error>;
    type HaltReason = HaltReason;
    type Spec = SpecId;
    type Precompiles = PrecompilesMap;
    type Inspector = I;
    type BlockEnv = revm::context::BlockEnv;

    fn block(&self) -> &revm::context::BlockEnv {
        self.inner.block()
    }

    fn chain_id(&self) -> u64 {
        self.inner.chain_id()
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.transact_raw(tx.0)
    }

    fn transact_system_call(
        &mut self,
        caller: Address,
        contract: Address,
        data: Bytes,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.transact_system_call(caller, contract, data)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec>) {
        self.inner.finish()
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inner.set_inspector_enabled(enabled)
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        self.inner.components()
    }

    fn components_mut(
        &mut self,
    ) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        self.inner.components_mut()
    }
}

/// Factory for creating Arbitrum EVM instances with custom precompiles.
#[derive(Default, Debug, Clone, Copy)]
pub struct ArbEvmFactory(pub alloy_evm::EthEvmFactory);

impl ArbEvmFactory {
    pub fn new() -> Self {
        Self::default()
    }
}

impl EvmFactory for ArbEvmFactory {
    type Evm<DB: Database, I: revm::inspector::Inspector<EthEvmContext<DB>>> = ArbEvm<DB, I>;
    type Context<DB: Database> = EthEvmContext<DB>;
    type Tx = ArbTransaction;
    type Error<DBError: core::error::Error + Send + Sync + 'static> = EVMError<DBError>;
    type HaltReason = HaltReason;
    type Spec = SpecId;
    type Precompiles = PrecompilesMap;
    type BlockEnv = revm::context::BlockEnv;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let evm = ArbEvm::new(self.0.create_evm(db, input));
        // TODO: Register Arbitrum precompiles (ArbSys, ArbGasInfo, etc.)
        evm
    }

    fn create_evm_with_inspector<DB: Database, I: revm::inspector::Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let evm = ArbEvm::new(self.0.create_evm_with_inspector(db, input, inspector));
        // TODO: Register Arbitrum precompiles
        evm
    }
}
