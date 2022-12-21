;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (global $depth (export "depth") (mut i32) (i32.const 0))
    (func $recurse (export "recurse")
        global.get $depth   ;; push 1        -- 1 on stack
        i32.const 1         ;; push 1        -- 2 on stack  <- 2 words max
        i32.add             ;; pop 2, push 1 -- 1 on stack
        global.set $depth   ;; pop 1         -- 0 on stack
        call $recurse))
