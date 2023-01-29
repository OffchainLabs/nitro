
(module
    (import "hostio" "link_module"        (func $link       (param i32) (result i32)    ))
    (import "hostio" "unlink_module"      (func $unlink                                 ))
    (import "hostio" "program_set_gas"    (func $set_gas    (param i32 i32 i64)         ))
    (import "hostio" "program_gas_left"   (func $gas_left   (param i32 i32) (result i64)))
    (import "hostio" "program_gas_status" (func $gas_status (param i32 i32) (result i32)))
    (data (i32.const 0x0)
        "\84\10\70\b5\13\fa\91\d3\44\84\24\c9\b1\79\ac\7a\2b\09\56\4d\d1\e6\6d\87\cc\82\85\4c\02\f1\f5\12") ;; globals
    (func $start (local $user i32) (local $internals i32)
        ;; link in globals.wat
        i32.const 0
        call $link
        local.set $user

        ;; set internals offset
        i32.const 1
        local.set $internals

        ;; set gas globals
        local.get $user
        local.get $internals
        i64.const 1024
        call $set_gas

        ;; get gas
        local.get $user
        local.get $internals
        call $gas_left
        i64.const 1024
        i64.ne
        (if
            (then (unreachable)))

        ;; get gas status
        local.get $user
        local.get $internals
        call $gas_status
        i32.const 0
        i32.ne
        (if
            (then (unreachable)))

        ;; unlink module
        call $unlink)
    (start $start)
    (memory 1))
