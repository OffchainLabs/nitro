;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
;;
;; This file implements the bulk memory instructions as per the specification below
;; https://github.com/WebAssembly/bulk-memory-operations/blob/master/proposals/bulk-memory-operations/Overview.md

(module
    (memory 0)
    (func $memory_fill (param $dest i32) (param $value i32) (param $size i32)
        (local $value64 i64)
        ;; the bounds check happens before any data is written according to the spec

        ;; get the last offset
        (i64.add
            (i64.extend_i32_u (local.get $dest))
            (i64.extend_i32_u (local.get $size)))

        ;; memory size in bytes
        (i64.mul
            (i64.extend_i32_u (memory.size))
            (i64.const 0x10000))

        ;; trap if out of bounds
        i64.gt_u
        (if (then unreachable))

        ;; create an 8-byte value for chunked filling
        (local.tee $value64 (i64.extend_i32_u (local.get $value)))
        (local.set $value64
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64))
            (i64.shl (i64.const 8)) (i64.add (local.get $value64)))

        ;; fill the region 8-bytes at a time
        (block $done
            (loop $loop
                ;; see if there's more data to set
                (i32.lt_u (local.get $size) (i32.const 8))
                br_if $done

                ;; walk back from the end
                (i32.sub (local.get $size) (i32.const 8))
                local.tee $size
                local.get $dest
                i32.add

                ;; write the value
                local.get $value64
                i64.store
                br $loop
            )
        )

        ;; fill the rest of the region
        (loop $loop
            ;; see if there's more data to set
            local.get $size
            i32.eqz
            (if (then return))

            ;; walk back from the end
            (i32.sub (local.get $size) (i32.const 1))
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

        (i32.gt_u (local.get $source) (local.get $dest))

        (if ;; copy forward when source > dest
            (then
                ;; trap if out of bounds
                (i64.gt_u
                    ;; get the source boundary
                    (i64.add
                        (i64.extend_i32_u (local.get $source))
                        (i64.extend_i32_u (local.get $size)))

                    ;; memory size in bytes
                    (i64.mul
                        (i64.extend_i32_u (memory.size))
                        (i64.const 0x10000)))
                (if (then unreachable))

                ;; copy the region 8-bytes at a time
                (block $done
                    (loop $forward
                        ;; see if there's more data to set
                        (i32.add (local.get $offset) (i32.const 8))
                        local.get $size
                        i32.gt_u
                        br_if $done

                        ;; push target
                        local.get $offset
                        local.get $dest
                        i32.add

                        ;; load the value
                        local.get $offset
                        local.get $source
                        i32.add
                        i64.load

                        ;; write the value
                        i64.store

                        ;; increment offset
                        (i32.add (local.get $offset) (i32.const 8))
                        local.set $offset
                        br $forward
                    )
                )

                ;; copy the rest of the region
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
                    (i32.add (local.get $offset) (i32.const 1))
                    local.set $offset
                    br $forward
                )
            )

            ;; copy backward when source <= dest
            (else
                ;; trap if out of bounds
                (i64.gt_u
                    ;; get the destination boundary
                    (i64.add
                        (i64.extend_i32_u (local.get $dest))
                        (i64.extend_i32_u (local.get $size)))

                    ;; memory size in bytes
                    (i64.mul
                        (i64.extend_i32_u (memory.size))
                        (i64.const 0x10000)))
                (if (then unreachable))

                ;; copy the region 8 bytes at a time
                (block $done
                    (loop $backward
                        ;; see if there's more data to set
                        (i32.lt_u (local.get $size) (i32.const 8))
                        br_if $done

                        ;; walk backwards
                        local.get $size
                        i32.const 8
                        i32.sub
                        local.tee $size
                        local.get $dest
                        i32.add

                        ;; load the value
                        local.get $size
                        local.get $source
                        i32.add
                        i64.load

                        ;; write the value
                        i64.store
                        br $backward
                    )
                )

                ;; copy the rest of the region
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
                    br $backward
                )
            )
        )
    )
)
