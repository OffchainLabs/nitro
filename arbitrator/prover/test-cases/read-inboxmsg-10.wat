(import "wrapper" "set_globalstate_u64" (func $set (param i32) (param i64)))
(import "wrapper" "get_globalstate_u64" (func $get (param i32) (result i64)))
(import "wrapper" "read_inbox_message" (func $readinbox (param i64) (param i32) (param i32) (result i32)))
(import "wrapper" "halt_and_set_finished" (func $halt))

(memory 1)

(func $entry
    (i64.const 10)
    (i32.const 0)
    (i32.const 0)
    (call $readinbox)
    (drop)
    ;; halt
    (call $halt)
)

(start $entry)