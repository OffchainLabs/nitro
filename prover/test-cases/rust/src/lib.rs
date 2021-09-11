#![no_std]

#[panic_handler]
pub fn panic(_: &core::panic::PanicInfo) -> ! {
	unsafe {
		core::arch::wasm32::unreachable();
	}
}

