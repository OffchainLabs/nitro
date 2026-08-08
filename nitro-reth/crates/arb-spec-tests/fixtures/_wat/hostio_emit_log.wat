;; emit_log(data_ptr, data_len, num_topics)
;;
;; Stylus packs topics inline at the head of the data buffer:
;;     bytes  0 .. 32*N    : N topic words (N = num_topics, 0..=4)
;;     bytes 32*N .. data_len : payload bytes
;; The hostio rejects calls where `data_len < num_topics * 32` or `topics > 4`.
;;
;; Calldata layout : [    0]    num_topics u8 (one byte; 0..=4)
;;                   [    1..]  topics ++ payload (each topic is 32 bytes)
;; Return data     : empty
;;
;; Variants for fixture authors (.json):
;;   * 0 topics, 0-byte data
;;   * 0 topics, non-empty data
;;   * 1..=4 topics, varying payload sizes
;;   * num_topics=5 (must revert: "bad topic data")
;;   * data_len shorter than topics*32 (must revert)
;;
;; Pricing draws on EVM_LOG_TOPIC_GAS + EVM_LOG_DATA_GAS per byte; fixture
;; captures should split per topic count + payload length so regressions in
;; pay_for_evm_log are caught.

(module
    (import "vm_hooks" "emit_log"
        (func $emit_log (param i32 i32 i32)))

    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (local $topics i32)
        (local $payload_len i32)

        ;; load full calldata at offset 0; first byte is num_topics
        (call $read_args (i32.const 0))

        ;; topics = u8 at offset 0
        (i32.load8_u (i32.const 0))
        local.set $topics

        ;; payload_len = $args_len - 1 (everything after the leading byte)
        (i32.sub (local.get $args_len) (i32.const 1))
        local.set $payload_len

        ;; emit_log expects (data_ptr, data_len, topics).
        ;; data buffer starts at offset 1 (skipping the num_topics byte) and is
        ;; topics*32 bytes of topic data immediately followed by the payload.
        (call $emit_log
            (i32.const 1)
            (local.get $payload_len)
            (local.get $topics))

        ;; no return payload
        (call $write_result (i32.const 0) (i32.const 0))

        i32.const 0
    )
)
