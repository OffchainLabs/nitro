;; Template for a single-hostio Stylus reference program.
;;
;; A Stylus contract is a WebAssembly module that:
;;   1) imports host functions from the "vm_hooks" namespace,
;;   2) exports a `memory` of at least one page,
;;   3) exports `user_entrypoint(args_len: i32) -> i32` (return 0 = Ok, 1 = Revert).
;;
;; The runtime delivers calldata into linear memory via `read_args(dest_ptr)`
;; and reads the return payload via `write_result(src_ptr, len)`. The argument
;; passed to `user_entrypoint` is the calldata length in bytes.
;;
;; To author a new hostio program:
;;   1) Replace `<HOSTIO>` below with the hostio name (must match arbreth's
;;      `arb-stylus::native::imports!()` registration; same names as Nitro's
;;      `vm_hooks` module).
;;   2) Set the parameter / result types to match the hostio signature.
;;   3) Lay out calldata at offset 0 and return data at offset 0 (or wherever).
;;   4) Validate by running `wat::parse_str` against the file in a unit test.
;;
;; Convention for fixture programs: keep memory at exactly 1 page; pass
;; arguments via calldata only; place writeable scratch at offset 0; place the
;; return payload contiguously starting at a higher offset (e.g., 0x100). This
;; gives fixture authors deterministic memory layouts when crafting calldata.

(module
    ;; TODO: import the hostio under test. Replace name + signature.
    ;; Reference: arb-stylus/src/native.rs lists every "vm_hooks" import.
    (import "vm_hooks" "<HOSTIO>" (func $hostio (param i32 i32) (result)))

    ;; Standard plumbing — every fixture needs these two.
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))

    (memory (export "memory") 1)

    (func $user_entrypoint (export "user_entrypoint") (param $args_len i32) (result i32)
        ;; 1) Pull calldata into memory at offset 0.
        (call $read_args (i32.const 0))

        ;; 2) Invoke the hostio. Adjust pointers + literals to taste.
        ;;    Example shape:
        ;;        (call $hostio (i32.const 0) (i32.const 0x100))
        ;; TODO

        ;; 3) Hand the runtime back a return payload (zero-length is fine).
        (call $write_result (i32.const 0x100) (i32.const 0))

        ;; 4) Status code: 0 = Ok, 1 = Revert.
        i32.const 0
    )
)
