use std::time::Instant;

/// Wall-clock + CPU-time scope.
pub struct Stopwatch {
    wall_start: Instant,
    cpu_start_ns: u64,
}

impl Stopwatch {
    pub fn start() -> Self {
        Self {
            wall_start: Instant::now(),
            cpu_start_ns: process_cpu_ns(),
        }
    }

    pub fn elapsed_ns(&self) -> (u64, u64) {
        let wall = self.wall_start.elapsed().as_nanos() as u64;
        let cpu = process_cpu_ns().saturating_sub(self.cpu_start_ns);
        (wall, cpu)
    }
}

#[cfg(target_os = "linux")]
fn process_cpu_ns() -> u64 {
    use std::mem::MaybeUninit;
    let mut ts = MaybeUninit::<libc::timespec>::uninit();
    // CLOCK_PROCESS_CPUTIME_ID = 2
    // SAFETY: `clock_gettime` writes a fully initialised `timespec` to
    // the pointer when it returns 0, which is checked before
    // `assume_init`. On non-zero return, the buffer is dropped without
    // being read.
    let rc = unsafe { libc::clock_gettime(2, ts.as_mut_ptr()) };
    if rc != 0 {
        return 0;
    }
    // SAFETY: `clock_gettime` returned 0 above, so `ts` is initialised.
    let ts = unsafe { ts.assume_init() };
    (ts.tv_sec as u64) * 1_000_000_000 + (ts.tv_nsec as u64)
}

#[cfg(target_os = "macos")]
fn process_cpu_ns() -> u64 {
    use std::mem::MaybeUninit;
    extern "C" {
        fn clock_gettime_nsec_np(clock_id: u32) -> u64;
    }
    // CLOCK_PROCESS_CPUTIME_ID == 12 on Darwin.
    // SAFETY: `clock_gettime_nsec_np` is a Darwin-only libc call that
    // returns a nanosecond reading or 0 on error; no buffers are passed.
    let nsec = unsafe { clock_gettime_nsec_np(12) };
    if nsec != 0 {
        return nsec;
    }
    // Fallback to monotonic.
    let mut ts = MaybeUninit::<libc::timespec>::uninit();
    // SAFETY: see the linux branch above — `assume_init` only after a
    // zero return from `clock_gettime`.
    let rc = unsafe { libc::clock_gettime(libc::CLOCK_MONOTONIC, ts.as_mut_ptr()) };
    if rc != 0 {
        return 0;
    }
    // SAFETY: `clock_gettime` returned 0, so `ts` is initialised.
    let ts = unsafe { ts.assume_init() };
    (ts.tv_sec as u64) * 1_000_000_000 + (ts.tv_nsec as u64)
}

#[cfg(not(any(target_os = "linux", target_os = "macos")))]
fn process_cpu_ns() -> u64 {
    Instant::now().elapsed().as_nanos() as u64
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stopwatch_increases() {
        let sw = Stopwatch::start();
        let mut acc = 0u64;
        for i in 0..1_000_000u64 {
            acc = acc.wrapping_add(i);
        }
        std::hint::black_box(acc);
        let (wall, _cpu) = sw.elapsed_ns();
        assert!(wall > 0);
    }
}
