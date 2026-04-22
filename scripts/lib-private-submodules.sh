#!/bin/sh
# Helpers shared across the nitro-private submodule-routing scripts.
# Source via `. "$(dirname "$0")/lib-private-submodules.sh"` — do not
# execute directly.
#
# Deliberately NOT factored into this lib:
#   - The .private-submodules.conf parser. It is re-implemented inline in
#     .github/workflows/submodule-pin-check.yml (which must not run code
#     from HEAD under pull_request_target) and keeping the other copies
#     inline too lets their validation rules stay on-screen alongside the
#     code that uses them.
#
# POSIX-compatible so it runs under bash, dash, and make's default shell.

# normalize_github_url <url>
# Prints <owner>/<repo> (lowercased) for any GitHub-shaped URL we see in
# our submodules or origin remotes. Handles HTTPS, SSH, and the
# CodeStar-wrapped HTTPS form CodeBuild uses so origin checks classify
# a CodeBuild checkout as OffchainLabs, not external.
normalize_github_url() {
  printf '%s' "$1" | sed -E '
    s,^https?://([^@/]+@)?github\.com/,,
    s,^ssh://([^@]+@)?github\.com(:[0-9]+)?/,,
    s,^git@github\.com:,,
    s,^https?://([^@/]+@)?codestar-connections\.[^/]+\.amazonaws\.com/git-http/[^/]+/[^/]+/[^/]+/,,
    s,\.git$,,
    s,/+$,,
  ' | tr '[:upper:]' '[:lower:]'
}

# detect_prefers_ssh
# Sets `prefers_ssh=1` when a higher-scope (global/system, not local)
# insteadOf rule maps `https://github.com/` to `git@github.com:` (or an
# equivalent SSH form) — the classic "use SSH for GitHub" convention.
#
# Write-side scripts use this to avoid shadowing that global rule with a
# more-specific HTTPS→HTTPS-private rewrite (longest match wins, so our
# rule would otherwise force HTTPS auth on a dev set up for SSH).
#
# Local scope is skipped because our own scripts may write an
# SSH-targeted local rule on a prior run; reading that back in as
# evidence would pin `prefers_ssh` on forever. `--show-scope` requires
# git ≥ 2.26; on older versions this returns 0 leaving `prefers_ssh=0`,
# which matches the pre-SSH-aware behaviour.
#
# shellcheck disable=SC2034  # prefers_ssh is consumed by callers after sourcing.
detect_prefers_ssh() {
  prefers_ssh=0
  _rc=0
  _rules=$(git config --show-scope --get-regexp '^url\..+\.insteadof$' 2>/dev/null) || _rc=$?
  if [ "$_rc" -ne 0 ] && [ "$_rc" -ne 1 ]; then
    return 0
  fi
  while IFS= read -r _line; do
    [ -z "$_line" ] && continue
    _scope=${_line%%	*}
    case "$_scope" in local) continue;; esac
    _rest=${_line#*	}
    _key=${_rest%% *}
    _val=${_rest#* }
    _base=${_key#url.}
    _base=${_base%.insteadof}
    case "$_base" in
      git@github.com:|ssh://git@github.com/|ssh://git@github.com/*)
        case "$_val" in
          https://github.com/|http://github.com/)
            prefers_ssh=1
            return 0
            ;;
        esac
        ;;
    esac
  done <<EOF
$_rules
EOF
}
