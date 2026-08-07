;; msg_value(dest_ptr) — write 32-byte BE msg.value at dest_ptr.
;;
;; Calldata layout : empty
;; Return data     : [0..32]  msg.value (uint256)

(module
    (import "vm_hooks" "msg_value"
        (func $msg_value (param i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $msg_value (i32.const 0))
        (call $write_result (i32.const 0) (i32.const 32))
        i32.const 0
    )
)
