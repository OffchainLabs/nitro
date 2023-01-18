;; Copyright 2022-2023, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (global $global (mut i64) (i64.const 32))
    (func $start
        global.get $global
        i64.clz
        drop
        )
    (start $start))
