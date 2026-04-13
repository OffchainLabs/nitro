### Configuration
- Add `--stylus-target.native-stack-size` config to set the initial Wasmer coroutine stack size for Stylus execution.

### Fixed
- Fix Wasmer stack pool reusing stale smaller stacks after a stack size change.
- Automatically detect native stack overflow during Stylus execution and recover
