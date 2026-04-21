#!/bin/sh
# Sync OffchainLabs -private URL rewrites to match .private-submodules.conf
# when origin is OffchainLabs/nitro-private, or clear them otherwise.
#
# Each insteadOf is keyed on the URL as it appears in .gitmodules.
# CodeStar-connected CodeBuild injects "x-access-token:<ghs>@" into those
# URLs before this script runs; a rule keyed on the clean form would not
# match the credentialed URL (insteadOf is literal prefix).
#
# POSIX-compatible so it runs under bash (GitHub Actions), dash (CodeBuild),
# and make's default shell.
#
# KEEP THE CONF PARSER IN SYNC with the inline copies in
# scripts/check-submodules.sh, .github/actions/init-submodules/action.yml,
# and .github/workflows/submodule-pin-check.yml (which stays inline
# deliberately: pull_request_target must not run parser code from HEAD).

set -eu

CONF=".private-submodules.conf"

# Check origin before touching config so unrelated forks are a true
# no-op — we do not want to delete a developer's own OffchainLabs/*-private
# insteadOf rules on those. rc=2 is "no such remote" (fresh clones);
# anything else is a real git failure and must surface.
# 2>&1 folds stderr into the captured output so the error branch can
# include git's own diagnostic without a second invocation.
rc=0
output=$(git remote get-url origin 2>&1) || rc=$?
case "$rc" in
  0) remote_url="$output" ;;
  2)
    echo "$(basename "$0"): no 'origin' remote set; skipping private-fork rewrites." >&2
    exit 0
    ;;
  *)
    echo "ERROR: git remote get-url origin failed (rc=$rc): $output" >&2
    exit 1
    ;;
esac
# Anchoring rules out wrapper URLs like https://evil.com/github.com/....
# CodeStar-connected CodeBuild checks out via codestar-connections.<region>
# .amazonaws.com/git-http/<account>/<region>/<conn-id>/<owner>/<repo>, so
# without a branch for it this script would skip rewrites on CodeBuild.
origin_owner_repo=$(printf '%s' "$remote_url" | sed -E '
  s,^https?://([^@/]+@)?github\.com/,,
  s,^ssh://([^@]+@)?github\.com(:[0-9]+)?/,,
  s,^git@github\.com:,,
  s,^https?://([^@/]+@)?codestar-connections\.[^/]+\.amazonaws\.com/git-http/[^/]+/[^/]+/[^/]+/,,
  s,\.git$,,
  s,/+$,,
' | tr '[:upper:]' '[:lower:]')
if [ "$origin_owner_repo" != "offchainlabs/nitro-private" ]; then
  echo "$(basename "$0"): origin is '${remote_url}', not OffchainLabs/nitro-private; skipping private-fork rewrites." >&2
  exit 0
fi

if [ ! -f "$CONF" ]; then
  echo "ERROR: $CONF not found (cwd=$(pwd)). Run this script from the repo root." >&2
  exit 1
fi
if [ ! -f .gitmodules ]; then
  echo "ERROR: .gitmodules not found (cwd=$(pwd)). Run this script from the repo root." >&2
  exit 1
fi

# Clear only the insteadof keys we manage, not the whole url.<base> section
# (which may hold unrelated keys like pushInsteadOf). Input-file checks
# above run first: clearing rules then aborting would let the next
# `git submodule update` silently fall back to the public URL.
# rc=1 is "no matches" (expected); anything else must surface.
rc=0
existing=$(git config --get-regexp '^url\..*OffchainLabs/.+-private\.git.*\.insteadof$') || rc=$?
if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then
  echo "ERROR: git config read failed (rc=$rc) while scanning insteadof rules" >&2
  exit 1
fi
while IFS= read -r line; do
  [ -z "$line" ] && continue
  key=${line%% *}
  # rc=5 is "no such section/key" — expected if a concurrent run already
  # removed it. Anything else is a real git-config error.
  rc=0
  git config --unset-all "$key" || rc=$?
  case "$rc" in
    0|5) ;;
    *)
      echo "ERROR: Failed to unset stale rewrite '$key' (rc=$rc)" >&2
      exit 1
      ;;
  esac
done <<EOF
$existing
EOF

conf_list=" "
while IFS= read -r repo || [ -n "$repo" ]; do
  repo=${repo#"${repo%%[![:space:]]*}"}
  repo=${repo%"${repo##*[![:space:]]}"}
  case "$repo" in ''|'#'*) continue;; esac
  case "$repo" in
    *[!A-Za-z0-9._-]*|[!A-Za-z0-9]*)
      echo "ERROR: Invalid entry in $CONF: '${repo}'" >&2
      exit 1
      ;;
    *-private)
      echo "ERROR: $CONF entries must not include the -private suffix: '${repo}'" >&2
      exit 1
      ;;
  esac
  conf_list="${conf_list}${repo} "
done < "$CONF"

if ! keys=$(git config --file .gitmodules --name-only --get-regexp '^submodule\..+\.url$'); then
  echo "ERROR: Failed to enumerate submodules from .gitmodules" >&2
  exit 1
fi
while IFS= read -r key; do
  [ -z "$key" ] && continue
  name=${key#submodule.}
  name=${name%.url}
  # A corrupted .gitmodules with a .path but no .url would otherwise leave
  # $url="" and silently skip private rewrites for that entry.
  rc=0
  url=$(git config --file .gitmodules --get "submodule.${name}.url") || rc=$?
  if [ "$rc" -ne 0 ] || [ -z "$url" ]; then
    echo "ERROR: Failed to read submodule.${name}.url from .gitmodules (rc=$rc)" >&2
    exit 1
  fi
  rc=0
  path=$(git config --file .gitmodules --get "submodule.${name}.path") || rc=$?
  if [ "$rc" -ne 0 ] || [ -z "$path" ]; then
    echo "ERROR: Failed to read submodule.${name}.path from .gitmodules (rc=$rc)" >&2
    exit 1
  fi

  case "$url" in
    *OffchainLabs/*) ;;
    *) continue ;;
  esac

  after=${url##*OffchainLabs/}
  repo_seg=${after%%/*}
  repo=${repo_seg%.git}
  case "$repo" in
    *-private) continue ;;
  esac

  prefix=${url%"OffchainLabs/$repo_seg"*}
  suffix=${url#"${prefix}OffchainLabs/$repo_seg"}
  case "$suffix" in
    .git*) tail=${suffix#.git} ;;
    *)     tail=$suffix ;;
  esac
  new_url="${prefix}OffchainLabs/${repo}-private.git${tail}"

  # Cover the bare SSH form too. The insteadOf rule above only matches
  # URLs starting with the .gitmodules prefix (HTTPS in practice), so a
  # developer or tool that adds an SSH remote — e.g., `git remote add
  # upstream git@github.com:OffchainLabs/<repo>.git` — would otherwise
  # bypass the rewrite entirely. SSH URLs don't get the CodeBuild
  # x-access-token injection the HTTPS prefix/suffix logic handles, so
  # the pair is fixed.
  ssh_public="git@github.com:OffchainLabs/${repo}.git"
  ssh_private="git@github.com:OffchainLabs/${repo}-private.git"

  # Probe once; both the apply and cleanup branches below need to know
  # whether the submodule has a working git dir (file-gitlink or dir).
  submodule_initialized=0
  if [ -e "$path/.git" ] && git -C "$path" rev-parse --git-dir >/dev/null 2>&1; then
    submodule_initialized=1
  fi

  case "$conf_list" in
    *" $repo "*) in_conf=1 ;;
    *)           in_conf=0 ;;
  esac

  if [ "$in_conf" = "1" ]; then
    if ! git config "url.${new_url}.insteadOf" "$url"; then
      echo "ERROR: Failed to configure URL rewrite for submodule '${name}'" >&2
      exit 1
    fi
    if ! git config "url.${ssh_private}.insteadOf" "$ssh_public"; then
      echo "ERROR: Failed to configure SSH URL rewrite for submodule '${name}'" >&2
      exit 1
    fi
    # Parent-local insteadOf isn't visible to git operations run from
    # within the submodule (those use submodule-local + global config
    # only), so without a rule at submodule scope an in-tree push or
    # fetch silently targets the public remote. pushurl is
    # belt-and-suspenders: even if the rewrite is later cleared, pushes
    # still target -private rather than falling back to public.
    if [ "$submodule_initialized" = "1" ]; then
      if ! git -C "$path" config "url.${new_url}.insteadOf" "$url"; then
        echo "ERROR: Failed to configure submodule-local URL rewrite for '${name}'" >&2
        exit 1
      fi
      if ! git -C "$path" config "url.${ssh_private}.insteadOf" "$ssh_public"; then
        echo "ERROR: Failed to configure submodule-local SSH URL rewrite for '${name}'" >&2
        exit 1
      fi
      if ! git -C "$path" config remote.origin.pushurl "$new_url"; then
        echo "ERROR: Failed to set remote.origin.pushurl for '${name}'" >&2
        exit 1
      fi
    fi
  elif [ "$submodule_initialized" = "1" ]; then
    # Not in conf: drop any submodule-local rule / pushurl we may have
    # set on a previous run, so removing an entry actually stops routing
    # through the -private remote. rc=5 is "no such key" (expected if
    # the key was never set or already unset).
    for stale_key in "url.${new_url}.insteadOf" "url.${ssh_private}.insteadOf"; do
      rc=0
      git -C "$path" config --unset "$stale_key" || rc=$?
      case "$rc" in
        0|5) ;;
        *)
          echo "ERROR: Failed to unset '$stale_key' in '${path}' (rc=$rc)" >&2
          exit 1
          ;;
      esac
    done
    # Only clear pushurl if it still points at the -private URL we would
    # have written; leave unrelated values (e.g., a developer's manual
    # override) untouched.
    rc=0
    cur_pushurl=$(git -C "$path" config --get remote.origin.pushurl) || rc=$?
    case "$rc" in
      0)
        if [ "$cur_pushurl" = "$new_url" ]; then
          rc2=0
          git -C "$path" config --unset remote.origin.pushurl || rc2=$?
          case "$rc2" in
            0|5) ;;
            *)
              echo "ERROR: Failed to unset remote.origin.pushurl in '${path}' (rc=$rc2)" >&2
              exit 1
              ;;
          esac
        fi
        ;;
      1|5) ;;
      *)
        echo "ERROR: git config read failed (rc=$rc) reading remote.origin.pushurl in '${path}'" >&2
        exit 1
        ;;
    esac
  fi
done <<EOF
$keys
EOF
