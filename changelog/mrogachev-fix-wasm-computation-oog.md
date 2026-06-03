### Fixed
- Stylus pre-VM OOG: when `contract.BurnGas(callCost)` drains the call frame before WASM runs, attribute the residual to `WasmComputation` so `receipt.MultiGasUsed.SingleGas()` matches `receipt.GasUsed`.
