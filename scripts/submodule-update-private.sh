#!/bin/sh
# Run `git submodule sync/update --init --recursive` with the url.* rewrites
# from .private-submodules.conf injected via GIT_CONFIG_COUNT env vars, so
# that git subprocesses spawned *inside* each newly-cloned submodule also
# see the rewrites.
#
# Without this, a fresh `git clone` of nitro-private followed by
# `git submodule update --init --recursive` fails when git fetches a pinned
# SHA that only exists in the -private fork: the parent-scope insteadOf
# rule set by configure-private-submodules.sh isn't visible to the fetch
# process running under the submodule's own .git/config. Symptom:
#   fatal: remote error: upload-pack: not our ref <sha>
#   fatal: Fetched in submodule path '...', but it did not contain <sha>.
# Env-var config entries propagate to every child git process, so the
# submodule's fetch is rewritten too.
#
# When origin is not OffchainLabs/nitro-private (public nitro, a non-OCL
# fork, or a shallow repro env) this is a plain `submodule update` — the
# rewrite logic is dead code there and we don't want to touch anything.
#
# POSIX-compatible so it runs under sh/dash/bash.

set -eu

# shellcheck source=scripts/lib-private-submodules.sh
. "$(dirname "$0")/lib-private-submodules.sh"

if [ ! -f .gitmodules ]; then
  echo "ERROR: .gitmodules not found (cwd=$(pwd))." >&2
  exit 1
fi

# rc=2 is "no such remote" (fresh repro w/o origin); treat as public-ish.
rc=0
output=$(git remote get-url origin 2>&1) || rc=$?
case "$rc" in
  0) remote_url="$output" ;;
  2) remote_url="" ;;
  *)
    echo "ERROR: git remote get-url origin failed (rc=$rc): $output" >&2
    exit 1
    ;;
esac

is_nitro_private=0
if [ -n "$remote_url" ]; then
  origin_owner_repo=$(normalize_github_url "$remote_url")
  [ "$origin_owner_repo" = "offchainlabs/nitro-private" ] && is_nitro_private=1
fi

if [ "$is_nitro_private" != "1" ] || [ ! -f .private-submodules.conf ]; then
  git submodule sync --recursive
  git submodule update --init --recursive
  exit 0
fi

# prefers_ssh=1 → collapse both HTTPS and SSH public URL forms onto the
# SSH-private base key (multi-valued) so a dev with a global HTTPS→SSH
# rewrite keeps SSH auth; prefers_ssh=0 → today's HTTPS-private routing.
# See configure-private-submodules.sh and the lib for rationale. The
# explicit default matches the lib's own initial assignment.
prefers_ssh=0
detect_prefers_ssh

# Build GIT_CONFIG_COUNT / GIT_CONFIG_KEY_N / GIT_CONFIG_VALUE_N env pairs
# for each conf entry. Positional parameters double as our accumulator —
# POSIX sh lacks arrays.
#
# With GIT_CONFIG_*, the same key appearing at multiple indices is
# additive (multi-valued), not last-write-wins — verified against
# git 2.30+. That's what makes the prefers_ssh path below work: the
# SSH-private base key collects both HTTPS and SSH public forms as
# separate insteadOf values.
idx=0
set --

while IFS= read -r repo || [ -n "$repo" ]; do
  repo=${repo#"${repo%%[![:space:]]*}"}
  repo=${repo%"${repo##*[![:space:]]}"}
  case "$repo" in ''|'#'*) continue;; esac
  case "$repo" in
    *[!A-Za-z0-9._-]*|[!A-Za-z0-9]*|*-private)
      echo "ERROR: invalid entry in .private-submodules.conf: '$repo'" >&2
      exit 1
      ;;
  esac

  https_val="https://github.com/OffchainLabs/${repo}.git"
  ssh_val="git@github.com:OffchainLabs/${repo}.git"
  ssh_key="url.git@github.com:OffchainLabs/${repo}-private.git.insteadof"
  # Under prefers_ssh=1 the HTTPS-public URL is routed through the
  # SSH-private base key (so the SSH target wins longest-match over the
  # global HTTPS→SSH rewrite); under prefers_ssh=0 it stays on its own
  # HTTPS-private base key.
  if [ "$prefers_ssh" = "1" ]; then
    https_key=$ssh_key
  else
    https_key="url.https://github.com/OffchainLabs/${repo}-private.git.insteadof"
  fi

  set -- "$@" "GIT_CONFIG_KEY_${idx}=${https_key}" "GIT_CONFIG_VALUE_${idx}=${https_val}"
  idx=$((idx + 1))
  set -- "$@" "GIT_CONFIG_KEY_${idx}=${ssh_key}" "GIT_CONFIG_VALUE_${idx}=${ssh_val}"
  idx=$((idx + 1))
done < .private-submodules.conf

if [ "$idx" -eq 0 ]; then
  # Empty conf (or only comments): nothing to inject, just delegate.
  git submodule sync --recursive
  git submodule update --init --recursive
  exit 0
fi

set -- "$@" "GIT_CONFIG_COUNT=${idx}"

# `env` rather than exporting so the rewrites are scoped to these two
# commands and don't linger in the caller's shell.
env "$@" git submodule sync --recursive
env "$@" git submodule update --init --recursive
