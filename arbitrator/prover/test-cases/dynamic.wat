
(module
    (import "hostio" "wavm_link_module"   (func $link       (param i32)     (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink                                 ))
    (import "hostio" "program_set_ink"    (func $set_ink    (param i32 i64)             ))
    (import "hostio" "program_ink_left"   (func $ink_left   (param i32)     (result i64)))
    (import "hostio" "program_ink_status" (func $ink_status (param i32)     (result i32)))
    (import "hostio" "program_call_main"  (func $user_func  (param i32 i32) (result i32)))
    (data (i32.const 0x0)
         "\80\d3\d5\e6\1a\9a\9d\58\9a\e8\42\d5\69\2f\c2\38\16\47\44\b1\5b\66\c5\d6\dc\8f\f5\b3\66\91\4a\ee")
    (func $start (local $user i32) (local $internals i32)
        ;; link in user.wat
        i32.const 0
        call $link
        local.set $user

        ;; set gas globals
        local.get $user
        i64.const 1024
        call $set_ink

        ;; get gas
        local.get $user
        call $ink_left
        i64.const 1024
        i64.ne
        (if
            (then (unreachable)))

        ;; get gas status
        local.get $user
        call $ink_status
        i32.const 0
        i32.ne
        (if
            (then (unreachable)))

        ;; call a successful func in user.wat ($safe)
        local.get $user
        i32.const 1 ;; $safe
        call $user_func
        i32.const 5
        i32.ne
        (if
            (then (unreachable)))

        ;; recover from an unreachable
        local.get $user
        i32.const 2 ;; $unreachable
        call $user_func
        i32.const 1 ;; indicates failure
        i32.ne
        (if
            (then (unreachable)))

        ;; push some items to the stack
        i32.const 0xa4b0
        i64.const 0xa4b1
        i32.const 0xa4b2

        ;; recover from an out-of-bounds memory access
        local.get $user
        i32.const 3 ;; $out_of_bounds
        call $user_func
        i32.const 1 ;; indicates failure
        i32.ne
        (if
            (then (unreachable)))

        ;; drop the items from the stack
        drop
        drop
        drop

        ;; unlink module
        call $unlink)
    (start $start)
    (memory 1))
