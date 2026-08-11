#!/usr/bin/env bash
# Selects the build provenance for CI image builds. Meant to be sourced by the
# CodeBuild buildspecs; sets and exports:
#   NITRO_TAG      canonical SemVer tag pointing at HEAD ("" when untagged)
#   NITRO_BRANCH   branch that triggered an untagged build ("" for tagged builds)
#   NITRO_COMMIT   short commit SHA of HEAD
#   IMAGE_TAG      hyphenated tag for Docker artifacts, which cannot contain
#                  the "+" separating the commit in the binary's revision
# Fails before anything is built when the selected tag at HEAD is noncanonical.

nitro_ci_select_version() {
  # Canonical SemVer as defined by Go's semver.Canonical: leading "v", full
  # MAJOR.MINOR.PATCH without leading zeros, optional prerelease, no build
  # metadata. This rejects shorthand (v3.11), metadata (v3.11.3+foo), calendar
  # tags (v2024.01.10), and private-patch tag formats.
  local ident='(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
  local canonical="^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-${ident}(\.${ident})*)?\$"

  NITRO_COMMIT=$(git rev-parse --short=7 HEAD)
  NITRO_BRANCH=""
  # Prefer the highest tag pointing at HEAD, a stable release before its own
  # prereleases (the temporary "_" suffix makes it sort after them).
  NITRO_TAG=$(git tag --points-at HEAD | sed '/-/!s/$/_/' | sort -rV | sed 's/_$//' | head -n 1)

  if [ -n "$NITRO_TAG" ]; then
    if ! [[ $NITRO_TAG =~ $canonical ]]; then
      echo "ERROR: tag ${NITRO_TAG} is not a canonical SemVer release tag (vMAJOR.MINOR.PATCH with optional prerelease); refusing to build" >&2
      return 1
    fi
  else
    # Untagged: name the branch that triggered the build. Webhook builds carry
    # it in CODEBUILD_WEBHOOK_HEAD_REF; manual builds may name one in
    # CODEBUILD_SOURCE_VERSION, which can instead hold a commit hash and is
    # then ignored; otherwise fall back to the checkout's git decorations.
    NITRO_BRANCH=${CODEBUILD_WEBHOOK_HEAD_REF:-}
    NITRO_BRANCH=${NITRO_BRANCH#refs/heads/}
    if [ -z "$NITRO_BRANCH" ]; then
      NITRO_BRANCH=${CODEBUILD_SOURCE_VERSION:-}
    fi
    if [[ $NITRO_BRANCH =~ ^[0-9a-f]{7,40}$ ]]; then
      NITRO_BRANCH=""
    fi
    if [ -z "$NITRO_BRANCH" ]; then
      NITRO_BRANCH=$(git show -s --pretty=%D | sed 's/, /\n/g' | sed 's/^HEAD -> //' |
        grep -Ev '^(origin/.*|tag: .*|grafted|HEAD|master|main)$' | head -n 1 || true)
    fi
    if [ -z "$NITRO_BRANCH" ]; then
      NITRO_BRANCH="dev"
    fi
  fi

  # Docker image tags allow only [A-Za-z0-9_.-]; sanitize branch separators
  # like "/" here and nowhere else.
  local docker_ref=$NITRO_TAG
  if [ -z "$docker_ref" ]; then
    docker_ref=$(printf '%s' "$NITRO_BRANCH" | sed 's/[^A-Za-z0-9_.-]/-/g')
  fi
  IMAGE_TAG="${docker_ref}-${NITRO_COMMIT}"

  export NITRO_TAG NITRO_BRANCH NITRO_COMMIT IMAGE_TAG
  echo "Selected nitro version: tag='${NITRO_TAG}' branch='${NITRO_BRANCH}' commit='${NITRO_COMMIT}' docker tag='${IMAGE_TAG}'"
}

nitro_ci_select_version
