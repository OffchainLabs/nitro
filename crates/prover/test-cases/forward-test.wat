
(module
    (import "forward" "add" (func $add (param i32 i32) (result i32)))
    (import "forward" "sub" (func $sub (param i32 i32) (result i32)))
    (import "forward" "mul" (func $mul (param i32 i32) (result i32)))
    (func $start
        ;; this address will update each time a forwarded call is made
        i32.const 0xa4b
        i32.const 805
        i32.store

        i32.const 11
        i32.const 5
        call $sub

        i32.const 3
        i32.const -2
        call $mul

        call $add
        (if
            (then (unreachable)))

        ;; check that the address has changed
        i32.const 0xa4b
        i32.load
        i32.const 808
        i32.ne
        (if
            (then (unreachable))))
    (start $start)
    (memory 1))
