;; tx_origin(dest_ptr) — write 20-byte tx.origin at dest_ptr.
;;
;; Calldata layout : empty
;; Return data     : [0..20]  tx.origin

(module
    (import "vm_hooks" "tx_origin"
        (func $tx_origin (param i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $tx_origin (i32.const 0))
        (call $write_result (i32.const 0) (i32.const 20))
        i32.const 0
    )
)
