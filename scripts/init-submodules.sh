#!/bin/sh
# Rewrite OffchainLabs/* submodule URLs to their -private variants for
# repos listed in .private-submodules.conf, optionally embed GITHUB_TOKEN
# into github.com URLs for auth, then sync + update submodules.
#
# The token is baked into the URL (not a credential helper) so it's
# naturally inherited by the git subprocesses that clone/fetch inside
# each submodule — no GIT_CONFIG_PARAMETERS propagation or helper-chain
# fighting required. .gitmodules is restored on exit.

set -eu

if [ ! -f .gitmodules ]; then
  echo "ERROR: .gitmodules not found (cwd=$(pwd))." >&2
  exit 1
fi

backup=$(mktemp)
cp .gitmodules "$backup"
trap 'cp "$backup" .gitmodules; rm -f "$backup"' EXIT INT TERM

edit() {
  sed "$@" .gitmodules > .gitmodules.tmp && mv .gitmodules.tmp .gitmodules
}

if [ -f .private-submodules.conf ]; then
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
    edit "s|OffchainLabs/${repo}\.git|OffchainLabs/${repo}-private.git|g"
  done < .private-submodules.conf
fi

if [ -n "${GITHUB_TOKEN:-}" ]; then
  edit "s|https://github.com/|https://x-access-token:${GITHUB_TOKEN}@github.com/|g"
  edit "s|git@github.com:|https://x-access-token:${GITHUB_TOKEN}@github.com/|g"
fi

git submodule sync --recursive
git submodule update --init --recursive
