;; block_number() -> i64 — Arbitrum returns the recorded L1 block number
;; (NOT the L2 block number). Per-message-different in real chain traffic.
;;
;; Calldata layout : empty
;; Return data     : [0..8]  block.number (u64 BE)

(module
    (import "vm_hooks" "block_number"
        (func $block_number (result i64)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (i64.store (i32.const 0) (call $block_number))
        (call $write_result (i32.const 0) (i32.const 8))
        i32.const 0
    )
)
