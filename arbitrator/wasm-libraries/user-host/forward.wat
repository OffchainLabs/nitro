;; Copyright 2022-2023, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (import "user_host" "arbitrator_forward__read_args"             (func $read_args             (param i32)))
    (import "user_host" "arbitrator_forward__return_data"           (func $return_data           (param i32 i32)))
    (import "user_host" "arbitrator_forward__account_load_bytes32"  (func $account_load_bytes32  (param i32 i32)))
    (import "user_host" "arbitrator_forward__account_store_bytes32" (func $account_store_bytes32 (param i32 i32)))
    (export "forward__read_args"             (func $read_args))
    (export "forward__return_data"           (func $return_data))
    (export "forward__account_load_bytes32"  (func $account_load_bytes32))
    (export "forward__account_store_bytes32" (func $account_store_bytes32)))
