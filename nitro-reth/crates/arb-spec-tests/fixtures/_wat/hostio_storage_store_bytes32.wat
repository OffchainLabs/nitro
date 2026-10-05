;; "Storage store" in Stylus is a two-hostio sequence:
;;     storage_cache_bytes32(key_ptr, value_ptr)   -- stage the write
;;     storage_flush_cache(clear)                  -- commit to state
;;
;; Calldata layout : [ 0..32]  storage slot key   (B256)
;;                   [32..64]  storage slot value (B256)
;; Return data     : empty
;;
;; Variants for fixture authors (.json):
;;   * zero -> non-zero (SET)
;;   * non-zero -> non-zero (RESET)
;;   * non-zero -> zero (CLEAR -> refund eligible)
;;   * cache without flush (no state change committed)
;;   * cache + flush(clear=true) — exercises the gas-failure branch in
;;     `storage_flush_cache` against tight gas budgets
;;
;; Stylus charges SSTORE_SENTRY_GAS + STORAGE_CACHE_REQUIRED_ACCESS_GAS up front
;; and the actual SSTORE cost only at flush time, so per-variant gas captures
;; should split cache and flush.

(module
    (import "vm_hooks" "storage_cache_bytes32"
        (func $storage_cache_bytes32 (param i32 i32)))
    (import "vm_hooks" "storage_flush_cache"
        (func $storage_flush_cache  (param i32)))

    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; load (key || value) at offset 0
        (call $read_args (i32.const 0))

        ;; stage the write: key at 0, value at 32
        (call $storage_cache_bytes32 (i32.const 0) (i32.const 32))

        ;; commit to state; clear=0 keeps the cache around if reused
        (call $storage_flush_cache (i32.const 0))

        ;; no return payload
        (call $write_result (i32.const 0) (i32.const 0))

        i32.const 0
    )
)
