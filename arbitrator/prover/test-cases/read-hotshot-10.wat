(import "env" "wavm_set_globalstate_u64" (func $set (param i32) (param i64)))
(import "env" "wavm_get_globalstate_u64" (func $get (param i32) (result i64)))
(import "env" "wavm_read_hotshot_commitment" (func $readhotshot (param i32) (param i64)))
(import "env" "wavm_halt_and_set_finished" (func $halt))

(memory 1)

(func $entry
    (i32.const 0)
    (i64.const 10)
    (call $readhotshot)
    ;; halt
    (call $halt)
)

(start $entry)