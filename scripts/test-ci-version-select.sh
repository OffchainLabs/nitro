#!/usr/bin/env bash
# Smoke tests for scripts/ci-version-select.sh: tag selection and canonical
# SemVer enforcement, plus the branch and detached fallbacks used when HEAD is
# untagged. Creates throwaway git repositories; needs no network.
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
selector="$script_dir/ci-version-select.sh"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failures=0

make_repo() {
  local dir="$1"
  git init -q -b master "$dir"
  git -C "$dir" -c user.email=test@test -c user.name=test \
    commit -q --allow-empty -m init
}

# Runs the selector in the given repo with only the given VAR=VALUE
# environment; prints "TAG|BRANCH|COMMIT|VERSION" on success.
run_selector() {
  local dir="$1"
  shift
  (
    cd "$dir"
    unset CODEBUILD_WEBHOOK_HEAD_REF CODEBUILD_SOURCE_VERSION
    local kv
    for kv in "$@"; do export "${kv?}"; done
    # shellcheck source=/dev/null
    . "$selector" >/dev/null 2>&1 || exit 1
    printf '%s|%s|%s|%s\n' "$NITRO_TAG" "$NITRO_BRANCH" "$NITRO_COMMIT" "$IMAGE_TAG"
  )
}

expect() {
  local name="$1" got="$2" want="$3"
  if [ "$got" = "$want" ]; then
    echo "PASS: $name"
  else
    echo "FAIL: $name: got '$got', want '$want'" >&2
    failures=$((failures + 1))
  fi
}

expect_reject() {
  local name="$1" dir="$2"
  if run_selector "$dir" >/dev/null; then
    echo "FAIL: $name: selector accepted a noncanonical tag" >&2
    failures=$((failures + 1))
  else
    echo "PASS: $name"
  fi
}

repo="$tmp/repo"
make_repo "$repo"
sha=$(git -C "$repo" rev-parse --short=7 HEAD)

git -C "$repo" tag v3.11.3
expect "canonical stable tag" \
  "$(run_selector "$repo")" "v3.11.3||$sha|v3.11.3-$sha"

git -C "$repo" tag -d v3.11.3 >/dev/null
git -C "$repo" tag v3.11.3-rc.1
expect "canonical prerelease tag" \
  "$(run_selector "$repo")" "v3.11.3-rc.1||$sha|v3.11.3-rc.1-$sha"

git -C "$repo" tag v3.11.3
expect "stable tag preferred over its prerelease" \
  "$(run_selector "$repo")" "v3.11.3||$sha|v3.11.3-$sha"
git -C "$repo" tag -d v3.11.3 v3.11.3-rc.1 >/dev/null

for bad in v3.11 v3.11.3+build 3.11.3 v2024.01.10 v3.11.3-patch_1; do
  git -C "$repo" tag "$bad"
  expect_reject "noncanonical tag $bad rejected" "$repo"
  git -C "$repo" tag -d "$bad" >/dev/null
done

expect "webhook head ref names the branch" \
  "$(run_selector "$repo" CODEBUILD_WEBHOOK_HEAD_REF=refs/heads/feature/foo)" \
  "|feature/foo|$sha|feature-foo-$sha"

expect "source version names the branch" \
  "$(run_selector "$repo" CODEBUILD_SOURCE_VERSION=my-branch)" \
  "|my-branch|$sha|my-branch-$sha"

expect "commit hash masquerading as branch falls back to dev" \
  "$(run_selector "$repo" CODEBUILD_SOURCE_VERSION="$(git -C "$repo" rev-parse HEAD)")" \
  "|dev|$sha|dev-$sha"

expect "master decoration falls back to dev" \
  "$(run_selector "$repo")" "|dev|$sha|dev-$sha"

git -C "$repo" checkout -q -b v9.9.9
expect "SemVer-looking branch stays untagged" \
  "$(run_selector "$repo")" "|v9.9.9|$sha|v9.9.9-$sha"

git -C "$repo" checkout -q -b some/branch
expect "checked-out branch from decorations, sanitized for docker" \
  "$(run_selector "$repo")" "|some/branch|$sha|some-branch-$sha"

git -C "$repo" checkout -q --detach master
git -C "$repo" branch -q -D v9.9.9 some/branch
expect "detached checkout falls back to dev" \
  "$(run_selector "$repo")" "|dev|$sha|dev-$sha"

if [ "$failures" -ne 0 ]; then
  echo "$failures test(s) failed" >&2
  exit 1
fi
echo "All ci-version-select tests passed"
