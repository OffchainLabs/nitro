use sp1_zkvm::{io, syscalls};
use validation::ValidationInput;

pub fn print_string(fd: u32, bytes: &[u8]) {
    syscalls::syscall_write(fd, bytes.as_ptr(), bytes.len());
}

pub fn read_input() -> ValidationInput {
    let s = io::read::<Vec<u8>>();
    ValidationInput::from_reader(std::io::Cursor::new(s)).expect("parse input file")
}

pub fn exit(code: u32) -> ! {
    syscalls::syscall_halt(code as u8)
}

pub fn dump_elf() {
    syscalls::syscall_dump_elf();
}
