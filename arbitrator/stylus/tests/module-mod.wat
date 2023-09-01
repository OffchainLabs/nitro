;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (import "test" "noop" (func))
    (memory 0 0)
    (export "memory" (memory 0))
    (func (export "void"))
    (func (export "more") (param i32 i64) (result f32)
        unreachable))
