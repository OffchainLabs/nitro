#!/usr/bin/env bash
# Smoke tests for scripts/ci-version-select.sh: tag selection and Docker-safe
# naming, plus the branch, commit, and detached fallbacks used when untagged.
# Creates throwaway git repositories; needs no network.
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
    if [ "${MOCK_EMPTY_COMMIT:-}" = "1" ]; then
      # shellcheck disable=SC2329  # Invoked indirectly by the sourced version selector.
      git() {
        if [ "${1:-}" = "rev-parse" ]; then
          return 0
        fi
        command git "$@"
      }
    fi
    # shellcheck source=/dev/null
    . "$selector" >/dev/null || exit 1
    # shellcheck disable=SC2153  # IMAGE_TAG is assigned by the sourced version selector.
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

expect_tag_reject() {
  local name="$1" dir="$2" tag="$3" image_tag="$4" output
  if output=$(run_selector "$dir" 2>&1); then
    echo "FAIL: $name: selector accepted tag '$tag' as '$output'" >&2
    failures=$((failures + 1))
  elif [[ $output == *"Git tag '$tag'"* && $output == *"Docker image tag '$image_tag'"* ]]; then
    echo "PASS: $name"
  else
    echo "FAIL: $name: unexpected error '$output'" >&2
    failures=$((failures + 1))
  fi
}

expect_convention_reject() {
  local name="$1" dir="$2" tag="$3" output
  if output=$(run_selector "$dir" 2>&1); then
    echo "FAIL: $name: selector accepted tag '$tag' as '$output'" >&2
    failures=$((failures + 1))
  elif [[ $output == *"SemVer-like tag '$tag'"* && $output == *"Nitro release convention"* ]]; then
    echo "PASS: $name"
  else
    echo "FAIL: $name: unexpected error '$output'" >&2
    failures=$((failures + 1))
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

while read -r tag; do
  git -C "$repo" tag "$tag"
  expect "Nitro prerelease tag $tag" \
    "$(run_selector "$repo")" "$tag||$sha|$tag-$sha"
  git -C "$repo" tag -d "$tag" >/dev/null
done <<'EOF'
v3.12.0-dev.1
v3.12.0-dev.1.private.2
v3.12.0-rc.1.private.2
EOF

git -C "$repo" tag consensus-v61
expect "consensus tag" \
  "$(run_selector "$repo")" "consensus-v61||$sha|consensus-v61-$sha"
git -C "$repo" tag -d consensus-v61 >/dev/null

git -C "$repo" tag devnet-consensus-v3.1
expect "devnet consensus tag" \
  "$(run_selector "$repo")" "devnet-consensus-v3.1||$sha|devnet-consensus-v3.1-$sha"
git -C "$repo" tag -d devnet-consensus-v3.1 >/dev/null

while read -r tag; do
  git -C "$repo" tag "$tag"
  expect "Docker-compatible tag $tag is preserved" \
    "$(run_selector "$repo")" "$tag||$sha|$tag-$sha"
  git -C "$repo" tag -d "$tag" >/dev/null
done <<'EOF'
v3.11
3.11.3
v2024.01.10
v3.11.x-private-patches-1
arbitrary-tag
EOF

while read -r tag; do
  git -C "$repo" tag "$tag"
  expect_convention_reject "SemVer-like tag $tag is rejected" "$repo" "$tag"
  git -C "$repo" tag -d "$tag" >/dev/null
done <<'EOF'
v3.11.3-dev.0
v3.11.3-dev.01
v3.11.3-dev.1.private.0
v3.11.3-dev.1.private.01
v3.11.3-rc.0
v3.11.3-alpha.1
v3.11.3-beta.1
v3.11.3-alice.1
v3.11.3-rc.1-abcdef
v3.11.3-abcdef
v3.11.3-private.1
v3.11.3-private-patches-1
v3.11.3-patch_1
v3.11.3-rc.1.private.1.extra
v3.11.3-rc.1+build
v3.11.3+build
EOF

tag='release/foo+bar'
git -C "$repo" tag "$tag"
expect_tag_reject "Docker-incompatible tag $tag is rejected" \
  "$repo" "$tag" "$tag-$sha"
git -C "$repo" tag -d "$tag" >/dev/null

git -C "$repo" update-ref refs/tags/-candidate HEAD
expect_tag_reject "tag beginning with dash is rejected" \
  "$repo" "-candidate" "-candidate-$sha"
git -C "$repo" update-ref -d refs/tags/-candidate

printf -v long_tag '%*s' 121 ''
long_tag=${long_tag// /a}
git -C "$repo" tag "$long_tag"
expect_tag_reject "commit-suffixed tag exceeding 128 characters is rejected" \
  "$repo" "$long_tag" "$long_tag-$sha"
git -C "$repo" tag -d "$long_tag" >/dev/null

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

expect "empty commit falls back to latest" \
  "$(run_selector "$repo" MOCK_EMPTY_COMMIT=1)" "|dev|latest|dev-latest"

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
