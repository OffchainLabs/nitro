#!/bin/sh
# Print per-submodule routing (path, pin, URL, class).
#
# Local-dev diagnostic, invoked via `make check-submodules`. CI enforcement
# of submodule pins lives in .github/workflows/submodule-pin-check.yml, not
# here; this script is intentionally exit-0-on-bad-classes so a human can
# read the CLASS column.
#
# With --strict (used by the parent-scope pre-push hook), the script still
# prints the same table but exits non-zero if any routed submodule is in a
# class that indicates broken local routing (OVERRIDE-NOT-APPLIED /
# STRAY-PRIVATE-REWRITE) or if any pre-push hook is missing. That gives us
# a local block for the "I swapped a routed submodule for its public
# counterpart" class of mistake, which the URL-based pre-push check cannot
# see — the parent push is heading to nitro-private either way.
#
# KEEP THE CONF PARSER IN SYNC with scripts/configure-private-submodules.sh,
# .github/actions/init-submodules/action.yml, and the intentionally-inline
# copy in .github/workflows/submodule-pin-check.yml (inline because it runs
# under pull_request_target and must not execute HEAD code).

set -eu

# shellcheck source=scripts/lib-private-submodules.sh
# shellcheck disable=SC1091  # CI runs shellcheck without -x, so it cannot follow the source.
. "$(dirname "$0")/lib-private-submodules.sh"

strict=0
for arg in "$@"; do
  case "$arg" in
    --strict) strict=1 ;;
    -h|--help)
      printf 'Usage: %s [--strict]\n' "$(basename "$0")"
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument '$arg'" >&2
      exit 2
      ;;
  esac
done

if [ ! -f .gitmodules ]; then
  echo "ERROR: .gitmodules not found; run from the repo root." >&2
  exit 1
fi

# Accumulated in --strict mode; consulted at the very end of the script.
# A single flag would be enough to change the exit code, but keeping the
# list of offending paths lets the end-of-run message tell the dev exactly
# which submodules to look at.
strict_failures=""

# rc=1 is "no matches" (expected). Anything else (corrupt config, perms)
# must surface, otherwise the WARNING below blames the user for an
# unrelated issue.
rc=0
rewrites=$(git config --get-regexp '^url\..*\.insteadof$') || rc=$?
if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
  echo "ERROR: git config read failed (rc=$rc) while scanning insteadOf rules" >&2
  exit 1
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
  origin_owner_repo=$(normalize_github_url "$remote_url")
  if [ "$origin_owner_repo" = "offchainlabs/nitro-private" ]; then
    is_private_fork=1
  fi
fi

# Drift detection below uses this to know whether to expect the
# HTTPS-keyed rule (prefers_ssh=0) or both public URL forms on the
# SSH-private base key (prefers_ssh=1). See the lib for rationale. The
# explicit default matches the lib's own initial assignment.
prefers_ssh=0
detect_prefers_ssh

# Gated on prefers_ssh=0: an origin-SSH clone or a PREFER_SSH=1 override
# is a legitimate signal on its own, so nagging about a missing global
# HTTPS→SSH insteadOf rule there would be actively misleading.
if [ "$prefers_ssh" = "0" ] \
  && ! printf '%s\n' "$rewrites" | awk '$2 ~ /^https?:\/\/github\.com\/?$/' | grep -q .; then
  echo "WARNING: no https://github.com/ -> SSH rewrite found in git config." >&2
  echo "WARNING: submodule fetches may prompt for HTTPS credentials. Consider:" >&2
  echo "  git config --global url.git@github.com:.insteadOf https://github.com/" >&2
  echo "  (or re-clone with SSH / set PREFER_SSH=1)" >&2
fi

# has_value <values-string-with-newlines> <expected>
# Returns 0 if any line of <values-string> equals <expected>. Used by the
# drift checks below to verify both public URL forms land as multi-valued
# insteadOf entries on the SSH-private base key when prefers_ssh=1.
has_value() {
  _values=$1
  _expected=$2
  while IFS= read -r _v; do
    [ "$_v" = "$_expected" ] && return 0
  done <<EOF
$_values
EOF
  return 1
}

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

# Collected while iterating; emitted after the table so table columns stay
# stable/greppable. Parent hook is checked once up-front.
hook_warnings=""
sentinel='# nitro-private: pre-push-private-check'
if [ "$is_private_fork" = "1" ]; then
  rc=0
  parent_git_dir=$(git rev-parse --git-dir) || rc=$?
  if [ "$rc" -ne 0 ]; then
    hook_warnings="${hook_warnings}ERROR: could not resolve parent git-dir; re-run 'make init-submodules'.
"
  else
    parent_hook="${parent_git_dir}/hooks/pre-push"
    if [ ! -f "$parent_hook" ] || ! grep -qF "$sentinel" "$parent_hook" 2>/dev/null; then
      hook_warnings="${hook_warnings}HOOK-MISSING  (parent pre-push)  ${parent_hook}
"
    fi
  fi
fi

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

  # Precompute the -private URL forms once per iteration for the two
  # drift checks below (override + no-override), which would otherwise
  # reconstruct them separately. Kept in sync with
  # scripts/configure-private-submodules.sh — if these diverge,
  # OVERRIDE-NOT-APPLIED / STRAY-PRIVATE-REWRITE stop firing when they
  # should.
  if [ "$class" = "override" ] || [ "$class" = "no-override" ]; then
    prefix=${url%"OffchainLabs/${repo}"*}
    suffix=${url#"${prefix}OffchainLabs/${repo}"}
    case "$suffix" in
      .git*) tail=${suffix#.git} ;;
      *)     tail=$suffix ;;
    esac
    private_url="${prefix}OffchainLabs/${repo}-private.git${tail}"
    # The SSH-form rule guards against a manually-added SSH remote
    # (git@github.com:OffchainLabs/<repo>.git) bypassing the rewrite.
    ssh_public="git@github.com:OffchainLabs/${repo}.git"
    ssh_private="git@github.com:OffchainLabs/${repo}-private.git"
  fi

  # Drift check for routed submodules. Parent scope matters for
  # `submodule update --init` (that's what the rewrite hits to route
  # clones to -private); submodule scope + pushurl matter once the
  # submodule is initialized, because in-tree git operations don't see
  # the parent's local config and would otherwise push to the public
  # remote.
  if [ "$class" = "override" ] && [ "$is_private_fork" = "1" ]; then
    # Expected pushurl matches what configure-private-submodules.sh
    # writes: SSH-private when the dev prefers SSH globally, otherwise
    # HTTPS-private.
    if [ "$prefers_ssh" = "1" ]; then
      expected_pushurl=$ssh_private
    else
      expected_pushurl=$private_url
    fi

    # --get-all rather than --get: under prefers_ssh=1, the SSH-private
    # base key carries BOTH public URL forms as multi-valued insteadOf
    # entries; --get would only surface the last and mislabel the row.
    rc=0
    ssh_vals=$(git config --get-all "url.${ssh_private}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${ssh_private}'" >&2
      exit 1
    fi
    rc=0
    https_vals=$(git config --get-all "url.${private_url}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${private_url}'" >&2
      exit 1
    fi

    if [ "$prefers_ssh" = "1" ]; then
      # SSH-private base key must carry both public forms; HTTPS base key
      # must be empty (otherwise the HTTPS-keyed rule would shadow the
      # HTTPS→SSH global rule again — exactly the bug this branch fixes).
      if ! has_value "$ssh_vals" "$url" \
        || ! has_value "$ssh_vals" "$ssh_public" \
        || [ -n "$https_vals" ]; then
        class="OVERRIDE-NOT-APPLIED"
      fi
    else
      if ! has_value "$https_vals" "$url" \
        || ! has_value "$ssh_vals" "$ssh_public"; then
        class="OVERRIDE-NOT-APPLIED"
      fi
    fi

    # Submodule-scope checks only apply once the submodule is
    # initialized. Uninitialized is fine: the parent rules above ensure
    # the first `submodule update --init` still clones from -private.
    if [ "$class" = "override" ] && [ -e "$path/.git" ] \
      && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
      rc=0
      sub_ssh_vals=$(git -C "$path" config --get-all "url.${ssh_private}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${ssh_private}'" >&2
        exit 1
      fi
      rc=0
      sub_https_vals=$(git -C "$path" config --get-all "url.${private_url}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${private_url}'" >&2
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
      if [ "$prefers_ssh" = "1" ]; then
        if ! has_value "$sub_ssh_vals" "$url" \
          || ! has_value "$sub_ssh_vals" "$ssh_public" \
          || [ -n "$sub_https_vals" ] \
          || [ "${sub_pushurl:-}" != "$expected_pushurl" ]; then
          class="OVERRIDE-NOT-APPLIED"
        fi
      else
        if ! has_value "$sub_https_vals" "$url" \
          || ! has_value "$sub_ssh_vals" "$ssh_public" \
          || [ "${sub_pushurl:-}" != "$expected_pushurl" ]; then
          class="OVERRIDE-NOT-APPLIED"
        fi
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
    # --get-all so a stale multi-valued insteadOf (written by a prior
    # prefers_ssh=1 run and since removed from conf) is still seen. A
    # single --get would only return the last value and miss stray
    # entries sitting earlier in the list.
    rc=0
    https_vals=$(git config --get-all "url.${private_url}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${private_url}'" >&2
      exit 1
    fi
    rc=0
    ssh_vals=$(git config --get-all "url.${ssh_private}.insteadof") || rc=$?
    if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
      echo "ERROR: git config read failed (rc=$rc) querying insteadof for '${ssh_private}'" >&2
      exit 1
    fi
    if has_value "$https_vals" "$url" \
      || has_value "$ssh_vals" "$ssh_public" \
      || has_value "$ssh_vals" "$url"; then
      class="STRAY-PRIVATE-REWRITE"
    fi

    if [ "$class" = "no-override" ] && [ -e "$path/.git" ] \
      && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
      rc=0
      sub_https_vals=$(git -C "$path" config --get-all "url.${private_url}.insteadof") || rc=$?
      if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
        echo "ERROR: git config read failed (rc=$rc) in '${path}' querying insteadof for '${private_url}'" >&2
        exit 1
      fi
      rc=0
      sub_ssh_vals=$(git -C "$path" config --get-all "url.${ssh_private}.insteadof") || rc=$?
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
      if has_value "$sub_https_vals" "$url" \
        || has_value "$sub_ssh_vals" "$ssh_public" \
        || has_value "$sub_ssh_vals" "$url" \
        || [ "${sub_pushurl:-}" = "$private_url" ] \
        || [ "${sub_pushurl:-}" = "$ssh_private" ]; then
        class="STRAY-PRIVATE-REWRITE"
      fi
    fi
  fi

  # Submodule hook check. Only meaningful for initialized submodules that
  # should be routed private — otherwise there's nothing for the hook to
  # guard and its absence isn't a problem.
  if [ "$class" = "override" ] && [ "$is_private_fork" = "1" ] \
    && [ -e "$path/.git" ] \
    && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
    rc=0
    sub_git_dir=$(git -C "$path" rev-parse --git-dir) || rc=$?
    if [ "$rc" -eq 0 ]; then
      case "$sub_git_dir" in
        /*) abs_git_dir=$sub_git_dir ;;
        *)  abs_git_dir="$(pwd)/$path/$sub_git_dir" ;;
      esac
      sub_hook="${abs_git_dir}/hooks/pre-push"
      if [ ! -f "$sub_hook" ] || ! grep -qF "$sentinel" "$sub_hook" 2>/dev/null; then
        hook_warnings="${hook_warnings}HOOK-MISSING  ${path}  ${sub_hook}
"
        strict_failures="${strict_failures}  ${path}: pre-push hook missing (${sub_hook})
"
      fi
    fi
  fi

  # Accumulate strict failures for the routing classes. "no-gitlink" and
  # "external" are out of scope — nothing in this repo's machinery
  # governs them. "no-override" + "URL-IS-PRIVATE" are fine-by-design.
  case "$class" in
    OVERRIDE-NOT-APPLIED|STRAY-PRIVATE-REWRITE)
      strict_failures="${strict_failures}  ${path}: CLASS=${class}
"
      ;;
  esac

  printf '%-50s %-12s %-45s %s\n' "$path" "$short" "$owner_repo" "$class"
done <<EOF
$keys
EOF

if [ -n "$hook_warnings" ]; then
  printf '\n'
  printf 'pre-push guard hook status:\n'
  printf '%s' "$hook_warnings"
  printf "Re-run 'make init-submodules' to reinstall. Bypass knobs live in scripts/pre-push-private-check.sh.\n"
fi

if [ "$strict" = "1" ] && [ -n "$strict_failures" ]; then
  printf '\n' >&2
  printf 'ERROR: strict routing check failed. Offenders:\n' >&2
  printf '%s' "$strict_failures" >&2
  printf "Re-run 'make init-submodules' to repair. See docs/private-submodules.md for context.\n" >&2
  exit 1
fi
