;; Copyright 2022-2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "user_host" "arbitrator_forward__read_args"              (func $read_args             (param i32)))
    (import "user_host" "arbitrator_forward__return_data"            (func $return_data           (param i32 i32)))
    (import "user_host" "arbitrator_forward__account_load_bytes32"   (func $account_load_bytes32  (param i32 i32)))
    (import "user_host" "arbitrator_forward__account_store_bytes32"  (func $account_store_bytes32 (param i32 i32)))
    (import "user_host" "arbitrator_forward__call_contract"
        (func $call_contract (param i32 i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__delegate_call_contract"
        (func $delegate_call (param i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__static_call_contract"
        (func $static_call   (param i32 i32 i32 i64 i32) (result i32)))
    (import "user_host" "arbitrator_forward__create1"          (func $create1 (param i32 i32 i32 i32 i32)))
    (import "user_host" "arbitrator_forward__create2"          (func $create2 (param i32 i32 i32 i32 i32 i32)))
    (import "user_host" "arbitrator_forward__read_return_data" (func $read_return_data (param i32)))
    (import "user_host" "arbitrator_forward__return_data_size" (func $return_data_size (result i32)))
    (import "user_host" "arbitrator_forward__emit_log"         (func $emit_log  (param i32 i32 i32)))
    (import "user_host" "arbitrator_forward__tx_origin"        (func $tx_origin (param i32)))
    (export "forward__read_args"              (func $read_args))
    (export "forward__return_data"            (func $return_data))
    (export "forward__account_load_bytes32"   (func $account_load_bytes32))
    (export "forward__account_store_bytes32"  (func $account_store_bytes32))
    (export "forward__call_contract"          (func $call_contract))
    (export "forward__delegate_call_contract" (func $delegate_call))
    (export "forward__static_call_contract"   (func $static_call))
    (export "forward__create1"                (func $create1))
    (export "forward__create2"                (func $create2))
    (export "forward__read_return_data"       (func $read_return_data))
    (export "forward__return_data_size"       (func $return_data_size))
    (export "forward__emit_log"               (func $emit_log))
    (export "forward__tx_origin"              (func $tx_origin)))
