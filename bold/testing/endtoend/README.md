# End to End Testing

## Host Requirements

The following tools are required to be on the host system and are not provided by bazel.

- [foundry](https://github.com/foundry-rs/foundry) -- Specifically the `anvil` tool.

## Running Tests

- Running with bazel is recommended, but not required. 
- The ANVIL environment variable is required or the test will attempt to guess where anvil is installed.

Note: The test which depend on running anvil should be run exclusively. Bazel targets that depend
on anvil should include "exclusive-if-local" or "exclusive" tags.

**Running with Bazel**

```
bazel test //testing/endtoend:endtoend_suite --test_env=ANVIL=$(which anvil) --test_output=all
```

**Running with Go**

```
ANVIL=$(which anvil) go test ./testing/endtoend/...
```
