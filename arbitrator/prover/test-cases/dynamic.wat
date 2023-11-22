
(module
    (import "hostio" "wavm_link_module"      (func $link       (param i32)     (result i32)))
    (import "hostio" "wavm_unlink_module"    (func $unlink                                 ))
    (import "hostio" "wavm_set_error_policy" (func $set_policy (param i32)                 ))
    (import "hostio" "program_set_ink"       (func $set_ink    (param i32 i64)             ))
    (import "hostio" "program_ink_left"      (func $ink_left   (param i32)     (result i64)))
    (import "hostio" "program_ink_status"    (func $ink_status (param i32)     (result i32)))
    (import "hostio" "program_call_main"     (func $user_func  (param i32 i32) (result i32)))

    ;; WAVM Module hash
    (data (i32.const 0x0)
         "\97\0c\df\6a\a9\bf\d4\3c\03\80\7f\8a\7e\67\9a\5c\12\05\94\4f\c6\5e\39\9e\00\df\5c\b3\7d\de\55\ad") ;; user

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

        (;; enable error recovery
        i32.const 1
        call $set_policy

        ;; recover from an unreachable
        local.get $user
        i32.const 2 ;; $unreachable
        call $user_func
        i32.const 1 ;; indicates failure
        i32.ne
        (if
            (then (unreachable));)

        ;; push some items to the stack
        i32.const 0xa4b0
        i64.const 0xa4b1
        i32.const 0xa4b2

        (;; recover from an out-of-bounds memory access
        local.get $user
        i32.const 3 ;; $out_of_bounds
        call $user_func
        i32.const 1 ;; indicates failure
        i32.ne
        (if
            (then (unreachable));)

        ;; drop the items from the stack
        drop
        drop
        drop

        ;; unlink module
        call $unlink)
    (start $start)
    (memory 1))
