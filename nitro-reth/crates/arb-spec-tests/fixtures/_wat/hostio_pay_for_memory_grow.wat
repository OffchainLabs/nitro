;; pay_for_memory_grow(pages: i32) — charge for growing the WASM linear
;; memory by N pages. We don't actually grow memory; this exercises the
;; per-version pricing of the hostio.
;;
;; Calldata layout : [0..4]  number of pages (BE u32, but Stylus passes the
;;                            low 16 bits; we just pass the i32 verbatim)
;; Return data     : empty

(module
    (import "vm_hooks" "pay_for_memory_grow"
        (func $pay_for_memory_grow (param i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (local $pages i32)
        (call $read_args (i32.const 0))
        ;; pages = u32 BE at offset 0..4
        (i32.load (i32.const 0))
        local.set $pages
        (call $pay_for_memory_grow (local.get $pages))
        (call $write_result (i32.const 0) (i32.const 0))
        i32.const 0
    )
)
