;; msg_sender(dest_ptr)
;;
;; Calldata layout : empty
;; Return data     : [0..20] msg.sender (raw 20-byte address)
;;
;; Stylus's msg_sender writes 20 bytes (not left-padded). To return a
;; 32-byte ABI-style word the fixture would need to memset the leading 12
;; bytes; here we keep the raw form so callers can compare against
;; `tx.from` / aliasing-derived addresses byte-for-byte.
;;
;; Variants for fixture authors (.json):
;;   * EOA -> WASM (sender == tx.from)
;;   * EVM contract -> WASM CALL    (sender == EVM caller)
;;   * EVM contract -> WASM DELEGATECALL (sender preserved across delegate)
;;   * L1 -> L2 retryable redeem (sender == aliased L1 address)
;;   * Stylus self-call via call_contract (sender == this contract)

(module
    (import "vm_hooks" "msg_sender"
        (func $msg_sender (param i32)))

    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; no input; ignore $args_len
        (call $read_args (i32.const 0))

        ;; write 20-byte sender at offset 0
        (call $msg_sender (i32.const 0))

        ;; return raw 20-byte address
        (call $write_result (i32.const 0) (i32.const 20))

        i32.const 0
    )
)
