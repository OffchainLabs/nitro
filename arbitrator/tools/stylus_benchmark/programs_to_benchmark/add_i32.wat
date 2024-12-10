(module
    (import "debug" "toggle_benchmark" (func $toggle_benchmark))
    (memory (export "memory") 0 0)
    (global $i (mut i32) (i32.const 0))
    (func (export "user_entrypoint") (param i32) (result i32)
        call $toggle_benchmark

        (loop $loop
            global.get $i
            i32.const 1
            i32.add
            global.set $i

            global.get $i
            i32.const 10000000
            i32.lt_s
            br_if $loop)

        call $toggle_benchmark

        i32.const 0)
)
