### Internal
- System tests now drive address filtering through the production S3 pipeline (backed by an in-process fake S3 server) instead of injecting in-memory checkers; the unused `ExecutionEngine.SetAddressChecker` and `TxPreChecker.SetTxFiltererForTest` test hooks were removed.
