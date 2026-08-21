// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! End-to-end tests for `Module::to_wavm_bytes` / `from_wavm_bytes`.
//!
//! These tests construct `Module`s directly (taking advantage of `pub(crate)`
//! field access from inside `machine.rs`) so they exercise the full round-trip
//! and merkle reconstruction without requiring a full Stylus toolchain.

use std::sync::Arc;

use arbutil::{Bytes32, crypto};
use brotli::Dictionary;
use eyre::Result;
use fnv::FnvHashMap as HashMap;
use wasmparser::{RefType, TableType};

use crate::{
    binary::{ExportKind, ExportMap, NameCustomSection},
    machine::{
        Function, GlobalState, Machine, Module, Table, TableElement,
        format_missing_stylus_module_error,
    },
    memory::Memory,
    merkle::{Merkle, MerkleType},
    value::{ArbValueType, FunctionType, Value},
    wavm::{Instruction, Opcode},
    wavm_serialize::{Cursor, WAVM_MAGIC, WAVM_SERIALIZE_VERSION},
};

/// Build a small but non-trivial `Module` covering the consensus-relevant
/// fields. The exact contents don't have to be executable — what matters is
/// that `Module::hash()` inputs (globals, memory, `tables_merkle.root()`,
/// `funcs_merkle.root()`, `extra_hash`, `internals_offset`) and the
/// non-hashed but round-tripped fields are populated.
fn build_test_module() -> Module {
    let mut memory = Memory::new(64, 1);
    memory.set_range(0, &[1, 2, 3, 4, 5, 6, 7, 8]).unwrap();
    memory.cache_merkle_tree();

    let function_type = FunctionType {
        inputs: vec![ArbValueType::I32, ArbValueType::I64],
        outputs: vec![ArbValueType::I32],
    };

    let func = Function::new_from_wavm(
        vec![
            Instruction {
                opcode: Opcode::InitFrame,
                argument_data: 0,
                proving_argument_data: Some(Bytes32([0u8; 32])),
            },
            Instruction::with_data(Opcode::I32Const, 42),
            Instruction::simple(Opcode::Return),
        ],
        function_type.clone(),
        vec![ArbValueType::I32, ArbValueType::I64],
    );

    // Match the activator path (`Module::from_binary`): `elems_merkle` stays
    // empty so `Table::hash` sees `Merkle::default().root()` (all zeros), which
    // is what gets committed in `tables_merkle` on activation.
    let table = Table {
        ty: TableType {
            element_type: RefType::FUNCREF,
            initial: 2,
            maximum: Some(8),
            table64: false,
            shared: false,
        },
        elems: vec![
            TableElement {
                func_ty: function_type.clone(),
                val: Value::FuncRef(0),
            },
            TableElement::default(),
        ],
        elems_merkle: Merkle::default(),
    };
    let tables = vec![table];
    let tables_hashes: Result<_> = tables.iter().map(Table::hash).collect();
    let tables_merkle = Merkle::new(MerkleType::Table, tables_hashes.unwrap());

    let funcs = Arc::new(vec![func]);
    let funcs_merkle = Arc::new(Merkle::new(
        MerkleType::Function,
        funcs.iter().map(Function::hash).collect(),
    ));

    let mut func_exports: HashMap<String, u32> = HashMap::default();
    func_exports.insert("main".to_owned(), 0);

    let mut all_exports = ExportMap::default();
    all_exports.insert("main".to_owned(), (0, ExportKind::Func));

    Module {
        globals: vec![Value::I32(7), Value::I64(99), Value::RefNull],
        memory,
        tables,
        tables_merkle,
        funcs,
        funcs_merkle,
        types: Arc::new(vec![function_type.clone()]),
        internals_offset: 0,
        names: Arc::new(NameCustomSection {
            module: "test_module".to_owned(),
            functions: Default::default(),
        }),
        host_call_hooks: Arc::new(vec![None, Some(("env".to_owned(), "alloc".to_owned()))]),
        start_function: Some(0),
        func_types: Arc::new(vec![function_type]),
        func_exports: Arc::new(func_exports),
        all_exports: Arc::new(all_exports),
        extra_hash: Arc::new(Bytes32([0xAA; 32])),
    }
}

#[test]
fn module_round_trip_preserves_hash() {
    // The core invariant: `Module::hash()` (consensus-locked) must be
    // identical after a serialize → deserialize round trip.
    let original = build_test_module();
    let original_hash = original.hash();

    let bytes = original.to_wavm_bytes().expect("encode should succeed");
    let rebuilt = Module::from_wavm_bytes(&bytes).expect("round-trip should succeed");

    assert_eq!(
        rebuilt.hash(),
        original_hash,
        "rebuilt module hash differs from original",
    );
}

fn assert_activation_round_trip(label: &str, wat: &[u8]) {
    let wasm = wasmer::wat2wasm(wat).expect("wat2wasm");

    let codehash = Bytes32::default();
    let stylus_version = 3u16;
    let mut gas = u64::MAX;
    let (module, _stylus_data) = Module::activate(
        &wasm,
        &codehash,
        stylus_version,
        0,
        65535,
        false,
        &mut gas,
        0,
    )
    .expect("activation");

    let activation_hash = module.hash();
    let bytes = module
        .to_wavm_bytes()
        .expect("to_wavm_bytes after activation");
    let rebuilt = Module::from_wavm_bytes(&bytes).expect("from_wavm_bytes after activation");

    assert_eq!(
        rebuilt.hash(),
        activation_hash,
        "[{label}] deserialized hash must match the activator's hash so LinkModule's lookup-by-hash matches the rebuilt module's identity",
    );
}

#[test]
fn activation_to_round_trip_hash_matches() {
    // Take a real Stylus user WASM through `Module::activate` (the
    // production activator path), serialize, deserialize, and assert hash
    // equality. The WAT declares a table with `initial > 0` and a
    // non-empty `elem` segment, so `Table::hash` is evaluated against a
    // non-trivially-shaped table; any future code that re-derives
    // `elems_merkle` from `elems` on one side but not the other would
    // change `tables_merkle.root()` and fail this assertion. (Today both
    // activator and deserializer hold `Merkle::default()` for this field;
    // the test guards against either side starting to populate it.)
    let minimal = br#"(module
        (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
        (memory (export "memory") 1 1)
        (table 2 funcref)
        (elem (i32.const 0) $user_entrypoint)
        (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
            i32.const 0
        )
    )"#;
    assert_activation_round_trip("minimal", minimal);

    // Richer WAT: exercises IBinOp (add/sub/mul), IRelOp (eq),
    // MemoryLoad/MemoryStore, branching (if/else), a multi-entry function
    // table, and a non-entrypoint helper. This is the configuration that
    // catches "an encoding bug in a non-hashed sub-arg surfaces only on
    // contracts with control flow"; the minimal WAT above does not reach
    // any of these opcodes through its 1-instruction body.
    let rich = br#"(module
        (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
        (memory (export "memory") 1 1)
        (table 2 funcref)
        (elem (i32.const 0) $user_entrypoint $helper)
        (func $helper (param i32) (result i32)
            local.get 0
            i32.const 7
            i32.add
        )
        (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
            (local $acc i32)
            ;; memory store + load
            i32.const 0
            i32.const 42
            i32.store
            i32.const 0
            i32.load
            local.set $acc
            ;; branch on argument
            local.get 0
            i32.const 0
            i32.eq
            if (result i32)
                local.get $acc
                i32.const 1
                i32.add
            else
                local.get $acc
                i32.const 1
                i32.sub
            end
            ;; call a helper to exercise an extra Function entry
            call $helper
            ;; final mul to exercise another IBinOp variant
            i32.const 3
            i32.mul
        )
    )"#;
    assert_activation_round_trip("rich", rich);
}

#[test]
fn non_hashed_fields_round_trip() {
    // `Module::hash()` (consensus) only covers globals/memory/tables_merkle/
    // funcs_merkle/extra_hash/internals_offset. Several fields round-trip
    // through the wire format but never contribute to the hash, so a
    // serialize/deserialize bug in any of them would slip past the hash
    // assertion in `module_round_trip_preserves_hash`.
    //
    // The most important of these is `TableElement::{val, func_ty}`; for
    // activator-shaped modules `Table::hash` only commits `elems.len()` +
    // empty `elems_merkle.root()`, so per-element data is invisible to the
    // hash until a `Machine` rebuilds `elems_merkle`. A bug here would
    // surface only mid-execution, far from the encode/decode site.
    let original = build_test_module();
    let bytes = original.to_wavm_bytes().unwrap();
    let r = Module::from_wavm_bytes(&bytes).expect("decode");

    assert_eq!(r.types, original.types, "types");
    assert_eq!(r.func_types, original.func_types, "func_types");
    assert_eq!(
        r.internals_offset, original.internals_offset,
        "internals_offset"
    );
    assert_eq!(r.names.module, original.names.module, "names.module");
    assert_eq!(
        r.names.functions, original.names.functions,
        "names.functions"
    );
    assert_eq!(
        r.host_call_hooks, original.host_call_hooks,
        "host_call_hooks"
    );
    assert_eq!(r.start_function, original.start_function, "start_function");
    assert_eq!(r.func_exports, original.func_exports, "func_exports");
    // `all_exports` is a HashMap — compare via sorted keys to be order-insensitive.
    let mut got: Vec<_> = r.all_exports.iter().collect();
    let mut want: Vec<_> = original.all_exports.iter().collect();
    got.sort_by_key(|(k, _)| (*k).clone());
    want.sort_by_key(|(k, _)| (*k).clone());
    assert_eq!(got, want, "all_exports");

    // Per-function: `ty` and `local_types` are not in Function::hash().
    assert_eq!(r.funcs.len(), original.funcs.len(), "funcs.len()");
    for (rf, of) in r.funcs.iter().zip(original.funcs.iter()) {
        assert_eq!(rf.ty, of.ty, "Function::ty");
        assert_eq!(rf.local_types, of.local_types, "Function::local_types");
        // (Function::code itself IS covered by code_merkle in Function::hash,
        // which is in turn covered by funcs_merkle in module_round_trip_preserves_hash.)
    }

    // Per-table: TableType invariants beyond element_type + elems.len() are
    // not in Table::hash() (Table::hash uses elems.len() + elems_merkle.root()).
    assert_eq!(r.tables.len(), original.tables.len(), "tables.len()");
    for (rt, ot) in r.tables.iter().zip(original.tables.iter()) {
        assert_eq!(
            rt.ty.element_type, ot.ty.element_type,
            "TableType.element_type"
        );
        assert_eq!(rt.ty.initial, ot.ty.initial, "TableType.initial");
        assert_eq!(rt.ty.maximum, ot.ty.maximum, "TableType.maximum");
        assert_eq!(rt.ty.table64, ot.ty.table64, "TableType.table64");
        assert_eq!(rt.ty.shared, ot.ty.shared, "TableType.shared");

        // The highest-leverage assertion: per-element `val` and `func_ty`.
        // `Table::hash` only commits `elems_merkle.root()` (empty for
        // activator modules), so without this check a corrupt encoder/
        // decoder for `TableElement` would silently round-trip.
        assert_eq!(rt.elems.len(), ot.elems.len(), "table.elems.len()");
        for (re, oe) in rt.elems.iter().zip(ot.elems.iter()) {
            assert_eq!(re.val, oe.val, "TableElement.val");
            assert_eq!(re.func_ty, oe.func_ty, "TableElement.func_ty");
        }
    }

    // memory.max_size is covered by Memory::hash, but explicit here for clarity.
    assert_eq!(
        r.memory.max_size, original.memory.max_size,
        "memory.max_size"
    );
    assert_eq!(r.memory.size(), original.memory.size(), "memory.size()");
}

#[test]
fn distinct_modules_differing_in_non_hashed_fields_hash_equally() {
    // Property: two modules that differ ONLY in fields outside `Module::hash`'s
    // commit set (globals + memory + tables_merkle + funcs_merkle + extra_hash
    // + internals_offset) must produce identical hashes. This is what catches
    // "a future PR adds a non-consensus field anywhere in the Module type
    // graph and on-chain `module_hash` silently shifts for every contract";
    // round-trip tests pass trivially in that scenario because encode→decode
    // is symmetric on the new field.
    let base_hash = build_test_module().hash();

    let check = |label: &str, m: &Module| {
        assert_eq!(
            m.hash(),
            base_hash,
            "[{label}] mutating a non-hashed field changed Module::hash() — \
             either the mutation accidentally touched a hashed input, or this \
             field actually IS in the hash",
        );
    };

    // 1. Module.types
    {
        let mut v = build_test_module();
        v.types = Arc::new(vec![]);
        check("Module.types", &v);
    }

    // 2. Module.names.module
    {
        let mut v = build_test_module();
        v.names = Arc::new(NameCustomSection {
            module: "a_different_module_name".to_owned(),
            functions: Default::default(),
        });
        check("Module.names.module", &v);
    }

    // 3. Module.names.functions
    {
        let mut v = build_test_module();
        let mut fnames: HashMap<u32, String> = HashMap::default();
        fnames.insert(0, "named_function".to_owned());
        v.names = Arc::new(NameCustomSection {
            module: "test_module".to_owned(),
            functions: fnames,
        });
        check("Module.names.functions", &v);
    }

    // 4. Module.host_call_hooks
    {
        let mut v = build_test_module();
        v.host_call_hooks = Arc::new(vec![Some(("env".to_owned(), "different_hook".to_owned()))]);
        check("Module.host_call_hooks", &v);
    }

    // 5. Module.start_function
    {
        let mut v = build_test_module();
        v.start_function = None;
        check("Module.start_function", &v);
    }

    // 6. Module.func_types
    {
        let mut v = build_test_module();
        v.func_types = Arc::new(vec![]);
        check("Module.func_types", &v);
    }

    // 7. Module.func_exports
    {
        let mut v = build_test_module();
        let mut e: HashMap<String, u32> = HashMap::default();
        e.insert("renamed_main".to_owned(), 0);
        v.func_exports = Arc::new(e);
        check("Module.func_exports", &v);
    }

    // 8. Module.all_exports
    {
        let mut v = build_test_module();
        let mut e = ExportMap::default();
        e.insert("renamed_main".to_owned(), (0, ExportKind::Func));
        v.all_exports = Arc::new(e);
        check("Module.all_exports", &v);
    }

    // 9. Function.ty and Function.local_types (keep code identical so the
    // function's code_merkle — and therefore Function::hash — is unchanged).
    // funcs_merkle is rebuilt from the mutated funcs to prove the property
    // rather than rely on a stale merkle root.
    {
        let mut v = build_test_module();
        let same_code = v.funcs[0].code.clone();
        let different_ty = FunctionType {
            inputs: vec![ArbValueType::F32],
            outputs: vec![ArbValueType::F64],
        };
        let different_locals = vec![ArbValueType::F32, ArbValueType::F64, ArbValueType::I32];
        let new_func = Function::new_from_wavm(same_code, different_ty, different_locals);
        v.funcs = Arc::new(vec![new_func]);
        v.funcs_merkle = Arc::new(Merkle::new(
            MerkleType::Function,
            v.funcs.iter().map(Function::hash).collect(),
        ));
        check("Function.ty + Function.local_types", &v);
    }

    // 10. TableElement.val and TableElement.func_ty. `Table::hash` reads
    // `elems_merkle.root()` + `elems.len()` + `ty`; for activator-shaped
    // tables `elems_merkle = Merkle::default()` so per-element bytes never
    // feed it. `elems.len()` and `ty` are unchanged here. Rebuild
    // tables_merkle from the mutated tables to prove the property.
    {
        let mut v = build_test_module();
        let different_func_ty = FunctionType {
            inputs: vec![ArbValueType::F32],
            outputs: vec![ArbValueType::F64],
        };
        v.tables[0].elems[0] = TableElement {
            func_ty: different_func_ty,
            val: Value::FuncRef(99),
        };
        let tables_hashes: Result<_> = v.tables.iter().map(Table::hash).collect();
        v.tables_merkle = Merkle::new(MerkleType::Table, tables_hashes.unwrap());
        check("TableElement.val + TableElement.func_ty", &v);
    }
}

#[test]
fn module_serialization_is_canonical() {
    // Same logical module → same bytes. Catches HashMap-iteration-order leaks
    // and any other non-determinism creeping into the wire format.
    let module = build_test_module();
    let a = module.to_wavm_bytes().unwrap();
    let b = module.to_wavm_bytes().unwrap();
    assert_eq!(a, b, "two serializations of the same module differ");

    // And byte-for-byte identical to a fresh round-trip's re-serialization.
    let rebuilt = Module::from_wavm_bytes(&a).unwrap();
    let c = rebuilt.to_wavm_bytes().unwrap();
    assert_eq!(
        a, c,
        "round-tripped module re-serializes to different bytes"
    );
}

#[test]
fn header_magic_and_version_are_emitted() {
    let bytes = build_test_module().to_wavm_bytes().unwrap();
    assert!(bytes.len() > 8);
    assert_eq!(&bytes[..4], WAVM_MAGIC);
    assert_eq!(&bytes[4..8], &WAVM_SERIALIZE_VERSION.to_be_bytes());
}

#[test]
fn bad_magic_is_rejected() {
    let mut bytes = build_test_module().to_wavm_bytes().unwrap();
    bytes[0] = b'X';
    let err = Module::from_wavm_bytes(&bytes).unwrap_err();
    assert!(err.to_string().contains("magic"), "unexpected error: {err}");
}

#[test]
fn wrong_version_is_rejected() {
    let mut bytes = build_test_module().to_wavm_bytes().unwrap();
    bytes[4..8].copy_from_slice(&WAVM_SERIALIZE_VERSION.wrapping_add(1).to_be_bytes());
    let err = Module::from_wavm_bytes(&bytes).unwrap_err();
    assert!(
        err.to_string().contains("WavmSerializeVersion"),
        "unexpected error: {err}"
    );
}

#[test]
fn truncated_header_is_rejected() {
    // Just the magic, no version byte and no body.
    let bytes = WAVM_MAGIC.to_vec();
    let err = Module::from_wavm_bytes(&bytes).unwrap_err();
    assert!(
        err.to_string().contains("too short"),
        "unexpected error: {err}"
    );
}

#[test]
fn empty_body_is_rejected() {
    // Valid header but no body — the decoder must error on the first missing
    // field rather than producing a half-built Module or panicking.
    let mut bytes = Vec::new();
    bytes.extend_from_slice(WAVM_MAGIC);
    bytes.extend_from_slice(&WAVM_SERIALIZE_VERSION.to_be_bytes());

    let result = Module::from_wavm_bytes(&bytes);
    assert!(result.is_err(), "empty body should error: {result:?}");
}

#[test]
fn truncation_anywhere_in_body_is_rejected() {
    // Build a valid module, then truncate the body at every byte boundary.
    // The positional decoder must report an error rather than panic for any
    // cut — coarser sweeps would miss truncations that land inside
    // u8/u16/u32/Bytes32 reads, which is where slice-indexing-without-bounds
    // regressions would surface.
    let bytes = build_test_module().to_wavm_bytes().unwrap();
    let header = WAVM_MAGIC.len() + 1;
    for cut in header + 1..bytes.len() {
        let result = Module::from_wavm_bytes(&bytes[..cut]);
        assert!(
            result.is_err(),
            "truncation at byte {cut} should error, got {result:?}",
        );
    }
}

#[test]
fn trailing_garbage_is_rejected() {
    // Positional decoder consumes exactly the bytes it needs; anything left
    // over indicates either corruption or an encoder bug, and must error
    // rather than silently succeed.
    let mut bytes = build_test_module().to_wavm_bytes().unwrap();
    bytes.extend_from_slice(b"unexpected trailing content");
    let err = Module::from_wavm_bytes(&bytes).unwrap_err();
    assert!(
        err.to_string().contains("trailing"),
        "expected trailing-bytes error, got: {err}",
    );
}

#[test]
fn corrupt_compressed_payload_is_rejected() {
    // Flip a byte inside the brotli stream. Decompression must fail
    // rather than producing partial / garbage output, and the error must
    // surface — not be silently fed to the inner cursor which could then
    // bail with a confusing wire-format error far from the real cause.
    let mut bytes = build_test_module().to_wavm_bytes().unwrap();
    // 9-byte envelope (MAGIC+VERSION+LEN) + 5 into the brotli payload.
    let target = WAVM_MAGIC.len() + 1 + 4 + 5;
    assert!(target < bytes.len(), "test setup: payload too small");
    bytes[target] ^= 0xFF;
    let err = Module::from_wavm_bytes(&bytes).unwrap_err();
    let msg = err.to_string();
    // The error should mention brotli — if we end up downstream of the
    // decompressor (e.g. with the inner cursor bailing on a garbage
    // count), the diagnostic for operators chasing a corrupt cache
    // entry becomes much harder to read.
    assert!(
        msg.contains("brotli"),
        "expected brotli decompression error, got: {err}",
    );
}

#[test]
fn linkmodule_hash_mismatch_is_detectable() {
    // Pins the property the `Opcode::LinkModule` arm in `Machine::step`
    // relies on: if wasmdb bytes stored under key K decode to a module
    // whose hash is K' != K, the mismatch is observable. The bail! site
    // needs a full `Machine` to exercise; this hits the detection logic
    // directly.
    //
    // Two distinct modules with different HASHED fields must produce
    // different `Module::hash()` outputs after a wire-format round trip.
    let mut module_a = build_test_module();
    let mut module_b = build_test_module();
    // Mutate a HASHED field on B so the hashes diverge. `extra_hash` is
    // covered by `Module::hash`, so changing it shifts B's hash.
    module_b.extra_hash = Arc::new(Bytes32([0xBB; 32]));

    let hash_a = module_a.hash();
    let hash_b = module_b.hash();
    assert_ne!(
        hash_a, hash_b,
        "test setup: A and B must have different hashes for the check to discriminate",
    );

    // Encode A's bytes and decode them. The decoded module's hash must
    // match A (the round-trip invariant) and must NOT match B (the
    // discriminating property the LinkModule check exploits).
    let bytes_a = module_a.to_wavm_bytes().expect("encode A");
    let decoded_hash = Module::from_wavm_bytes(&bytes_a).expect("decode A").hash();
    assert_eq!(
        decoded_hash, hash_a,
        "round-trip preserves the encoded module's hash",
    );
    assert_ne!(
        decoded_hash, hash_b,
        "if bytes for A were stored under B's wasmdb key, LinkModule's \
         `decoded.hash() != lookup_key` check would fire",
    );

    // Suppress unused-mut warnings; the modules are mutable so this
    // test can be extended to other hashed-field mutations without
    // restructuring.
    let _ = (&mut module_a, &mut module_b);
}

#[test]
fn compression_actually_shrinks_realistic_modules() {
    // Activate a realistic WAT with branching, calls, and arithmetic.
    // The output of `to_wavm_bytes` must be smaller than the inner body
    // — otherwise compression is silently disabled and we've regressed
    // to the 20–80x disk growth the wire format was designed to avoid.
    // This is a guard against future changes that accidentally skip
    // the brotli step (e.g. swapping `compress` for a passthrough).
    let wat = br#"(module
        (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
        (memory (export "memory") 4 4)
        (table 4 funcref)
        (elem (i32.const 0) $user_entrypoint $h1 $h2 $h3)
        (func $h1 (param i32) (result i32) local.get 0 i32.const 1 i32.add)
        (func $h2 (param i32) (result i32) local.get 0 i32.const 2 i32.mul)
        (func $h3 (param i32) (result i32) local.get 0 i32.const 3 i32.sub)
        (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
            (local $i i32) (local $acc i32)
            i32.const 0 local.set $acc
            i32.const 0 local.set $i
            (loop $L
                local.get $i i32.const 4 i32.rem_s
                local.get $acc i32.const 7 i32.add local.set $acc
                drop
                local.get $i i32.const 1 i32.add local.set $i
                local.get $i i32.const 64 i32.lt_s br_if $L
            )
            local.get $acc
        )
    )"#;
    let wasm = wasmer::wat2wasm(wat).expect("wat2wasm");
    let codehash = Bytes32::default();
    let mut gas = u64::MAX;
    let (module, _) =
        Module::activate(&wasm, &codehash, 3u16, 0, 65535, false, &mut gas, 0).expect("activate");

    let envelope = module.to_wavm_bytes().expect("to_wavm_bytes");
    // Envelope = MAGIC(4) + VERSION u32(4) + LEN u32(4) = 12 bytes.
    let envelope_overhead = WAVM_MAGIC.len() + 4 + 4;
    let compressed_payload_len = envelope.len() - envelope_overhead;

    // 2x is the floor; q=0 reliably hits ~20x on real modules.
    let rebuilt = Module::from_wavm_bytes(&envelope).expect("from_wavm_bytes");
    let body_len = {
        let again = rebuilt.to_wavm_bytes().unwrap();
        let mut env = Cursor::new(&again[WAVM_MAGIC.len() + 4..]);
        let compressed = env.read_bytes().unwrap();
        brotli::decompress(&compressed, Dictionary::Empty)
            .unwrap()
            .len()
    };
    assert!(
        compressed_payload_len * 2 < body_len,
        "brotli payload ({compressed_payload_len}) must compress the body \
         ({body_len}) by at least 2x; if this fails, compression is silently \
         disabled or window settings have regressed",
    );
}

/// Catches silent field swaps in the positional encoder — round-trip
/// tests miss adjacent same-width swaps. Pins keccak of the full envelope
/// (brotli output included). On legitimate format change: bump
/// `WAVM_SERIALIZE_VERSION` and paste the new `actual=` hex from the
/// failure into `WAVM_BLOB_LAYOUT_GOLDEN_HEX`.
#[test]
fn wavm_blob_layout_keccak_pinned_to_golden() {
    const WAVM_BLOB_LAYOUT_GOLDEN_HEX: &str =
        "2f845c1167e83eba161345841522a9de3e8192506500b8a43327c4753f77e105";

    let module = build_test_module();
    let bytes = module.to_wavm_bytes().expect("encode");
    let actual_hex = hex::encode(crypto::keccak(&bytes));

    assert_eq!(
        actual_hex, WAVM_BLOB_LAYOUT_GOLDEN_HEX,
        "WAVM wire-format keccak drifted; if this is intentional, bump \
         WAVM_SERIALIZE_VERSION and update WAVM_BLOB_LAYOUT_GOLDEN_HEX in \
         crates/prover/src/machine.rs. actual=0x{actual_hex}",
    );

    // Sanity: a mutated module must hash differently. Without this, a
    // regression encoding a constant would still match the pin.
    let mut mutated = build_test_module();
    mutated.internals_offset = mutated.internals_offset.wrapping_add(1);
    let mutated_bytes = mutated.to_wavm_bytes().expect("encode mutated");
    let mutated_hex = hex::encode(crypto::keccak(&mutated_bytes));
    assert_ne!(
        actual_hex, mutated_hex,
        "encoder must produce a different keccak after mutating a \
         persisted field; otherwise the pin is vacuous",
    );
}

/// Pins the failure branch: operator triage needs the marker line to
/// see that a wasmdb key is bad.
#[test]
fn write_modules_emits_corrupt_entry_marker() {
    let corrupt_bytes = vec![0xDE, 0xAD, 0xBE, 0xEF];
    let corrupt_hash = Bytes32([0x42; 32]);

    let mut machine = Machine::new_finished(GlobalState::default());
    machine.stylus_modules.insert(corrupt_hash, corrupt_bytes);

    let mut buf: Vec<u8> = Vec::new();
    machine.write_modules(&mut buf).expect("write_modules");
    let out = String::from_utf8(buf).expect("utf-8 output");

    assert!(
        out.contains("<failed to decode stylus module"),
        "corrupt entry must render the failure marker; got:\n{out}",
    );
}

/// Pins the hash hex in the bail message — without it, recovery becomes
/// "grep every wasmdb key".
#[test]
fn link_module_bail_names_missing_hash() {
    let missing_hash = Bytes32([0xAB; 32]);
    let modules: HashMap<Bytes32, Vec<u8>> = Default::default();
    let formatted = format_missing_stylus_module_error(missing_hash, &modules);
    let expected_hex = hex::encode(missing_hash.0);
    assert!(
        formatted.contains(&expected_hex),
        "LinkModule diagnostic must include the offending hash hex; got: {formatted}",
    );

    // Populated cache: hash must still be present (the key list is
    // auxiliary signal, not a substitute).
    let mut populated: HashMap<Bytes32, Vec<u8>> = Default::default();
    populated.insert(Bytes32([0x01; 32]), vec![]);
    populated.insert(Bytes32([0x02; 32]), vec![]);
    let formatted2 = format_missing_stylus_module_error(missing_hash, &populated);
    assert!(
        formatted2.contains(&expected_hex),
        "diagnostic must still name the missing hash when other keys are present; got: {formatted2}",
    );
}
