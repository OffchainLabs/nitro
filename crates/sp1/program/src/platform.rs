use sp1_zkvm::{io, syscalls};
use validation::ValidationInput;

pub fn print_string(fd: u32, bytes: &[u8]) {
    syscalls::syscall_write(fd, bytes.as_ptr(), bytes.len());
}

pub fn read_input() -> ValidationInput {
    let s = io::read::<Vec<u8>>();
    if s.is_empty() {
        // Bootload-only sentinel: the builder feeds an empty payload so that
        // after the `beforeFirstIO` hook has dumped the ELF there is nothing
        // left to do. Real validation runs always provide a non-empty
        // rkyv-encoded `ValidationInput`, so this branch is never taken there.
        exit(0);
    }
    ValidationInput::from_reader(std::io::Cursor::new(s)).expect("parse input file")
}

pub fn exit(code: u32) -> ! {
    syscalls::syscall_halt(code as u8)
}

pub fn dump_elf() {
    syscalls::syscall_dump_elf();
}
