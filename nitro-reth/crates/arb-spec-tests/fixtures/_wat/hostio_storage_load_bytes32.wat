;; storage_load_bytes32(key_ptr, dest_ptr)
;;
;; Calldata layout : [0..32]  storage slot key (B256)
;; Return data     : [0..32]  storage word at that slot
;;
;; Variants for fixture authors (.json):
;;   * cold load of an unset slot (returns zeros)
;;   * cold load of a pre-populated slot
;;   * warm reload of the same slot in a single tx
;;   * load against a slot written by a sibling EVM contract via DELEGATECALL
;;
;; The hostio internally pays COLD_SLOAD_GAS + STORAGE_CACHE_REQUIRED_ACCESS_GAS
;; + an evm_api_gas charge that is ArbOS-version gated; expectations should be
;; captured per ArbOS version of interest.

(module
    (import "vm_hooks" "storage_load_bytes32"
        (func $storage_load_bytes32 (param i32 i32)))

    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; key is the entire 32-byte calldata payload at offset 0
        (call $read_args (i32.const 0))

        ;; load: key at 0, write the loaded value to offset 32
        (call $storage_load_bytes32 (i32.const 0) (i32.const 32))

        ;; return the loaded value
        (call $write_result (i32.const 32) (i32.const 32))

        i32.const 0
    )
)
