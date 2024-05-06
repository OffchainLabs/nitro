;; By default, arbitrator doesn't allow host IO use directly from the main module.
;; This fixes that by wrapping the host IO in this library, which then reexports it with the same name.

(import "env" "wavm_set_globalstate_u64" (func $set (param i32) (param i64)))
(import "env" "wavm_get_globalstate_u64" (func $get (param i32) (result i64)))
(import "env" "wavm_read_inbox_message" (func $readinbox (param i64) (param i32) (param i32) (result i32)))
(import "env" "wavm_halt_and_set_finished" (func $halt))

(export "wrapper__set_globalstate_u64" (func $set))
(export "wrapper__get_globalstate_u64" (func $get))
(export "wrapper__read_inbox_message" (func $readinbox))
(export "wrapper__halt_and_set_finished" (func $halt))
