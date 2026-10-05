/*
 * Workaround for wasmer_vm referencing __rust_probestack on Linux with
 * Rust 1.93. In some link configurations the symbol is not exported from
 * compiler_builtins, leaving an undefined reference at link time. The
 * function is only called for very deep recursive stack frames, which
 * the Stylus runtime does not hit; an empty body is safe.
 *
 * Declared weak so the real compiler_builtins-provided symbol wins when
 * available.
 */
#if defined(__linux__) && (defined(__x86_64__) || defined(__i386__))
__attribute__((weak, used)) void __rust_probestack(void) {}
#endif
