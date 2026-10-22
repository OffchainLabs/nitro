;; block_basefee(dest_ptr) — write the L2 basefee (uint256, 32-byte BE) at
;; dest_ptr. On Arbitrum this is the L2 base_fee_wei, distinct from the L1
;; price_per_unit.
;;
;; Calldata layout : empty
;; Return data     : [0..32]  block.basefee

(module
    (import "vm_hooks" "block_basefee"
        (func $block_basefee (param i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $block_basefee (i32.const 0))
        (call $write_result (i32.const 0) (i32.const 32))
        i32.const 0
    )
)
