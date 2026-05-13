### Changed
- CI authenticates private submodule access via a GitHub App installation token instead of a long-lived personal access token. Workflows mint short-lived tokens via `actions/create-github-app-token` using `vars.NITRO_CI_APP_CLIENT_ID` and `secrets.NITRO_CI_APP_PRIVATE_KEY`. `PRIVATE_REPO_PAT` remains as a fallback during the migration window and will be removed in a follow-up.
