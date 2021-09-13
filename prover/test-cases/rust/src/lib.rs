#![no_std]

#[panic_handler]
pub fn panic(_: &core::panic::PanicInfo) -> ! {
	core::arch::wasm32::unreachable();
}

