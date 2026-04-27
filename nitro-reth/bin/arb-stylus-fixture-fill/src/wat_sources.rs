//! Inline WAT sources for fixture programs that don't have a dedicated file
//! under `_wat/`.

/// A small Stylus program that exposes a `multicall` style entrypoint:
/// the calldata's first byte selects the operation (0 = call, 1 = delegate,
/// 2 = static, 3 = create1, 4 = create2). The remainder is forwarded as the
/// call payload. The contract returns 4 status bytes (op + 3 zero pad).
///
/// Body intentionally minimal — fixture-runner asserts gas + side effects,
/// not return-data shape. This program is what backs every
/// `multicall` / `reentrant_multicall` deploy in the subcall fixtures.
pub fn multicall_wat() -> &'static str {
    r#"(module
    (import "vm_hooks" "read_args"     (func $read_args     (param i32)))
    (import "vm_hooks" "write_result"  (func $write_result  (param i32 i32)))
    (import "vm_hooks" "msg_sender"    (func $msg_sender    (param i32)))
    (import "vm_hooks" "msg_reentrant" (func $msg_reentrant (result i32)))
    (import "vm_hooks" "call_contract"
        (func $call_contract (param i32 i32 i32 i32 i64 i32) (result i32)))
    (import "vm_hooks" "delegate_call_contract"
        (func $delegate_call_contract (param i32 i32 i32 i64 i32) (result i32)))
    (import "vm_hooks" "static_call_contract"
        (func $static_call_contract (param i32 i32 i32 i64 i32) (result i32)))
    (import "vm_hooks" "create1"
        (func $create1 (param i32 i32 i32 i32 i32)))
    (import "vm_hooks" "create2"
        (func $create2 (param i32 i32 i32 i32 i32 i32)))

    (memory (export "memory") 2)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $write_result (i32.const 0) (i32.const 1))
        i32.const 0
    )
)"#
}

/// A factory program that takes calldata = `[op:u8, salt(0x20)?, value(0x20),
/// init_code(rest)]` and dispatches into create1 / create2. Returns 20 bytes
/// of the deployed address.
pub fn create_factory_wat() -> &'static str {
    r#"(module
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (import "vm_hooks" "create1"
        (func $create1 (param i32 i32 i32 i32 i32)))
    (import "vm_hooks" "create2"
        (func $create2 (param i32 i32 i32 i32 i32 i32)))

    (memory (export "memory") 2)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $write_result (i32.const 0) (i32.const 20))
        i32.const 0
    )
)"#
}

/// Trivial Stylus program that is unique per call site; the suffix is dropped
/// into a comment so the resulting WASM byte stream differs and produces a
/// distinct codehash.
pub fn distinct_keccak_program(suffix: &str) -> String {
    let suffix_clean: String = suffix
        .chars()
        .filter(|c| c.is_ascii_alphanumeric() || *c == '_')
        .collect();
    format!(
        r#"(module
    (import "vm_hooks" "native_keccak256"
        (func $keccak (param i32 i32 i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (data (i32.const 0x200) "{suffix_clean}")

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $keccak (i32.const 0) (local.get $args_len) (i32.const 0x100))
        (call $write_result (i32.const 0x100) (i32.const 32))
        i32.const 0
    )
)"#
    )
}

/// Build a `(data ...)` segment whose payload is `target_size` bytes of fill.
/// We attach this to a base WAT to make the compiled WASM exceed a threshold
/// without disturbing the program logic.
pub fn oversize_data_segment(target_size: usize) -> String {
    let payload: String = "\\01".repeat(target_size);
    format!("\n  (data (i32.const 0x4000) \"{payload}\")\n")
}
