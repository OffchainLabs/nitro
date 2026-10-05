;; transient_store_bytes32(key_ptr, value_ptr) — TSTORE per EIP-1153.
;;
;; Calldata layout : [0..32]  key
;;                   [32..64] value
;; Return data     : empty

(module
    (import "vm_hooks" "transient_store_bytes32"
        (func $transient_store_bytes32 (param i32 i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $transient_store_bytes32 (i32.const 0) (i32.const 32))
        (call $write_result (i32.const 0) (i32.const 0))
        i32.const 0
    )
)
