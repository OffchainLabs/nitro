;; account_balance(addr_ptr, dest_ptr) — write the balance (32-byte BE) of
;; the account at addr_ptr to memory at dest_ptr.
;;
;; Calldata layout : [0..20]  account address
;; Return data     : [0..32]  balance (uint256)

(module
    (import "vm_hooks" "account_balance"
        (func $account_balance (param i32 i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $account_balance (i32.const 0) (i32.const 0x100))
        (call $write_result (i32.const 0x100) (i32.const 32))
        i32.const 0
    )
)
