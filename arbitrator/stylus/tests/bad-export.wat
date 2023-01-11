;; Copyright 2022, Offchain Labs, Inc.
;; For license information, see https://github.com/nitro/blob/master/LICENSE

(module
    (global $status (export "stylus_gas_left") (mut i64) (i64.const -1)))
