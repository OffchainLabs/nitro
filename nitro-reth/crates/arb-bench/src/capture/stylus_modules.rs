//! Stylus WASM modules covering distinct hot paths.

use alloy_primitives::Address;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StylusModule {
    Noop,
    StorageChurn,
    MemoryGrow,
    ComputeLoop,
    LogEmit,
    HostFanout,
}

impl StylusModule {
    pub fn deploy_address(&self) -> Address {
        match self {
            Self::Noop => alloy_primitives::address!("000000000000000000000000000000005719cc01"),
            Self::StorageChurn => {
                alloy_primitives::address!("000000000000000000000000000000005719cc02")
            }
            Self::MemoryGrow => {
                alloy_primitives::address!("000000000000000000000000000000005719cc03")
            }
            Self::ComputeLoop => {
                alloy_primitives::address!("000000000000000000000000000000005719cc04")
            }
            Self::LogEmit => alloy_primitives::address!("000000000000000000000000000000005719cc05"),
            Self::HostFanout => {
                alloy_primitives::address!("000000000000000000000000000000005719cc06")
            }
        }
    }

    pub fn name(&self) -> &'static str {
        match self {
            Self::Noop => "noop",
            Self::StorageChurn => "storage_churn",
            Self::MemoryGrow => "memory_grow",
            Self::ComputeLoop => "compute_loop",
            Self::LogEmit => "log_emit",
            Self::HostFanout => "host_fanout",
        }
    }

    pub fn wat(&self) -> &'static str {
        match self {
            Self::Noop => NOOP_WAT,
            Self::StorageChurn => STORAGE_CHURN_WAT,
            Self::MemoryGrow => MEMORY_GROW_WAT,
            Self::ComputeLoop => COMPUTE_LOOP_WAT,
            Self::LogEmit => LOG_EMIT_WAT,
            Self::HostFanout => HOST_FANOUT_WAT,
        }
    }
}

const NOOP_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (memory (export "memory") 1 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (call $write_result (i32.const 0) (i32.const 0))
    (i32.const 0)))
"#;

/// 8 SLOAD+SSTORE cycles per call.
const STORAGE_CHURN_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (import "vm_hooks" "storage_load_bytes32" (func $sload (param i32 i32)))
  (import "vm_hooks" "storage_cache_bytes32" (func $sstore_cache (param i32 i32)))
  (import "vm_hooks" "storage_flush_cache" (func $sflush (param i32)))
  (memory (export "memory") 1 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (local $i i32)
    (local.set $i (i32.const 0))
    (block $exit
      (loop $loop
        (br_if $exit (i32.ge_u (local.get $i) (i32.const 8)))
        ;; key at offset 0..32 = i (last byte = i)
        (i32.store8 (i32.const 31) (local.get $i))
        ;; load existing value into 32..64
        (call $sload (i32.const 0) (i32.const 32))
        ;; flip bit 0 of byte 63
        (i32.store8
          (i32.const 63)
          (i32.xor (i32.load8_u (i32.const 63)) (i32.const 1)))
        ;; cache write at key (offset 0), value (offset 32)
        (call $sstore_cache (i32.const 0) (i32.const 32))
        (local.set $i (i32.add (local.get $i) (i32.const 1)))
        (br $loop)))
    ;; flush all cached writes (clear=0 so we don't reset stylus_caches)
    (call $sflush (i32.const 0))
    (call $write_result (i32.const 0) (i32.const 0))
    (i32.const 0)))
"#;

/// Grows + touches 4 memory pages per call.
const MEMORY_GROW_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (memory (export "memory") 1 32)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (local $base i32)
    (local $i i32)
    (memory.grow (i32.const 4))
    drop
    ;; touch each new page (4 pages = 4*65536 bytes) at 16-byte stride
    (local.set $i (i32.const 65536)) ;; start past initial page
    (block $exit
      (loop $loop
        (br_if $exit (i32.ge_u (local.get $i) (i32.const 327680)))
        (i32.store (local.get $i) (local.get $i))
        (local.set $i (i32.add (local.get $i) (i32.const 16)))
        (br $loop)))
    (call $write_result (i32.const 0) (i32.const 0))
    (i32.const 0)))
"#;

/// 4096-iteration arithmetic loop per call.
const COMPUTE_LOOP_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (memory (export "memory") 1 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (local $i i32)
    (local $acc i64)
    (local.set $acc (i64.const 1))
    (local.set $i (i32.const 0))
    (block $exit
      (loop $loop
        (br_if $exit (i32.ge_u (local.get $i) (i32.const 4096)))
        (local.set $acc
          (i64.add
            (i64.mul (local.get $acc) (i64.const 1103515245))
            (i64.const 12345)))
        (local.set $i (i32.add (local.get $i) (i32.const 1)))
        (br $loop)))
    (i64.store (i32.const 0) (local.get $acc))
    (call $write_result (i32.const 0) (i32.const 8))
    (i32.const 0)))
"#;

/// 4 log events per call.
const LOG_EMIT_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (import "vm_hooks" "emit_log" (func $emit_log (param i32 i32 i32)))
  (memory (export "memory") 1 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (local $i i32)
    ;; layout: [topic0:32][topic1:32][data:32]
    (i32.store (i32.const 0) (i32.const 0xdeadbeef))
    (i32.store (i32.const 32) (i32.const 0xc0debabe))
    (local.set $i (i32.const 0))
    (block $exit
      (loop $loop
        (br_if $exit (i32.ge_u (local.get $i) (i32.const 4)))
        (i32.store (i32.const 64) (local.get $i))
        ;; emit_log(data_ptr=0, len=96 = 2 topics + 32 data, topic_count=2)
        (call $emit_log (i32.const 0) (i32.const 96) (i32.const 2))
        (local.set $i (i32.add (local.get $i) (i32.const 1)))
        (br $loop)))
    (call $write_result (i32.const 0) (i32.const 0))
    (i32.const 0)))
"#;

/// Calls 7 host functions in a tight loop per call.
const HOST_FANOUT_WAT: &str = r#"
(module
  (import "vm_hooks" "pay_for_memory_grow" (func (param i32)))
  (import "vm_hooks" "read_args" (func $read_args (param i32)))
  (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
  (import "vm_hooks" "block_number" (func $block_number (param i32)))
  (import "vm_hooks" "block_timestamp" (func $block_timestamp (param i32)))
  (import "vm_hooks" "msg_sender" (func $msg_sender (param i32)))
  (import "vm_hooks" "msg_value" (func $msg_value (param i32)))
  (import "vm_hooks" "contract_address" (func $contract_address (param i32)))
  (import "vm_hooks" "evm_gas_left" (func $gas_left (result i64)))
  (import "vm_hooks" "evm_ink_left" (func $ink_left (result i64)))
  (memory (export "memory") 1 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (local $i i32)
    (local.set $i (i32.const 0))
    (block $exit
      (loop $loop
        (br_if $exit (i32.ge_u (local.get $i) (i32.const 8)))
        (call $block_number (i32.const 0))
        (call $block_timestamp (i32.const 32))
        (call $msg_sender (i32.const 64))
        (call $msg_value (i32.const 96))
        (call $contract_address (i32.const 128))
        (i64.store (i32.const 160) (call $gas_left))
        (i64.store (i32.const 168) (call $ink_left))
        (local.set $i (i32.add (local.get $i) (i32.const 1)))
        (br $loop)))
    (call $write_result (i32.const 0) (i32.const 0))
    (i32.const 0)))
"#;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn all_modules_compile() {
        for m in [
            StylusModule::Noop,
            StylusModule::StorageChurn,
            StylusModule::MemoryGrow,
            StylusModule::ComputeLoop,
            StylusModule::LogEmit,
            StylusModule::HostFanout,
        ] {
            wat::parse_bytes(m.wat().as_bytes())
                .unwrap_or_else(|e| panic!("{} fails to parse: {e}", m.name()));
        }
    }

    #[test]
    fn deploy_addresses_distinct() {
        let mut addrs = std::collections::HashSet::new();
        for m in [
            StylusModule::Noop,
            StylusModule::StorageChurn,
            StylusModule::MemoryGrow,
            StylusModule::ComputeLoop,
            StylusModule::LogEmit,
            StylusModule::HostFanout,
        ] {
            assert!(
                addrs.insert(m.deploy_address()),
                "duplicate addr for {}",
                m.name()
            );
        }
    }
}
