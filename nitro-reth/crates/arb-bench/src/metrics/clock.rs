use std::time::Instant;

/// Which process a [`Stopwatch`] bills CPU time to.
#[derive(Clone, Copy)]
pub struct CpuClock(Source);

#[derive(Clone, Copy)]
enum Source {
    /// This process, via `CLOCK_PROCESS_CPUTIME_ID`.
    ThisProcess,
    /// A resolved POSIX per-process CPU-time clock id for another process.
    OtherProcess(libc::clockid_t),
    /// No readable CPU clock; reads as zero.
    Unavailable,
}

impl CpuClock {
    pub fn this_process() -> Self {
        Self(Source::ThisProcess)
    }

    /// Resolve the CPU-time clock of another process, e.g. a spawned node.
    /// `None` when the platform or permissions do not allow reading it, in
    /// which case the caller should fall back to [`CpuClock::unavailable`]
    /// rather than to this process's clock, which would measure the wrong
    /// thing.
    pub fn for_pid(pid: u32) -> Option<Self> {
        pid_cpu_clock(pid).map(|id| Self(Source::OtherProcess(id)))
    }

    /// A clock that always reads zero, so an unmeasurable CPU time is reported
    /// as absent instead of being silently attributed to the wrong process.
    pub fn unavailable() -> Self {
        Self(Source::Unavailable)
    }

    fn now_ns(&self) -> u64 {
        match self.0 {
            Source::ThisProcess => process_cpu_ns(),
            Source::OtherProcess(id) => clock_ns(id),
            Source::Unavailable => 0,
        }
    }
}

/// Wall-clock + CPU-time scope.
pub struct Stopwatch {
    wall_start: Instant,
    cpu: CpuClock,
    cpu_start_ns: u64,
}

impl Stopwatch {
    /// Measure wall clock plus this process's CPU time.
    pub fn start() -> Self {
        Self::start_with(CpuClock::this_process())
    }

    /// Measure wall clock plus the CPU time of whichever process `cpu` names.
    pub fn start_with(cpu: CpuClock) -> Self {
        Self {
            wall_start: Instant::now(),
            cpu_start_ns: cpu.now_ns(),
            cpu,
        }
    }

    /// Returns `(wall_ns, cpu_ns)`. `cpu_ns` is 0 when the measured process has
    /// no readable CPU clock, including when it exited mid-scope.
    pub fn elapsed_ns(&self) -> (u64, u64) {
        let wall = self.wall_start.elapsed().as_nanos() as u64;
        let cpu = self.cpu.now_ns().saturating_sub(self.cpu_start_ns);
        (wall, cpu)
    }
}

#[cfg(target_os = "linux")]
fn pid_cpu_clock(pid: u32) -> Option<libc::clockid_t> {
    let mut clock_id: libc::clockid_t = 0;
    // SAFETY: `clock_getcpuclockid` writes through the pointer only when it
    // returns 0, which is checked before `clock_id` is read.
    let rc = unsafe { libc::clock_getcpuclockid(pid as libc::pid_t, &mut clock_id) };
    (rc == 0).then_some(clock_id)
}

/// Reading another process's CPU clock is a Linux extension; elsewhere the
/// caller gets `None` and reports the metric as absent.
#[cfg(not(target_os = "linux"))]
fn pid_cpu_clock(_pid: u32) -> Option<libc::clockid_t> {
    None
}

#[cfg(unix)]
fn clock_ns(clock_id: libc::clockid_t) -> u64 {
    use std::mem::MaybeUninit;
    let mut ts = MaybeUninit::<libc::timespec>::uninit();
    // SAFETY: `clock_gettime` writes a fully initialised `timespec` to the
    // pointer when it returns 0, which is checked before `assume_init`. On
    // non-zero return — including a process that has since exited — the buffer
    // is dropped without being read.
    let rc = unsafe { libc::clock_gettime(clock_id, ts.as_mut_ptr()) };
    if rc != 0 {
        return 0;
    }
    // SAFETY: `clock_gettime` returned 0 above, so `ts` is initialised.
    let ts = unsafe { ts.assume_init() };
    (ts.tv_sec as u64) * 1_000_000_000 + (ts.tv_nsec as u64)
}

#[cfg(target_os = "linux")]
fn process_cpu_ns() -> u64 {
    clock_ns(libc::CLOCK_PROCESS_CPUTIME_ID)
}

#[cfg(target_os = "macos")]
fn process_cpu_ns() -> u64 {
    unsafe extern "C" {
        fn clock_gettime_nsec_np(clock_id: u32) -> u64;
    }
    unsafe { clock_gettime_nsec_np(libc::CLOCK_PROCESS_CPUTIME_ID) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(target_os = "linux")]
    struct KillOnDrop(std::process::Child);

    #[cfg(target_os = "linux")]
    impl Drop for KillOnDrop {
        fn drop(&mut self) {
            let _ = self.0.kill();
            let _ = self.0.wait();
        }
    }

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

    #[test]
    fn unavailable_clock_reports_zero_cpu() {
        let sw = Stopwatch::start_with(CpuClock::unavailable());
        std::thread::sleep(std::time::Duration::from_millis(5));
        let (wall, cpu) = sw.elapsed_ns();
        assert!(wall > 0);
        assert_eq!(cpu, 0, "an unreadable clock must not invent CPU time");
    }

    /// The case the subprocess runner depends on: CPU is billed to the named
    /// process, not to whoever is holding the stopwatch. Two children are
    /// sampled over one window — one burning a core, one asleep. The contrast
    /// is against a second child rather than against this process, because
    /// `CLOCK_PROCESS_CPUTIME_ID` sums all threads and sibling tests share our
    /// thread group.
    #[cfg(target_os = "linux")]
    #[test]
    fn cpu_clock_bills_the_named_process() {
        use std::{
            process::{Command, Stdio},
            thread,
            time::Duration,
        };

        let busy = KillOnDrop(
            Command::new("sh")
                .arg("-c")
                .arg("while :; do :; done")
                .stdout(Stdio::null())
                .stderr(Stdio::null())
                .spawn()
                .expect("spawn busy child"),
        );
        let idle = KillOnDrop(
            Command::new("sleep")
                .arg("30")
                .spawn()
                .expect("spawn idle child"),
        );

        let busy_sw = Stopwatch::start_with(
            CpuClock::for_pid(busy.0.id()).expect("cpu clock resolves on linux"),
        );
        let idle_sw = Stopwatch::start_with(
            CpuClock::for_pid(idle.0.id()).expect("cpu clock resolves on linux"),
        );

        // Wait for the busy child to accrue an absolute amount of CPU rather
        // than asserting a share of a fixed window, which a loaded CI machine
        // does not guarantee.
        const TARGET_NS: u64 = 50_000_000;
        let deadline = Instant::now() + Duration::from_secs(10);
        let mut busy_cpu = 0;
        while busy_cpu < TARGET_NS && Instant::now() < deadline {
            thread::sleep(Duration::from_millis(10));
            busy_cpu = busy_sw.elapsed().cpu_ns.unwrap_or(0);
        }
        let idle_cpu = idle_sw.elapsed().cpu_ns;

        drop(busy);
        drop(idle);

        assert!(
            busy_cpu >= TARGET_NS,
            "busy child accrued only {busy_cpu}ns of cpu within the deadline"
        );
        let idle_cpu = idle_cpu.expect("idle child's clock stays readable while it is unreaped");
        assert!(
            idle_cpu < TARGET_NS / 10,
            "sleeping child burns nothing: cpu {idle_cpu} vs busy {busy_cpu}"
        );
    }
}
