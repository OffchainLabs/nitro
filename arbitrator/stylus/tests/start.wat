;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (global $status (export "status") (mut i32) (i32.const 10))
    (memory 0 0)
    (export "memory" (memory 0))
    (type $void (func (param) (result)))
    (func $start (export "move_me") (type $void)
        get_global $status
        i32.const 1
        i32.add
        set_global $status ;; increment the global
    )
    (start $start))
