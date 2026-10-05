### Internal
- Bound the CI go-tests build steps with timeouts and retry the build if it hangs, instead of burning the full 90-minute job timeout when npm hangs after installing safe-smart-account dependencies. When a hang is detected, capture diagnostics from the hung processes and upload them as a CI artifact to support an upstream bug report.
