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

# shellcheck source=scripts/lib-private-submodules.sh
. "$(dirname "$0")/lib-private-submodules.sh"

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
origin_owner_repo=$(normalize_github_url "$remote_url")
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

# Absolute path to the repo root so the hook installer can resolve
# scripts/pre-push-private-check.sh regardless of where the hook is run
# from (parent or submodule).
repo_root=$(pwd -P)
hook_script="${repo_root}/scripts/pre-push-private-check.sh"

# detect_prefers_ssh sets `prefers_ssh` to 0 or 1 — see lib for rationale.
# When 1 we fold both public URL forms as multi-valued insteadOf on the
# SSH-private base key and aim pushurl at SSH-private, so a dev with a
# global HTTPS→SSH rewrite keeps using SSH auth without being shadowed by
# a more-specific HTTPS→HTTPS-private rule. CI has no such global rule
# → prefers_ssh=0. The explicit default here matches the lib's own
# initial assignment and makes the variable's existence obvious to
# readers (and to static analyzers).
prefers_ssh=0
detect_prefers_ssh

# install_pre_push_hook <target-git-dir>
#
# Writes a one-line shell hook that execs pre-push-private-check.sh. If
# an existing hook is present and wasn't installed by us (no sentinel),
# skip with a warning rather than clobber a developer's own hook — they
# can integrate the check manually by sourcing / calling the script.
install_pre_push_hook() {
  tgt_git_dir=$1
  tgt_hooks_dir="${tgt_git_dir}/hooks"
  tgt_hook="${tgt_hooks_dir}/pre-push"
  sentinel="# nitro-private: pre-push-private-check"

  if [ ! -x "$hook_script" ]; then
    echo "ERROR: $hook_script missing or not executable" >&2
    return 1
  fi

  mkdir -p "$tgt_hooks_dir" || {
    echo "ERROR: failed to create $tgt_hooks_dir" >&2
    return 1
  }

  if [ -e "$tgt_hook" ] && ! grep -qF "$sentinel" "$tgt_hook" 2>/dev/null; then
    echo "WARNING: $tgt_hook exists and wasn't installed by this script; leaving it alone." >&2
    echo "         Integrate '$hook_script \"\$@\"' into your hook to re-enable the private-push guard." >&2
    return 0
  fi

  # Write atomically (tmp + mv) so a killed script never leaves a
  # half-written hook that would break all future pushes.
  tmp="${tgt_hook}.tmp.$$"
  cat > "$tmp" <<EOF
#!/bin/sh
$sentinel
# Installed by scripts/configure-private-submodules.sh.
# Blocks accidental pushes to public OffchainLabs remotes from a
# -private workspace. See the target script for the bypass knobs.
exec "$hook_script" "\$@"
EOF
  chmod +x "$tmp"
  mv "$tmp" "$tgt_hook"
}

# Parent-scope hook: protects the nitro-private repo itself from an
# accidental push to OffchainLabs/nitro. Submodule-scope hooks are
# installed inside the submodule loop below (only reachable once the
# submodule has been initialized).
parent_git_dir=$(git rev-parse --git-dir)
case "$parent_git_dir" in
  /*) ;;
  *) parent_git_dir="${repo_root}/${parent_git_dir}" ;;
esac
install_pre_push_hook "$parent_git_dir"

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

  # Under prefers_ssh=1 the HTTPS-public URL is routed through the
  # SSH-private base key (so both insteadOf values and pushurl hit SSH
  # auth); under prefers_ssh=0 they stay on their own HTTPS-private base
  # key and HTTPS auth. See detect_prefers_ssh in the lib.
  if [ "$prefers_ssh" = "1" ]; then
    https_target=$ssh_private
    pushurl_target=$ssh_private
  else
    https_target=$new_url
    pushurl_target=$new_url
  fi

  if [ "$in_conf" = "1" ]; then
    # --add here serves two purposes: (1) under prefers_ssh=1 both values
    # land on the same base key (multi-valued), (2) under prefers_ssh=0
    # it behaves like a single-value set because the top-level cleanup
    # sweep already cleared the key. Parent scope is cleared there;
    # submodule scope is cleared below.
    if ! git config --add "url.${https_target}.insteadOf" "$url"; then
      echo "ERROR: Failed to configure HTTPS→-private rewrite for submodule '${name}'" >&2
      exit 1
    fi
    if ! git config --add "url.${ssh_private}.insteadOf" "$ssh_public"; then
      echo "ERROR: Failed to configure SSH→-private rewrite for submodule '${name}'" >&2
      exit 1
    fi
    # Parent-local insteadOf isn't visible to git operations run from
    # within the submodule (those use submodule-local + global config
    # only), so without a rule at submodule scope an in-tree push or
    # fetch silently targets the public remote. pushurl is
    # belt-and-suspenders: even if the rewrite is later cleared, pushes
    # still target -private rather than falling back to public.
    if [ "$submodule_initialized" = "1" ]; then
      # Wipe existing submodule-scope insteadOf values first: a previous
      # run under the opposite prefers_ssh setting may have left the base
      # key populated, and `--add` would then append to a stale entry.
      # rc=5 is "no such key" (expected).
      for base_key in "url.${new_url}.insteadOf" "url.${ssh_private}.insteadOf"; do
        rc=0
        git -C "$path" config --unset-all "$base_key" || rc=$?
        case "$rc" in 0|5) ;;
          *) echo "ERROR: Failed to clear '$base_key' in '$path' (rc=$rc)" >&2; exit 1 ;;
        esac
      done
      if ! git -C "$path" config --add "url.${https_target}.insteadOf" "$url"; then
        echo "ERROR: Failed to configure submodule-local HTTPS→-private rewrite for '${name}'" >&2
        exit 1
      fi
      if ! git -C "$path" config --add "url.${ssh_private}.insteadOf" "$ssh_public"; then
        echo "ERROR: Failed to configure submodule-local SSH→-private rewrite for '${name}'" >&2
        exit 1
      fi
      if ! git -C "$path" config remote.origin.pushurl "$pushurl_target"; then
        echo "ERROR: Failed to set remote.origin.pushurl for '${name}'" >&2
        exit 1
      fi
      # The submodule's own hooks live under its git-dir (typically
      # .git/modules/<name>/hooks/ for a normal submodule). The guard
      # runs when a dev invokes `git push` from inside the submodule —
      # where parent-scope pushurl and insteadOf rules are invisible, so
      # a cleared submodule-scope rewrite would otherwise send the push
      # to the public remote without any warning.
      rc=0
      sub_git_dir=$(git -C "$path" rev-parse --git-dir 2>&1) || rc=$?
      if [ "$rc" -ne 0 ]; then
        echo "ERROR: git -C '$path' rev-parse --git-dir failed (rc=$rc): $sub_git_dir" >&2
        exit 1
      fi
      case "$sub_git_dir" in
        /*) ;;
        *) sub_git_dir="${repo_root}/${path}/${sub_git_dir}" ;;
      esac
      install_pre_push_hook "$sub_git_dir"
    fi
  elif [ "$submodule_initialized" = "1" ]; then
    # Not in conf: drop any submodule-local rule / pushurl we may have
    # set on a previous run, so removing an entry actually stops routing
    # through the -private remote. --unset-all (not --unset) so the
    # multi-valued SSH-base key written under prefers_ssh=1 is fully
    # cleared. rc=5 is "no such key" (expected).
    for stale_key in "url.${new_url}.insteadOf" "url.${ssh_private}.insteadOf"; do
      rc=0
      git -C "$path" config --unset-all "$stale_key" || rc=$?
      case "$rc" in
        0|5) ;;
        *)
          echo "ERROR: Failed to unset '$stale_key' in '${path}' (rc=$rc)" >&2
          exit 1
          ;;
      esac
    done
    # Only clear pushurl if it still points at a -private URL form we
    # would have written (either HTTPS-private or SSH-private, depending
    # on whether prefers_ssh was set on the run that planted it). Leave
    # unrelated values (e.g., a developer's manual override) untouched.
    rc=0
    cur_pushurl=$(git -C "$path" config --get remote.origin.pushurl) || rc=$?
    case "$rc" in
      0)
        if [ "$cur_pushurl" = "$new_url" ] || [ "$cur_pushurl" = "$ssh_private" ]; then
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
