//! Reaps harness child nodes orphaned by a previous run.
//!
//! Nodes live behind a process-wide `OnceLock`, which Rust never drops at
//! exit, and the runner SIGKILLs timed-out tests — in both cases the node's
//! `Drop` cannot run, leaking its Docker container or `arb-reth` process. Each
//! is tagged with the PID of the test process that spawned it; at startup we
//! remove any tag whose owner PID is no longer alive. Live siblings are kept,
//! so the sweep is safe under parallel test execution.

use std::{
    process::{Command, Stdio},
    sync::Once,
};

/// Label key carrying the owner PID on harness Nitro containers.
pub const OWNER_PID_LABEL: &str = "arb-harness-owner-pid";

/// Substring identifying a harness `arb-reth` process by its datadir path.
const ARBRETH_MARKER: &str = "arb-harness-arbreth-";

fn pid_alive(pid: u32) -> bool {
    Command::new("kill")
        .args(["-0", &pid.to_string()])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .map(|s| s.success())
        .unwrap_or(false)
}

fn docker_rm(id: &str) {
    let _ = Command::new("docker")
        .args(["rm", "-f", id])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status();
}

/// Remove harness Nitro containers whose owner test process has exited.
pub fn reap_orphan_containers() {
    static ONCE: Once = Once::new();
    ONCE.call_once(|| {
        let out = Command::new("docker")
            .args([
                "ps",
                "-a",
                "--filter",
                &format!("label={OWNER_PID_LABEL}"),
                "--format",
                &format!("{{{{.ID}}}} {{{{.Label \"{OWNER_PID_LABEL}\"}}}}"),
            ])
            .output();
        let Ok(out) = out else { return };
        for line in String::from_utf8_lossy(&out.stdout).lines() {
            let Some((id, pid)) = line.split_once(' ') else {
                continue;
            };
            match pid.trim().parse::<u32>() {
                Ok(pid) if !pid_alive(pid) => docker_rm(id),
                _ => {}
            }
        }
    });
}

/// Kill harness `arb-reth` processes whose owner test process has exited.
pub fn reap_orphan_arbreth() {
    static ONCE: Once = Once::new();
    ONCE.call_once(|| {
        let out = Command::new("pgrep").args(["-af", ARBRETH_MARKER]).output();
        let Ok(out) = out else { return };
        for line in String::from_utf8_lossy(&out.stdout).lines() {
            let Some((proc_pid, cmd)) = line.split_once(' ') else {
                continue;
            };
            let Some(owner) = owner_pid_from_cmd(cmd) else {
                continue;
            };
            if !pid_alive(owner) {
                let _ = Command::new("kill")
                    .args(["-9", proc_pid.trim()])
                    .stdout(Stdio::null())
                    .stderr(Stdio::null())
                    .status();
            }
        }
    });
}

fn owner_pid_from_cmd(cmd: &str) -> Option<u32> {
    let rest = cmd.split(ARBRETH_MARKER).nth(1)?;
    rest.split('-').next()?.parse().ok()
}
