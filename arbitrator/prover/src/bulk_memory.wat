;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
;;
;; This file implements the bulk memory instructions as per the specification below
;; https://github.com/WebAssembly/bulk-memory-operations/blob/master/proposals/bulk-memory-operations/Overview.md

(module
    (memory 0)
    (func $memory_fill (param $dest i32) (param $value i32) (param $size i32)
        ;; the bounds check happens before any data is written according to the spec

        ;; get the last offset
        local.get $dest
        i64.extend_i32_u
        local.get $size
        i64.extend_i32_u
        i64.add

        ;; memory size in bytes
        memory.size
        i64.extend_i32_u
        i64.const 0x10000
        i64.mul

        ;; trap if out of bounds
        i64.gt_u
        (if (then unreachable))

        ;; fill the region
        (loop $loop
            ;; see if there's more data to set
            local.get $size
            i32.eqz
            (if (then return))

            ;; walk back from the end
            local.get $size
            i32.const 1
            i32.sub
            local.tee $size
            local.get $dest
            i32.add

            ;; write the value
            local.get $value
            i32.store8

            br $loop
        )
    )
    (func $memory_copy (param $dest i32) (param $source i32) (param $size i32)
        (local $offset i32)

        local.get $source
        local.get $dest
        i32.gt_s
        (if ;; copy forward when source > dest
            (then
                ;; trap if out of bounds
                (i64.gt_u
                    ;; get the last source offset
                    local.get $source
                    i64.extend_i32_u
                    local.get $size
                    i64.extend_i32_u
                    i64.add

                    ;; memory size in bytes
                    memory.size
                    i64.extend_i32_u
                    i64.const 0x10000
                    i64.mul)
                (if (then unreachable))

                ;; do the copy
                (loop $forward
                    ;; see if there's more data to set
                    local.get $offset
                    local.get $size
                    i32.eq
                    (if (then return))

                    ;; push target
                    local.get $offset
                    local.get $dest
                    i32.add

                    ;; load the value
                    local.get $offset
                    local.get $source
                    i32.add
                    i32.load8_u

                    ;; write the value
                    i32.store8

                    ;; increment offset
                    local.get $offset
                    i32.const 1
                    i32.add
                    local.set $offset

                    br $forward))

            ;; copy backward when source <= dest
            (else
                ;; trap if out of bounds
                (i64.gt_u
                    ;; get the last destination offset
                    local.get $dest
                    i64.extend_i32_u
                    local.get $size
                    i64.extend_i32_u
                    i64.add

                    ;; memory size in bytes
                    memory.size
                    i64.extend_i32_u
                    i64.const 0x10000
                    i64.mul
                    )
                (if (then unreachable))

                ;; do the copy
                (loop $backward
                    ;; see if there's more data to set
                    local.get $size
                    i32.eqz
                    (if (then return))

                    ;; walk backwards
                    local.get $size
                    i32.const 1
                    i32.sub
                    local.tee $size
                    local.get $dest
                    i32.add

                    ;; load the value
                    local.get $size
                    local.get $source
                    i32.add
                    i32.load8_u

                    ;; write the value
                    i32.store8

                    br $backward)))))
