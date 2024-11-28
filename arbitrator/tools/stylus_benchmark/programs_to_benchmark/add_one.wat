(module
    (import "debug" "toggle_benchmark" (func $toggle_benchmark))
    (memory (export "memory") 0 0)
    (func (export "user_entrypoint") (param i32) (result i32)
        call $toggle_benchmark

        i32.const 0
        i32.const 2
        i32.add
        drop

        call $toggle_benchmark

        i32.const 0)
)
