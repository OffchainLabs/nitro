;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

(module
    (import "env" "wavm_halt_and_set_finished" (func $halt))
    (func $start (export "start") (local $i i32)
        ;; test memory_fill
        (memory.fill (i32.const 0x1003) (i32.const 5) (i32.const 4)) ;; ---5555---
        (memory.fill (i32.const 0x1001) (i32.const 8) (i32.const 3)) ;; -888555---
        (memory.fill (i32.const 0x1005) (i32.const 2) (i32.const 1)) ;; -888525---
        (memory.fill (i32.const 0x1001) (i32.const 9) (i32.const 0)) ;; -888525---
        (memory.fill (i32.const 0x1000) (i32.const 0) (i32.const 0)) ;; -888525---
        (call $check (i32.const 0x1000) (i32.const 0))
        (call $check (i32.const 0x1001) (i32.const 8))
        (call $check (i32.const 0x1002) (i32.const 8))
        (call $check (i32.const 0x1003) (i32.const 8))
        (call $check (i32.const 0x1004) (i32.const 5))
        (call $check (i32.const 0x1005) (i32.const 2))
        (call $check (i32.const 0x1006) (i32.const 5))
        (call $check (i32.const 0x1007) (i32.const 0))

        ;; test memory_copy
        (memory.copy (i32.const 0x1008) (i32.const 0x1000) (i32.const 8))  ;; -888525--888525-----------------
        (memory.copy (i32.const 0x1009) (i32.const 0x1004) (i32.const 4))  ;; -888525--525-25-----------------
        (memory.copy (i32.const 0x1009) (i32.const 0x1009) (i32.const 0))  ;; -888525--525-25-----------------
        (memory.copy (i32.const 0x1009) (i32.const 0x1009) (i32.const 1))  ;; -888525--525-25-----------------
        (memory.copy (i32.const 0x1009) (i32.const 0x100a) (i32.const 1))  ;; -888525--225-25-----------------
        (memory.copy (i32.const 0x100f) (i32.const 0x1001) (i32.const 1))  ;; -888525--225-258----------------
        (memory.copy (i32.const 0x100f) (i32.const 0x1000) (i32.const 32)) ;; ----------------888525--225-258-
        (memory.copy (i32.const 0x1001) (i32.const 0x100f) (i32.const 32)) ;; --888525--225-258---------------
        (memory.copy (i32.const 0x100f) (i32.const 0x1000) (i32.const 35)) ;; ----------------888525--225-258-
        (memory.copy (i32.const 0x1000) (i32.const 0x100f) (i32.const 35)) ;; --888525--225-258---------------
        (memory.copy (i32.const 0x100f) (i32.const 0x1002) (i32.const 39)) ;; ----------------888525--225-258-
        (memory.copy (i32.const 0x1002) (i32.const 0x100f) (i32.const 39)) ;; --888525--225-258---------------

        (call $check (i32.const 0x1009) (i32.const 0))
        (call $check (i32.const 0x100a) (i32.const 2))
        (call $check (i32.const 0x100b) (i32.const 2))
        (call $check (i32.const 0x100c) (i32.const 5))
        (call $check (i32.const 0x100d) (i32.const 0))
        (call $check (i32.const 0x100e) (i32.const 2))
        (call $check (i32.const 0x100f) (i32.const 5))
        (call $check (i32.const 0x1010) (i32.const 8))

        ;; check that these don't overflow (memory is 1 page = 2^16 bytes)
        (memory.fill (i32.const 0xffff) (i32.const 4) (i32.const 1))
        (memory.fill (i32.const 0xfffe) (i32.const 4) (i32.const 2))
        (memory.copy (i32.const 0xffff) (i32.const 0xffff) (i32.const 1))
        (memory.copy (i32.const 0xfffd) (i32.const 0xfffc) (i32.const 3))
        (memory.copy (i32.const 0xfff2) (i32.const 0xfff3) (i32.const 13))
        (memory.copy (i32.const 0xfff3) (i32.const 0xfff2) (i32.const 13))

        ;; write [1, 16]
        (local.set $i (i32.const 16))
        (loop $loop
            (memory.fill (i32.const 0x0) (local.get $i) (local.get $i))

            (i32.sub (local.get $i) (i32.const 1))
            local.tee $i
            i32.const 0
            i32.ne
            br_if $loop)

        ;; check [1, 16]
        (local.set $i (i32.const 16))
        (loop $loop
            (call $check (i32.sub (local.get $i) (i32.const 1)) (local.get $i))

            (i32.sub (local.get $i) (i32.const 1))
            local.tee $i
            i32.const 0
            i32.ne
            br_if $loop)

        (call $halt))
    (func $check (param i32 i32)
        local.get 0
        i32.load8_u
        local.get 1
        i32.ne
        (if (then (unreachable))))
    
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)
    )

    (start $start)
    (memory (export "memory") 1 1))
