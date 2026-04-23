//! Compile-time ABI for the Arbitrum precompiles, generated from the pinned
//! `nitro-precompile-interfaces` and `nitro-contracts` submodules via `sol!`.

macro_rules! sol_module {
    ($mod:ident, $path:literal) => {
        #[allow(missing_docs, non_snake_case, non_camel_case_types)]
        mod $mod {
            alloy_sol_types::sol!($path);
        }
    };
}

sol_module!(arbsys, ".gen/ArbSys.sol");
sol_module!(arbinfo, ".gen/ArbInfo.sol");
sol_module!(arbstatistics, ".gen/ArbStatistics.sol");
sol_module!(arbostest, ".gen/ArbosTest.sol");
sol_module!(arbfunctiontable, ".gen/ArbFunctionTable.sol");
sol_module!(
    arbfilteredtxmanager,
    ".gen/ArbFilteredTransactionsManager.sol"
);
sol_module!(arbnativetokenmanager, ".gen/ArbNativeTokenManager.sol");
sol_module!(arbwasmcache, ".gen/ArbWasmCache.sol");
sol_module!(arbdebug, ".gen/ArbDebug.sol");
sol_module!(arbaddresstable, ".gen/ArbAddressTable.sol");
sol_module!(arbaggregator, ".gen/ArbAggregator.sol");
sol_module!(arbretryabletx, ".gen/ArbRetryableTx.sol");
sol_module!(arbwasm, ".gen/ArbWasm.sol");
sol_module!(arbownerpublic, ".gen/ArbOwnerPublic.sol");
sol_module!(arbgasinfo, ".gen/ArbGasInfo.sol");
sol_module!(arbowner, ".gen/ArbOwner.sol");
sol_module!(nodeinterface, ".gen/NodeInterface.sol");
sol_module!(nodeinterfacedebug, ".gen/NodeInterfaceDebug.sol");

pub use arbaddresstable::ArbAddressTable as IArbAddressTable;
pub use arbaggregator::ArbAggregator as IArbAggregator;
pub use arbdebug::ArbDebug as IArbDebug;
pub use arbfilteredtxmanager::ArbFilteredTransactionsManager as IArbFilteredTxManager;
pub use arbfunctiontable::ArbFunctionTable as IArbFunctionTable;
pub use arbgasinfo::ArbGasInfo as IArbGasInfo;
pub use arbinfo::ArbInfo as IArbInfo;
pub use arbnativetokenmanager::ArbNativeTokenManager as IArbNativeTokenManager;
pub use arbostest::ArbosTest as IArbosTest;
pub use arbowner::ArbOwner as IArbOwner;
pub use arbownerpublic::ArbOwnerPublic as IArbOwnerPublic;
pub use arbretryabletx::ArbRetryableTx as IArbRetryableTx;
pub use arbstatistics::ArbStatistics as IArbStatistics;
pub use arbsys::ArbSys as IArbSys;
pub use arbwasm::ArbWasm as IArbWasm;
pub use arbwasmcache::ArbWasmCache as IArbWasmCache;
pub use nodeinterface::NodeInterface as INodeInterface;
pub use nodeinterfacedebug::NodeInterfaceDebug as INodeInterfaceDebug;
