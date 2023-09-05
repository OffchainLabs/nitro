;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (import "test" "noop" (func))
    (memory 0 0)
    (export "memory" (memory 0))
    (global $depth (export "depth") (mut i32) (i32.const 0))
    (func $recurse (export "recurse") (param $ignored i64) (local f32 f64)
        local.get $ignored  ;; push 1        -- 1 on stack
        global.get $depth   ;; push 1        -- 2 on stack
        i32.const 1         ;; push 1        -- 3 on stack  <- 3 words max
        i32.add             ;; pop 2, push 1 -- 2 on stack
        global.set $depth   ;; pop 1         -- 1 on stack
        call $recurse)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)
    ))
