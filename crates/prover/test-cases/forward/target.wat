
(module
    (import "env" "wavm_caller_load8" (func $load (param i32) (result i32)))
    (import "env" "wavm_caller_store8" (func $store (param i32 i32)))
    (func (export "target__add") (param i32 i32) (result i32)
        call $write_caller
        local.get 0
        local.get 1
        i32.add)
    (func (export "target__sub") (param i32 i32) (result i32)
        call $write_caller
        local.get 0
        local.get 1
        i32.sub)
    (func (export "target__mul") (param i32 i32) (result i32)
        call $write_caller
        local.get 0
        local.get 1
        i32.mul)
    (func $write_caller (export "write_caller")
        ;; increment the value at address 0xa4b in the caller
        i32.const 0xa4b
        i32.const 0xa4b
        call $load
        i32.const 1
        i32.add
        call $store))
