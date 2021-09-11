#![feature(start)]
#![no_std]

#[panic_handler]
fn panic(_: &core::panic::PanicInfo) -> ! {
	unsafe {
		core::arch::wasm32::unreachable();
	}
}

#[start]
fn main(argc: isize, _: *const *const u8) -> isize {
	let mut x: isize = 100;
	if argc == 0 {
		x = x.wrapping_add(1);
	}
	x
}
