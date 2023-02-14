
(module
    (func $safe (param $args_len i32) (result i32)
        local.get $args_len)
    (func $unreachable (param $args_len i32) (result i32)
        i32.const 0
        i64.const 4
        unreachable)
    (func $out_of_bounds (param $args_len i32) (result i32)
        i32.const 0xFFFFFF
        i32.load)
    (memory 1))
