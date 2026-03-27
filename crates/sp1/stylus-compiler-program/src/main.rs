#![cfg_attr(target_os = "zkvm", no_main)]

#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

fn main() {
    let input = sp1_zkvm::io::read::<stylus_compiler_program::CompileInput>();
    let rv64_binary = stylus_compiler_program::compile(&input)
        .expect("stylus compilation failed");
    sp1_zkvm::io::commit(&rv64_binary);
}

// Those are referenced by wasmer runtimes, but are never invoked
#[unsafe(no_mangle)]
pub extern "C" fn __negdf2(_x: f64) -> f64 {
    unreachable!()
}

#[unsafe(no_mangle)]
pub extern "C" fn __negsf2(_x: f32) -> f32 {
    unreachable!()
}
