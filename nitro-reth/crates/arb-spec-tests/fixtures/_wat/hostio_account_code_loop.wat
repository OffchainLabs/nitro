;; account_code(addr_ptr, offset, size, dest) read in a loop — mirrors a
;; program that reads a large data contract's code in fixed-size chunks.
;;
;; Calldata layout : [0..20] target account address
;; Reads 256 chunks of 128 bytes (offset i*128) from the target's code.

(module
    (import "vm_hooks" "account_code"
        (func $account_code (param i32 i32 i32 i32) (result i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (local $i i32)
        (call $read_args (i32.const 0))
        (local.set $i (i32.const 0))
        (block $done
            (loop $loop
                (br_if $done (i32.ge_u (local.get $i) (i32.const 256)))
                (drop (call $account_code
                    (i32.const 0)
                    (i32.mul (local.get $i) (i32.const 128))
                    (i32.const 128)
                    (i32.const 1024)))
                (local.set $i (i32.add (local.get $i) (i32.const 1)))
                (br $loop)))
        (call $write_result (i32.const 0) (i32.const 0))
        (i32.const 0)
    )
)
