#!/usr/bin/env python3
"""
Benchmark validator throughput in native and continuous modes.

For each mode:
  1. Starts the validator server (locally or via Docker)
  2. Waits for it to become ready (health-check polling)
  3. For every test-block directory under system_tests/target/TestRecord*,
     sends N requests and measures client-side latency
  4. Computes min / median / mean / p95 / max per block
  5. Prints a summary table and picks the fastest mode

Usage:
    # Local binary:
    python3 benchmark_validator.py [--runs 20] [--warmup 3] [--base-dir system_tests/target]

    # Docker (builds image if needed, uses --network=host):
    python3 benchmark_validator.py --docker [--docker-image nitro-validator] \
        [--machines-dir target/machines] [--base-dir system_tests/target]
"""

import argparse
import glob
import json
import os
import signal
import statistics
import subprocess
import sys
import time
import urllib.request
import urllib.error


# ── helpers ──────────────────────────────────────────────────────────────────

def percentile(data, p):
    """Return the p-th percentile of a sorted list."""
    k = (len(data) - 1) * (p / 100)
    f = int(k)
    c = f + 1
    if c >= len(data):
        return data[f]
    return data[f] + (k - f) * (data[c] - data[f])


def send_request(block_inputs_path: str, url: str) -> float:
    """Send one validation request and return latency in ms."""
    with open(block_inputs_path, "r") as f:
        block_input = json.load(f)

    payload = json.dumps({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "validation_validate",
        "params": [block_input],
    }).encode()

    req = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=120) as resp:
        body = resp.read()
    elapsed_ms = (time.perf_counter() - start) * 1000

    data = json.loads(body)
    if "error" in data:
        raise RuntimeError(f"JSON-RPC error: {data['error']}")
    result = data.get("result")
    if not isinstance(result, dict):
        raise RuntimeError(f"Unexpected response (no 'result' object): {data}")
    for key in ("BlockHash", "SendRoot", "Batch", "PosInBatch"):
        if key not in result:
            raise RuntimeError(f"Missing '{key}' in result: {result}")

    return elapsed_ms


def wait_for_server(url: str, timeout: int = 30):
    """Poll the server until it responds or timeout is reached."""
    # Derive a health check URL using the /validation_name GET endpoint
    # instead of POSTing to /validation_validate with a dummy body
    # (which returns HTTP 4xx and gets caught as URLError).
    from urllib.parse import urlparse, urlunparse
    parsed = urlparse(url)
    health_url = urlunparse(parsed._replace(path="/validation_name"))

    deadline = time.time() + timeout
    # Give the process a moment to start
    time.sleep(2)
    elapsed = 0
    while time.time() < deadline:
        try:
            req = urllib.request.Request(health_url, method="GET")
            with urllib.request.urlopen(req, timeout=5) as resp:
                resp.read()
            return  # server is up
        except (urllib.error.URLError, ConnectionRefusedError, OSError):
            elapsed = int(time.time() + timeout - deadline)
            if elapsed > 0 and elapsed % 15 == 0:
                print(f"    Still waiting for server... ({elapsed}s elapsed)")
            time.sleep(1)
    raise TimeoutError(f"Server at {url} did not become ready within {timeout}s")


def start_server_local(mode: str, validator_bin: str):
    """Start the validator as a local process, return the Popen object."""
    env = os.environ.copy()
    env["RUST_LOG"] = "tower_http=debug,info"
    proc = subprocess.Popen(
        [validator_bin, "--mode", mode],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        preexec_fn=os.setsid,  # so we can kill the whole process group
    )
    return proc


def stop_server_local(proc: subprocess.Popen):
    """Gracefully stop a locally-running validator server."""
    if proc.poll() is not None:
        return
    try:
        os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
        proc.wait(timeout=10)
    except (ProcessLookupError, subprocess.TimeoutExpired):
        try:
            os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
            proc.wait(timeout=5)
        except Exception:
            pass


def docker_image_exists(image: str) -> bool:
    """Check if a Docker image exists locally."""
    result = subprocess.run(
        ["docker", "image", "inspect", image],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return result.returncode == 0


def docker_build(image: str):
    """Build the validator Docker image from Dockerfile.validator."""
    dockerfile = "Dockerfile.validator"
    if not os.path.isfile(dockerfile):
        print(f"ERROR: {dockerfile} not found in current directory", file=sys.stderr)
        sys.exit(1)

    print(f"  Building Docker image '{image}' from {dockerfile} (target: nitro-validator)...")
    result = subprocess.run(
        ["docker", "build", "-f", dockerfile, "--target", "nitro-validator", "-t", image, "."],
        timeout=7200,  # 2 hour max for full build
    )
    if result.returncode != 0:
        print(f"ERROR: Docker build failed with exit code {result.returncode}", file=sys.stderr)
        sys.exit(1)
    print(f"  Docker image '{image}' built successfully.")


def start_server_docker(mode: str, image: str, machines_dir: str, container_name: str):
    """
    Start the validator in Docker with --network=host.
    Returns the container name (used to stop it later).
    If machines_dir is provided and exists, mount it to override the built-in machines.
    """
    cmd = [
        "docker", "run", "--rm", "-d",
        "--name", container_name,
        "-p", "4141:4141",
        "-e", "RUST_LOG=tower_http=debug,info",
    ]

    machines_abs = os.path.abspath(machines_dir)
    if os.path.isdir(machines_abs):
        cmd.extend(["-v", f"{machines_abs}:/machines:ro"])

    cmd.extend([
        image,
        "--mode", mode,
    ])

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"ERROR: Failed to start Docker container: {result.stderr.strip()}", file=sys.stderr)
        sys.exit(1)

    # Verify the container didn't crash immediately
    time.sleep(1)
    check = subprocess.run(
        ["docker", "inspect", "-f", "{{.State.Running}}", container_name],
        capture_output=True, text=True,
    )
    if check.stdout.strip() != "true":
        logs = subprocess.run(
            ["docker", "logs", "--tail", "20", container_name],
            capture_output=True, text=True,
        )
        print(f"ERROR: Container '{container_name}' exited immediately.", file=sys.stderr)
        if logs.stderr.strip():
            print(f"Container logs:\n{logs.stderr.strip()}", file=sys.stderr)
        if logs.stdout.strip():
            print(f"Container logs:\n{logs.stdout.strip()}", file=sys.stderr)
        sys.exit(1)

    return container_name


def stop_server_docker(container_name: str):
    """Stop a Docker container by name."""
    subprocess.run(
        ["docker", "stop", "-t", "10", container_name],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


# ── benchmark logic ──────────────────────────────────────────────────────────

def benchmark_mode(
    mode: str,
    test_blocks: list[str],
    url: str,
    runs: int,
    warmup: int,
    # Local mode params
    validator_bin: str = "",
    # Docker mode params
    use_docker: bool = False,
    docker_image: str = "",
    machines_dir: str = "",
) -> dict[str, dict]:
    """
    Benchmark all test blocks for a single mode.
    Returns {test_name: {mean, median, min, max, p95, latencies}}.
    """
    print(f"\n{'='*72}")
    print(f"  Mode: {mode.upper()}")
    print(f"{'='*72}")

    container_name = f"validator-bench-{mode}"
    proc = None

    if use_docker:
        start_server_docker(mode, docker_image, machines_dir, container_name)
    else:
        proc = start_server_local(mode, validator_bin)

    try:
        print(f"  Waiting for server to start...")
        wait_for_server(url, timeout=300 if use_docker else 120)
        print(f"  Server is up. Starting benchmark ({runs} runs + {warmup} warmup per block).\n")

        results = {}
        for i, block_dir in enumerate(test_blocks, 1):
            test_name = os.path.basename(block_dir)
            inputs_path = os.path.join(block_dir, "block_inputs.json")

            if not os.path.isfile(inputs_path):
                print(f"  [{i}/{len(test_blocks)}] {test_name}: SKIPPED (no block_inputs.json)")
                continue

            # Warmup runs (not counted)
            for w in range(warmup):
                try:
                    send_request(inputs_path, url)
                except Exception as e:
                    print(f"  [{i}/{len(test_blocks)}] {test_name}: warmup {w+1} failed: {e}")

            # Measured runs
            latencies = []
            for r in range(runs):
                try:
                    lat = send_request(inputs_path, url)
                    latencies.append(lat)
                except Exception as e:
                    print(f"  [{i}/{len(test_blocks)}] {test_name}: run {r+1} failed: {e}")

            if not latencies:
                print(f"  [{i}/{len(test_blocks)}] {test_name}: ALL RUNS FAILED")
                continue

            latencies.sort()
            stats = {
                "mean": statistics.mean(latencies),
                "median": statistics.median(latencies),
                "min": min(latencies),
                "max": max(latencies),
                "p95": percentile(latencies, 95),
                "latencies": latencies,
            }
            results[test_name] = stats
            print(
                f"  [{i:2d}/{len(test_blocks)}] {test_name:<45s} "
                f"mean={stats['mean']:7.1f}ms  median={stats['median']:7.1f}ms  "
                f"p95={stats['p95']:7.1f}ms"
            )

        return results

    finally:
        print(f"\n  Stopping {mode} server...")
        if use_docker:
            stop_server_docker(container_name)
        else:
            stop_server_local(proc)
        # Give port time to release
        time.sleep(2)


def print_table(mode: str, results: dict[str, dict]):
    """Print a formatted results table for one mode."""
    if not results:
        print(f"\nNo results for mode: {mode}")
        return

    print(f"\n{'─'*100}")
    print(f"  Results: {mode.upper()} mode")
    print(f"{'─'*100}")
    header = f"  {'Test Block':<45s} {'Mean':>8s} {'Median':>8s} {'Min':>8s} {'P95':>8s} {'Max':>8s}"
    print(header)
    print(f"  {'─'*85}")

    for test_name in sorted(results.keys()):
        s = results[test_name]
        print(
            f"  {test_name:<45s} "
            f"{s['mean']:7.1f}ms {s['median']:7.1f}ms {s['min']:7.1f}ms "
            f"{s['p95']:7.1f}ms {s['max']:7.1f}ms"
        )

    all_means = [s["mean"] for s in results.values()]
    overall = statistics.mean(all_means)
    print(f"  {'─'*85}")
    print(f"  {'OVERALL AVERAGE':<45s} {overall:7.1f}ms")


def print_comparison(all_results: dict[str, dict[str, dict]]):
    """Print a side-by-side comparison of modes."""
    modes = list(all_results.keys())
    if len(modes) < 2:
        return

    # Get union of all test names
    all_tests = set()
    for r in all_results.values():
        all_tests.update(r.keys())
    all_tests = sorted(all_tests)

    print(f"\n{'='*100}")
    print(f"  COMPARISON: {' vs '.join(m.upper() for m in modes)}")
    print(f"{'='*100}")

    header = f"  {'Test Block':<45s}"
    for mode in modes:
        header += f" {mode.upper():>10s}"
    header += f" {'Winner':>12s}"
    print(header)
    print(f"  {'─'*90}")

    mode_wins = {m: 0 for m in modes}

    for test_name in all_tests:
        row = f"  {test_name:<45s}"
        means = {}
        for mode in modes:
            if test_name in all_results[mode]:
                m = all_results[mode][test_name]["mean"]
                means[mode] = m
                row += f" {m:9.1f}ms"
            else:
                row += f" {'N/A':>10s}"

        if len(means) == len(modes):
            winner = min(means, key=means.get)
            mode_wins[winner] += 1
            row += f"  {winner:>10s}"
        else:
            row += f"  {'???':>10s}"

        print(row)

    print(f"  {'─'*90}")

    # Overall averages
    row = f"  {'OVERALL AVERAGE':<45s}"
    overall = {}
    for mode in modes:
        if all_results[mode]:
            avg = statistics.mean(s["mean"] for s in all_results[mode].values())
            overall[mode] = avg
            row += f" {avg:9.1f}ms"
        else:
            row += f" {'N/A':>10s}"
    if overall:
        best = min(overall, key=overall.get)
        row += f"  {best:>10s}"
    print(row)

    print(f"\n  Wins per mode: ", end="")
    print(", ".join(f"{m.upper()}={mode_wins[m]}" for m in modes))

    if overall:
        best = min(overall, key=overall.get)
        speedup = max(overall.values()) / min(overall.values())
        print(f"\n  >>> RECOMMENDATION: Use {best.upper()} mode ({speedup:.2f}x faster on average) <<<")


# ── main ─────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="Benchmark validator modes")
    parser.add_argument("--runs", type=int, default=20, help="Measured runs per block (default: 20)")
    parser.add_argument("--warmup", type=int, default=3, help="Warmup runs per block (default: 3)")
    parser.add_argument("--base-dir", default="system_tests/target", help="Directory with test blocks")
    parser.add_argument("--url", default="http://localhost:4141/validation_validate", help="Validator URL")
    parser.add_argument("--modes", nargs="+", default=["native", "continuous"], help="Modes to benchmark")

    # Local mode options
    parser.add_argument("--validator-bin", default="target/bin/validator", help="Path to validator binary (local mode)")

    # Docker mode options
    parser.add_argument("--docker", action="store_true", help="Run validator inside Docker (uses --network=host)")
    parser.add_argument("--docker-image", default="nitro-validator", help="Docker image name (default: nitro-validator)")
    parser.add_argument("--machines-dir", default="target/machines", help="Path to machines directory (Docker mode)")
    parser.add_argument("--docker-build", action="store_true", help="Force rebuild of Docker image even if it exists")

    args = parser.parse_args()

    # Docker validation and setup
    if args.docker:
        if args.docker_build or not docker_image_exists(args.docker_image):
            docker_build(args.docker_image)
        else:
            print(f"Using existing Docker image '{args.docker_image}' (use --docker-build to force rebuild)")

        machines_abs = os.path.abspath(args.machines_dir)
        if os.path.isdir(machines_abs):
            print(f"  Mounting local machines from {machines_abs}")
        else:
            print(f"  Using machines baked into the Docker image (no local {machines_abs} found)")

    # Discover test blocks
    test_blocks = sorted(glob.glob(os.path.join(args.base_dir, "TestRecord*")))
    if not test_blocks:
        print(f"ERROR: No TestRecord* directories found in {args.base_dir}", file=sys.stderr)
        sys.exit(1)

    print(f"Found {len(test_blocks)} test blocks in {args.base_dir}")
    print(f"Runs per block: {args.runs} (+{args.warmup} warmup)")
    print(f"Modes: {', '.join(args.modes)}")
    if args.docker:
        print(f"Runner: Docker ({args.docker_image}) with --network=host")
        print(f"Machines: {os.path.abspath(args.machines_dir)}")
    else:
        print(f"Validator: {args.validator_bin}")

    all_results = {}
    for mode in args.modes:
        all_results[mode] = benchmark_mode(
            mode=mode,
            test_blocks=test_blocks,
            url=args.url,
            runs=args.runs,
            warmup=args.warmup,
            validator_bin=args.validator_bin,
            use_docker=args.docker,
            docker_image=args.docker_image,
            machines_dir=args.machines_dir,
        )

    # Print per-mode tables
    for mode in args.modes:
        print_table(mode, all_results[mode])

    # Print comparison
    print_comparison(all_results)

    # Write raw results to JSON for later analysis
    output_file = "benchmark_results.json"
    export = {}
    for mode, results in all_results.items():
        export[mode] = {}
        for test_name, stats in results.items():
            export[mode][test_name] = {
                "mean_ms": round(stats["mean"], 2),
                "median_ms": round(stats["median"], 2),
                "min_ms": round(stats["min"], 2),
                "max_ms": round(stats["max"], 2),
                "p95_ms": round(stats["p95"], 2),
                "latencies_ms": [round(l, 2) for l in stats["latencies"]],
            }
    with open(output_file, "w") as f:
        json.dump(export, f, indent=2)
    print(f"\nRaw results saved to {output_file}")


if __name__ == "__main__":
    main()
