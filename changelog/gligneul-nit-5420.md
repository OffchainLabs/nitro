### Internal
- Bound the CI go-tests build steps with timeouts and retry the build once if it hangs, instead of burning the full 90-minute job timeout when npm hangs after installing safe-smart-account dependencies.
