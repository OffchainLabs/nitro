
(module
    (import "hostio" "link_module"        (func $link       (param i32)         (result i32)))
    (import "hostio" "unlink_module"      (func $unlink                                     ))
    (import "hostio" "program_set_gas"    (func $set_gas    (param i32 i32 i64)             ))
    (import "hostio" "program_ink_left"   (func $ink_left   (param i32 i32)     (result i64)))
    (import "hostio" "program_ink_status" (func $ink_status (param i32 i32)     (result i32)))
    (import "hostio" "program_call_main"  (func $user_func  (param i32 i32 i32) (result i32)))
    (data (i32.const 0x0)
        "\8a\df\60\ca\14\1a\80\2a\f2\50\bc\fd\db\10\89\cc\fc\8c\e0\ae\45\59\cc\e7\1e\14\e9\75\10\ab\6b\ed") ;; user
    (func $start (local $user i32) (local $internals i32)
        ;; link in user.wat
        i32.const 0
        call $link
        local.set $user

        ;; set internals offset
        i32.const 3
        local.set $internals

        ;; set gas globals
        local.get $user
        local.get $internals
        i64.const 1024
        call $set_gas

        ;; get gas
        local.get $user
        local.get $internals
        call $ink_left
        i64.const 1024
        i64.ne
        (if
            (then (unreachable)))

        ;; get gas status
        local.get $user
        local.get $internals
        call $ink_status
        i32.const 0
        i32.ne
        (if
            (then (unreachable)))

        ;; call a successful func in user.wat ($safe)
        local.get $user
        i32.const 0 ;; $safe
        i32.const 64
        call $user_func
        i32.const 64
        i32.ne
        (if
            (then (unreachable)))

        ;; recover from an unreachable
        local.get $user
        i32.const 1 ;; $unreachable
        i32.const 0
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
        i32.const 2 ;; $out_of_bounds
        i32.const 0
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
