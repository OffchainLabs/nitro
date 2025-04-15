;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (import "test" "noop" (func))
    (memory (export "memory") 0 0)
    (func (export "void"))
    (func (export "more") (param i32 i64) (result f32)
        unreachable))
