### Fixed
- CI no longer rejects GitHub's new stateless installation-token format (`ghs_<id>_<base64url JWT>`, containing `.` and `-`): the init-submodules token guard now only checks for empty/whitespace tokens instead of allow-listing an alphabet
- Removed the retired `PRIVATE_REPO_PAT` fallback from all CI workflows; the GitHub App is the sole auth source for private-submodule access
