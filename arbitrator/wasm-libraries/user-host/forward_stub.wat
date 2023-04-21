;; Copyright 2022-2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (func (export "forward__read_args")             (param i32) unreachable)
    (func (export "forward__return_data")           (param i32 i32) unreachable)
    (func (export "forward__account_load_bytes32")  (param i32 i32) unreachable)
    (func (export "forward__account_store_bytes32") (param i32 i32) unreachable))
