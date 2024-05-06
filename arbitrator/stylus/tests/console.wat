;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "console" "log_txt" (func $log_txt (param i32 i32)))
    (import "console" "log_i32" (func $log_i32 (param i32)))
    (import "console" "log_i64" (func $log_i64 (param i64)))
    (import "console" "log_f32" (func $log_f32 (param f32)))
    (import "console" "log_f64" (func $log_f64 (param f64)))
    (import "console" "tee_i32" (func $tee_i32 (param i32) (result i32)))
    (import "console" "tee_i64" (func $tee_i64 (param i64) (result i64)))
    (import "console" "tee_f32" (func $tee_f32 (param f32) (result f32)))
    (import "console" "tee_f64" (func $tee_f64 (param f64) (result f64)))
    (memory (export "memory") 1 1)
    (data (i32.const 0xa4b) "\57\65\20\68\61\76\65\20\74\68\65\20\69\6E\6B\21") ;; We have the ink!
    (func $start
        (call $log_txt (i32.const 0xa4b) (i32.const 16))

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
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)
    )
    (start $start))
