;; Copyright 2026, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
;;
;; Fixed version of the memory.fill bulk memory instruction.
;; memory_copy is unchanged from bulk_memory.wat and loaded from there directly.
;;
;; Fix: the value argument is masked to 8 bits before building the 64-bit fill pattern,
;; matching the WebAssembly spec which only uses the low 8 bits of the value.

(module
    (memory (export "memory") 0 0)
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
        ;; mask to 8 bits first so upper bits of $value don't leak into the pattern
        (local.tee $value64
            (i64.extend_i32_u (i32.and (local.get $value) (i32.const 0xff))))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        (i64.shl (i64.const 8))
        (i64.or (local.get $value64))
        local.set $value64

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

            ;; i32.store8 naturally truncates to the low 8 bits
            local.get $value
            i32.store8
            br $loop
        )
    )
)
