(module
    (memory (export "memory") 128 128)
    (func $start
        (local $counter i32)
        (local $scratch i32)

        (loop $loop
            local.get $scratch
            local.get $scratch
            i32.add
            drop

            local.get $counter
            i32.const 1
            i32.sub
            local.tee
            i32.const 0
            i32.ne
            br_if $loop)
    )
    (start $start)
)
