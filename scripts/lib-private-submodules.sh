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
# Sets `prefers_ssh=1` when the local environment indicates SSH is the
# preferred GitHub transport, using the first signal that matches:
#
#   1. `PREFER_SSH` env var — explicit override. `1`/`true`/`yes` forces
#      SSH; `0`/`false`/`no` forces HTTPS. Case-insensitive. Skips the
#      remaining checks either way.
#   2. A higher-scope (global/system, not local) insteadOf rule mapping
#      `https://github.com/` to `git@github.com:` — the classic "use SSH
#      for GitHub" convention.
#   3. `origin` URL scheme — if nitro-private itself was cloned over SSH
#      (`git@github.com:…` or `ssh://…`), the dev is already authing via
#      SSH to reach this repo, so private-submodule fetches should too.
#      This is the zero-config case for `ssh -A`-forwarded dev boxes.
#
# Write-side scripts use this to avoid shadowing a global HTTPS→SSH rule
# with a more-specific HTTPS→HTTPS-private rewrite (longest match wins,
# so our rule would otherwise force HTTPS auth on a dev set up for SSH),
# and to avoid prompting for HTTPS credentials on an SSH-only box.
#
# Local scope is skipped in the insteadOf scan because our own scripts
# may write an SSH-targeted local rule on a prior run; reading that back
# in as evidence would pin `prefers_ssh` on forever. `--show-scope`
# requires git ≥ 2.26; on older versions that scan is a no-op, leaving
# the origin-URL fallback to do the work.
#
# All three callers (configure / submodule-update / check) invoke this
# from the parent repo root, so the origin lookup always sees
# nitro-private's own origin — not a submodule's.
#
# shellcheck disable=SC2034  # prefers_ssh is consumed by callers after sourcing.
detect_prefers_ssh() {
  prefers_ssh=0
  case "${PREFER_SSH:-}" in
    1|true|TRUE|True|yes|YES|Yes)
      prefers_ssh=1
      return 0
      ;;
    0|false|FALSE|False|no|NO|No)
      return 0
      ;;
  esac
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
  # Origin-URL fallback. rc=2 is "no such remote" (fresh/bare repro);
  # anything else (including permission errors) is left unhandled on
  # purpose — this is a best-effort signal and an obscure failure here
  # must not abort the caller's bootstrap. prefers_ssh stays 0 in those
  # cases, matching pre-fallback behaviour.
  _rc=0
  _origin=$(git remote get-url origin 2>/dev/null) || _rc=$?
  if [ "$_rc" -eq 0 ] && [ -n "$_origin" ]; then
    case "$_origin" in
      git@github.com:*|ssh://git@github.com/*|ssh://git@github.com:*)
        prefers_ssh=1
        return 0
        ;;
    esac
  fi
}
