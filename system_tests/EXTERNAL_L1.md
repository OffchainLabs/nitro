# Running system tests against an external L1

Root system tests that create an L1 launch `target/bin/geth` as the parent
chain. There is no in-process L1 fallback. Build the default binary with
`make build-upstream-geth`; a missing or invalid binary reports that command.
`NITRO_TEST_L1_GETH` is an optional override, and a configured but invalid
override fails instead of falling back to the default. Tests that do not create
an L1 never resolve the path. The independent `system_tests/v2` harness keeps
its own simulated L1.

The Makefile installs the configured upstream geth into `target/bin/geth`:

```sh
make build-upstream-geth
make test-system-external-l1-smoke
make test-system-external-l1 run='^TestStakersCooperative$'
```

To use another compatible geth binary:

```sh
NITRO_TEST_L1_GETH=/path/to/geth go test ./system_tests -run '^TestBlockHash$'
```

The external harness derives a developer genesis from geth, removes the
post-Osaka `bogotaTime` activation to match Nitro's current in-process test
configuration, starts geth over IPC, and seals submitted transactions with the
testing RPC. It also provides RPC-backed equivalents for the blob reader,
same-block transaction insertion, and L1 reorg operations used by system
tests.

The Go CI workflows build `target/bin/geth` and use it through the default
path.
