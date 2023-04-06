;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "console" "log_i32" (func $log_i32 (param i32)))
    (import "console" "log_i64" (func $log_i64 (param i64)))
    (import "console" "log_f32" (func $log_f32 (param f32)))
    (import "console" "log_f64" (func $log_f64 (param f64)))
    (import "console" "tee_i32" (func $tee_i32 (param i32) (result i32)))
    (import "console" "tee_i64" (func $tee_i64 (param i64) (result i64)))
    (import "console" "tee_f32" (func $tee_f32 (param f32) (result f32)))
    (import "console" "tee_f64" (func $tee_f64 (param f64) (result f64)))
    (memory (export "memory") 0 0)
    (func $start
        i32.const 48
        call $tee_i32
        call $log_i32

        i64.const 96
        call $tee_i64
        call $log_i64

        f32.const 0.32
        call $tee_f32
        call $log_f32

        f64.const -64.32
        call $tee_f64
        call $log_f64)
    (start $start))
