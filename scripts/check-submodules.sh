#!/bin/sh
# Print per-submodule routing (path, pin, URL, class).
#
# Local-dev diagnostic, invoked via `make check-submodules`. CI enforcement
# of submodule pins lives in .github/workflows/submodule-pin-check.yml, not
# here; this script is intentionally exit-0-on-bad-classes so a human can
# read the CLASS column.
#
# KEEP THE CONF PARSER IN SYNC with scripts/configure-private-submodules.sh,
# .github/actions/init-submodules/action.yml, and the intentionally-inline
# copy in .github/workflows/submodule-pin-check.yml (inline because it runs
# under pull_request_target and must not execute HEAD code).

set -eu

if [ ! -f .gitmodules ]; then
  echo "ERROR: .gitmodules not found; run from the repo root." >&2
  exit 1
fi

# rc=1 is "no matches" (expected). Anything else (corrupt config, perms)
# must surface, otherwise the WARNING below blames the user for an
# unrelated issue.
rc=0
rewrites=$(git config --get-regexp '^url\..*\.insteadof$') || rc=$?
if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
  echo "ERROR: git config read failed (rc=$rc) while scanning insteadOf rules" >&2
  exit 1
fi
if ! printf '%s\n' "$rewrites" | awk '$2 ~ /^https?:\/\/github\.com\/?$/' | grep -q .; then
  echo "WARNING: no https://github.com/ -> SSH rewrite found in git config." >&2
  echo "WARNING: submodule fetches may prompt for HTTPS credentials. Consider:" >&2
  echo "  git config --global url.git@github.com:.insteadOf https://github.com/" >&2
fi

is_private_fork=0
# rc=2 is "no such remote" (expected on fresh clones). Anything else is a
# real git failure; surface it so every row isn't silently mislabeled.
rc=0
remote_url=$(git remote get-url origin 2>/dev/null) || rc=$?
case "$rc" in
  0|2) ;;
  *)
    echo "WARNING: git remote get-url origin failed (rc=$rc); is_private_fork checks skipped." >&2
    remote_url=""
    ;;
esac
if [ -n "$remote_url" ]; then
  # Same normalization as scripts/configure-private-submodules.sh; see there
  # for the wrapper-URL (https://evil.com/github.com/...) rationale.
  origin_owner_repo=$(printf '%s' "$remote_url" | sed -E '
    s,^https?://([^@/]+@)?github\.com/,,
    s,^ssh://([^@]+@)?github\.com(:[0-9]+)?/,,
    s,^git@github\.com:,,
    s,\.git$,,
    s,/+$,,
  ' | tr '[:upper:]' '[:lower:]')
  if [ "$origin_owner_repo" = "offchainlabs/nitro-private" ]; then
    is_private_fork=1
  fi
fi

conf_list=" "
if [ -f .private-submodules.conf ]; then
  while IFS= read -r repo || [ -n "$repo" ]; do
    repo=${repo#"${repo%%[![:space:]]*}"}
    repo=${repo%"${repo##*[![:space:]]}"}
    case "$repo" in ''|'#'*) continue;; esac
    case "$repo" in
      *[!A-Za-z0-9._-]*|[!A-Za-z0-9]*)
        echo "ERROR: Invalid entry in .private-submodules.conf: '${repo}'" >&2
        exit 1
        ;;
      *-private)
        echo "ERROR: .private-submodules.conf entries must not include the -private suffix: '${repo}'" >&2
        exit 1
        ;;
    esac
    conf_list="${conf_list}${repo} "
  done < .private-submodules.conf
fi

printf '%-50s %-12s %-45s %s\n' PATH PIN URL CLASS

if ! keys=$(git config --file .gitmodules --name-only --get-regexp '^submodule\..+\.path$'); then
  echo "ERROR: failed to enumerate submodules from .gitmodules" >&2
  exit 1
fi

while IFS= read -r key; do
  [ -z "$key" ] && continue
  name=${key#submodule.}
  name=${name%.path}
  # A corrupted .gitmodules entry missing either .path or .url would leave
  # those empty; `git ls-tree HEAD -- ""` then runs against the worktree
  # root and mislabels the row as no-gitlink / external.
  rc=0
  path=$(git config --file .gitmodules --get "submodule.${name}.path") || rc=$?
  if [ "$rc" -ne 0 ] || [ -z "$path" ]; then
    echo "ERROR: Failed to read submodule.${name}.path from .gitmodules (rc=$rc)" >&2
    exit 1
  fi
  rc=0
  url=$(git config --file .gitmodules --get "submodule.${name}.url") || rc=$?
  if [ "$rc" -ne 0 ] || [ -z "$url" ]; then
    echo "ERROR: Failed to read submodule.${name}.url from .gitmodules (rc=$rc)" >&2
    exit 1
  fi

  # Split from a pipeline so a real `git ls-tree` failure (e.g., no HEAD)
  # aborts loudly instead of leaving $sha empty and mislabeling the row.
  if ! lstree=$(git ls-tree HEAD -- "$path"); then
    echo "ERROR: git ls-tree HEAD failed for '$path'" >&2
    exit 1
  fi
  sha=$(printf '%s\n' "$lstree" | awk '$2 == "commit" {print $3}')
  [ -z "$sha" ] && sha="(missing)"
  short=$(printf '%s' "$sha" | cut -c1-12)

  # Same normalization as origin above so CodeBuild-injected
  # x-access-token:... URLs classify as OffchainLabs, not external.
  owner_repo=$(printf '%s' "$url" | sed -E '
    s,^https?://([^@/]+@)?github\.com/,,
    s,^ssh://([^@]+@)?github\.com(:[0-9]+)?/,,
    s,^git@github\.com:,,
    s,\.git$,,
    s,/+$,,
  ')
  owner=${owner_repo%/*}
  repo=${owner_repo#*/}

  if [ "$sha" = "(missing)" ]; then
    class="no-gitlink"
  elif [ "$owner" = "OffchainLabs" ]; then
    stem=${repo%-private}
    if [ "$stem" != "$repo" ]; then
      class="URL-IS-PRIVATE"
    else
      case "$conf_list" in
        *" $stem "*) class="override" ;;
        *)           class="no-override" ;;
      esac
    fi
  else
    class="external"
  fi

  # Drift check: the private_url computation and rewrite checks below
  # must match scripts/configure-private-submodules.sh — keep them in
  # sync or OVERRIDE-NOT-APPLIED stops firing when it should. Parent
  # scope matters for `submodule update --init` (that's what the rewrite
  # hits to route clones to -private); submodule scope + pushurl matter
  # once the submodule is initialized, because in-tree git operations
  # don't see the parent's local config and would otherwise push to the
  # public remote.
  if [ "$class" = "override" ] && [ "$is_private_fork" = "1" ]; then
    prefix=${url%"OffchainLabs/${repo}"*}
    suffix=${url#"${prefix}OffchainLabs/${repo}"}
    case "$suffix" in
      .git*) tail=${suffix#.git} ;;
      *)     tail=$suffix ;;
    esac
    private_url="${prefix}OffchainLabs/${repo}-private.git${tail}"
    # The SSH-form rule guards against a manually-added SSH remote
    # (git@github.com:OffchainLabs/<repo>.git) bypassing the rewrite.
    # configure-private-submodules.sh sets it at both parent and
    # submodule scope; parent scope is checked immediately below,
    # submodule scope is checked once initialized further down.
    ssh_public="git@github.com:OffchainLabs/${repo}.git"
    ssh_private="git@github.com:OffchainLabs/${repo}-private.git"
    # rc=1 is "key not present" (expected). Anything else is a real
    # git-config error and we want to surface it rather than mislabel.
    rc=0
    actual=$(git config --get "url.${private_url}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${private_url}'" >&2
      exit 1
    fi
    rc=0
    ssh_actual=$(git config --get "url.${ssh_private}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${ssh_private}'" >&2
      exit 1
    fi
    if [ "$actual" != "$url" ] || [ "$ssh_actual" != "$ssh_public" ]; then
      class="OVERRIDE-NOT-APPLIED"
    fi
    # Submodule-scope checks only apply once the submodule is
    # initialized. Uninitialized is fine: the parent rules above ensure
    # the first `submodule update --init` still clones from -private.
    if [ "$class" = "override" ] && [ -e "$path/.git" ] \
      && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
      rc=0
      sub_actual=$(git -C "$path" config --get "url.${private_url}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${private_url}'" >&2
        exit 1
      fi
      rc=0
      sub_ssh_actual=$(git -C "$path" config --get "url.${ssh_private}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${ssh_private}'" >&2
        exit 1
      fi
      # rc=5 is "no such section/key" (expected when pushurl was never
      # set). Anything other than 0/1/5 is a real read error.
      rc=0
      sub_pushurl=$(git -C "$path" config --get remote.origin.pushurl) || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ] && [ "$rc" -ne 5 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying remote.origin.pushurl" >&2
        exit 1
      fi
      if [ "$sub_actual" != "$url" ] \
        || [ "$sub_ssh_actual" != "$ssh_public" ] \
        || [ "${sub_pushurl:-}" != "$private_url" ]; then
        class="OVERRIDE-NOT-APPLIED"
      fi
    fi
  fi

  # Stray-rewrite check: the inverse of OVERRIDE-NOT-APPLIED. A submodule
  # whose stem isn't in .private-submodules.conf must not be routed to the
  # -private remote. A stale rule — e.g. from an entry previously in conf,
  # or a manual `git config url....insteadof` — would otherwise silently
  # send public-intended operations to the private remote. Unlike the
  # override drift check we run this regardless of is_private_fork: the
  # invariant "not-in-conf entries are never routed private" holds on any
  # clone, and configure-private-submodules.sh exits early on public
  # origins so it cannot clean up leftover rules there.
  if [ "$class" = "no-override" ]; then
    prefix=${url%"OffchainLabs/${repo}"*}
    suffix=${url#"${prefix}OffchainLabs/${repo}"}
    case "$suffix" in
      .git*) tail=${suffix#.git} ;;
      *)     tail=$suffix ;;
    esac
    private_url="${prefix}OffchainLabs/${repo}-private.git${tail}"
    ssh_public="git@github.com:OffchainLabs/${repo}.git"
    ssh_private="git@github.com:OffchainLabs/${repo}-private.git"

    rc=0
    actual=$(git config --get "url.${private_url}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${private_url}'" >&2
      exit 1
    fi
    rc=0
    ssh_actual=$(git config --get "url.${ssh_private}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${ssh_private}'" >&2
      exit 1
    fi
    if [ "$actual" = "$url" ] || [ "$ssh_actual" = "$ssh_public" ]; then
      class="STRAY-PRIVATE-REWRITE"
    fi

    if [ "$class" = "no-override" ] && [ -e "$path/.git" ] \
      && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
      rc=0
      sub_actual=$(git -C "$path" config --get "url.${private_url}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${private_url}'" >&2
        exit 1
      fi
      rc=0
      sub_ssh_actual=$(git -C "$path" config --get "url.${ssh_private}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${ssh_private}'" >&2
        exit 1
      fi
      rc=0
      sub_pushurl=$(git -C "$path" config --get remote.origin.pushurl) || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ] && [ "$rc" -ne 5 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying remote.origin.pushurl" >&2
        exit 1
      fi
      if [ "$sub_actual" = "$url" ] \
        || [ "$sub_ssh_actual" = "$ssh_public" ] \
        || [ "${sub_pushurl:-}" = "$private_url" ]; then
        class="STRAY-PRIVATE-REWRITE"
      fi
    fi
  fi

  printf '%-50s %-12s %-45s %s\n' "$path" "$short" "$owner_repo" "$class"
done <<EOF
$keys
EOF
