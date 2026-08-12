#!/usr/bin/env bash
# Selects the build provenance for CI image builds. Meant to be sourced by the
# CodeBuild buildspecs; sets and exports:
#   NITRO_TAG      tag pointing at HEAD ("" when untagged)
#   NITRO_BRANCH   branch that triggered an untagged build ("" for tagged builds)
#   NITRO_COMMIT   short commit SHA of HEAD ("latest" when unavailable)
#   IMAGE_TAG      hyphenated tag for Docker artifacts, which cannot contain
#                  the "+" separating the commit in the binary's revision
# Tags with a canonical SemVer release core must follow Nitro's stable/dev/rc
# convention. Tags without such a core are opaque Docker tag inputs.

nitro_ci_select_version() {
  # A tag with a canonical SemVer release core is an attempted release even if
  # its suffix is malformed. Enforce Nitro's convention so typos cannot bypass
  # validation by becoming non-SemVer. Ordering remains standard SemVer in Go.
  local number='(0|[1-9][0-9]*)'
  local release_core="^v${number}\.${number}\.${number}([-+].*)?\$"
  local positive='[1-9][0-9]*'
  local nitro_release="^v${number}\.${number}\.${number}(-(dev|rc)\.${positive}(\.private\.${positive})?)?\$"

  NITRO_COMMIT=$(git rev-parse --short=7 HEAD 2>/dev/null || true)
  if [ -z "$NITRO_COMMIT" ]; then
    NITRO_COMMIT="latest"
  fi
  NITRO_BRANCH=""
  # Prefer the highest tag pointing at HEAD, a stable release before its own
  # prereleases (the temporary "_" suffix makes it sort after them).
  NITRO_TAG=$(git tag --points-at HEAD | sed '/-/!s/$/_/' | sort -rV | sed 's/_$//' | head -n 1)

  if [ -n "$NITRO_TAG" ] && [[ $NITRO_TAG =~ $release_core ]] && ! [[ $NITRO_TAG =~ $nitro_release ]]; then
    echo "ERROR: SemVer-like tag '${NITRO_TAG}' does not follow the Nitro release convention; expected vMAJOR.MINOR.PATCH with optional -dev.N, -dev.N.private.N, -rc.N, or -rc.N.private.N" >&2
    return 1
  elif [ -z "$NITRO_TAG" ]; then
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

  # Tagged builds preserve the tag exactly so the binary and image have the
  # same identity. Untagged builds sanitize branch separators such as "/".
  local docker_ref=$NITRO_TAG
  if [ -z "$docker_ref" ]; then
    docker_ref=$(printf '%s' "$NITRO_BRANCH" | sed 's/[^A-Za-z0-9_.-]/-/g')
  fi
  IMAGE_TAG="${docker_ref}-${NITRO_COMMIT}"

  # Docker tags are at most 128 ASCII characters, must start with an
  # alphanumeric or underscore, and otherwise allow only alphanumerics,
  # underscores, periods, and dashes.
  local docker_tag_pattern='^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$'
  if ! [[ $IMAGE_TAG =~ $docker_tag_pattern ]]; then
    if [ -n "$NITRO_TAG" ]; then
      echo "ERROR: Git tag '${NITRO_TAG}' produces invalid Docker image tag '${IMAGE_TAG}'" >&2
    else
      echo "ERROR: branch '${NITRO_BRANCH}' produces invalid Docker image tag '${IMAGE_TAG}'" >&2
    fi
    return 1
  fi

  export NITRO_TAG NITRO_BRANCH NITRO_COMMIT IMAGE_TAG
  echo "Selected nitro version: tag='${NITRO_TAG}' branch='${NITRO_BRANCH}' commit='${NITRO_COMMIT}' docker tag='${IMAGE_TAG}'"
}

nitro_ci_select_version
