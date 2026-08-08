fn main() {
    if cfg!(all(
        target_os = "linux",
        any(target_arch = "x86_64", target_arch = "x86")
    )) {
        cc::Build::new()
            .file("probestack_stub.c")
            .compile("arb_stylus_probestack_stub");
        println!("cargo:rerun-if-changed=probestack_stub.c");
    }
}
