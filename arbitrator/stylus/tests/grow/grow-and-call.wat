;; Copyright 2023-2024, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
    (import "vm_hooks" "read_args"           (func $read_args     (param i32)))
    (import "vm_hooks" "write_result"        (func $write_result  (param i32 i32)))
    (import "vm_hooks" "msg_value"           (func $msg_value     (param i32)))
    (import "vm_hooks" "call_contract"       (func $call_contract (param i32 i32 i32 i32 i64 i32) (result i32)))
    (import "console" "tee_i32"              (func $tee           (param i32) (result i32)))
    (func (export "user_entrypoint") (param $args_len i32) (result i32)

        ;; store the target size argument at offset 0
        i32.const 0
        call $read_args

        ;; grow the extra pages
        i32.const 0
        i32.load8_u
        memory.grow
        drop

        ;; copy the message value
        i32.const 0x1000
        call $msg_value

        ;; static call target contract
        i32.const 1                                    ;; address
        i32.const 21                                   ;; calldata
        (i32.sub (local.get $args_len) (i32.const 21)) ;; calldata len
        i32.const 0x1000                               ;; callvalue
        i64.const -1                                   ;; all gas
        i32.const 0x2000                               ;; return_data_len ptr
        call $call_contract
    )
    (memory (export "memory") 4)
)
