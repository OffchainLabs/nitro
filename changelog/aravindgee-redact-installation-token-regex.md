### Fixed
- CI error-log redaction for the installation-token fetch now covers GitHub's new stateless token format (`ghs_<id>_<base64url JWT>`, containing `.` and `-`); the old `[A-Za-z0-9]+` character class stopped redacting at the first `.`/`-`, which could leak part of a real token into build logs
