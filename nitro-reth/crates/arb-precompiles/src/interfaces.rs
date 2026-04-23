//! Compile-time ABI for Arbitrum precompiles.
//!
//! Solidity source is read from two pinned submodules under
//! `crates/arb-precompiles/`:
//! - `nitro-precompile-interfaces` — ArbOS precompile interfaces.
//! - `nitro-contracts`             — node-interface (RPC-only) virtual contracts.

macro_rules! sol_module {
    ($mod:ident, $path:literal) => {
        #[allow(missing_docs, non_snake_case, non_camel_case_types)]
        mod $mod {
            alloy_sol_types::sol!($path);
        }
    };
}

sol_module!(arbsys, "nitro-precompile-interfaces/ArbSys.sol");
sol_module!(arbinfo, "nitro-precompile-interfaces/ArbInfo.sol");
sol_module!(arbstatistics, "nitro-precompile-interfaces/ArbStatistics.sol");
sol_module!(arbostest, "nitro-precompile-interfaces/ArbosTest.sol");
sol_module!(arbfunctiontable, "nitro-precompile-interfaces/ArbFunctionTable.sol");
sol_module!(
    arbfilteredtxmanager,
    "nitro-precompile-interfaces/ArbFilteredTransactionsManager.sol"
);
sol_module!(
    arbnativetokenmanager,
    "nitro-precompile-interfaces/ArbNativeTokenManager.sol"
);
sol_module!(arbwasmcache, "nitro-precompile-interfaces/ArbWasmCache.sol");
sol_module!(arbdebug, "nitro-precompile-interfaces/ArbDebug.sol");
sol_module!(arbaddresstable, "nitro-precompile-interfaces/ArbAddressTable.sol");
sol_module!(arbaggregator, "nitro-precompile-interfaces/ArbAggregator.sol");
sol_module!(arbretryabletx, "nitro-precompile-interfaces/ArbRetryableTx.sol");
sol_module!(arbwasm, "nitro-precompile-interfaces/ArbWasm.sol");
sol_module!(arbownerpublic, "nitro-precompile-interfaces/ArbOwnerPublic.sol");
sol_module!(arbgasinfo, ".gen/ArbGasInfo.sol");
sol_module!(arbowner, ".gen/ArbOwner.sol");
sol_module!(nodeinterface, "nitro-contracts/src/node-interface/NodeInterface.sol");
sol_module!(nodeinterfacedebug, "nitro-contracts/src/node-interface/NodeInterfaceDebug.sol");

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
