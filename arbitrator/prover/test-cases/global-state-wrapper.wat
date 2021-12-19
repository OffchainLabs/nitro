;; By default, arbitrator doesn't allow host IO use directly from the main module.
;; This fixes that by wrapping the host IO in this library, which then reexports it with the same name.

(import "env" "wavm_set_globalstate_u64" (func $set (param i32) (param i64)))
(import "env" "wavm_get_globalstate_u64" (func $get (param i32) (result i64)))

(export "env__wavm_set_globalstate_u64" (func $set))
(export "env__wavm_get_globalstate_u64" (func $get))
