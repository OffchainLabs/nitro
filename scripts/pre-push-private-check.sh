#!/bin/sh
# Pre-push guard installed by scripts/configure-private-submodules.sh in
# the parent nitro-private repo and in each submodule routed through a
# -private fork. Refuses pushes that target the public OffchainLabs/<repo>
# counterpart of anything the -private workspace is set up to route.
#
# Called with the arguments git passes to a pre-push hook:
#   $1 = remote name (e.g. 'origin')
#   $2 = remote URL  (e.g. 'git@github.com:OffchainLabs/nitro.git')
# stdin is the push refspec list; we don't consult it — the URL is enough.
#
# Bypass (two paths on purpose — the env var covers automation where
# --no-verify isn't reachable, e.g. an IDE that doesn't expose it):
#   git push --no-verify ...
#   NITRO_ALLOW_PUBLIC_PUSH=1 git push ...
#
# This script is invoked both from the parent repo and from submodules.
# It finds the containing nitro-private workspace via
# `rev-parse --show-superproject-working-tree`, then reads the parent's
# .private-submodules.conf to decide whether `$remote` is a repo it
# actively routes to -private.

set -eu

# shellcheck source=scripts/lib-private-submodules.sh
# shellcheck disable=SC1091  # CI runs shellcheck without -x, so it cannot follow the source.
. "$(dirname "$0")/lib-private-submodules.sh"

remote_url="${2:-}"
[ -n "$remote_url" ] || exit 0
[ "${NITRO_ALLOW_PUBLIC_PUSH:-}" = "1" ] && exit 0

# Locate the containing nitro-private workspace. From a submodule we walk
# up via superproject; from the parent itself toplevel returns us. The
# same path is used later by the URL-direction check and — when we are
# the parent — by the strict local-routing check below.
super=$(git rev-parse --show-superproject-working-tree 2>/dev/null || true)
at_superproject=0
if [ -z "$super" ]; then
  super=$(git rev-parse --show-toplevel 2>/dev/null || true)
  at_superproject=1
fi
[ -n "$super" ] || exit 0

rc=0
super_origin=$(git -C "$super" remote get-url origin 2>/dev/null) || rc=$?
if [ "$rc" -ne 0 ] || [ -z "$super_origin" ]; then
  exit 0
fi
super_owner_repo=$(normalize_github_url "$super_origin")
super_is_nitro_private=0
[ "$super_owner_repo" = "offchainlabs/nitro-private" ] && super_is_nitro_private=1

# Strict local-routing check. Runs when we're pushing from the parent
# nitro-private repo, *before* the URL-direction check — because that check
# only inspects the push target, not the commit payload. The check catches
# the "I deleted arbitrator/tools/wasmer and replaced it with the public
# one" class of mistake: the push target is still nitro-private (so the
# URL check would allow it) but the submodule's local config has lost
# pushurl / insteadOf / the pre-push hook, and the strict check sees those
# gaps. CI still enforces the pinned-SHA check (submodule-pin-check), but
# this gives a local block that fails fast and doesn't cost a round trip.
if [ "$at_superproject" = "1" ] && [ "$super_is_nitro_private" = "1" ]; then
  if [ -x "${super}/scripts/check-submodules.sh" ]; then
    rc=0
    ( cd "$super" && ./scripts/check-submodules.sh --strict ) || rc=$?
    if [ "$rc" -ne 0 ]; then
      printf '\npre-push blocked: local submodule routing is broken (see above).\n' >&2
      printf "Fix with 'make init-submodules', or bypass with 'git push --no-verify' /\n" >&2
      printf "NITRO_ALLOW_PUBLIC_PUSH=1 if you have already verified the change is safe.\n" >&2
      exit 1
    fi
  fi
fi

[ "$super_is_nitro_private" = "1" ] || exit 0

owner_repo=$(normalize_github_url "$remote_url")
owner=${owner_repo%%/*}
repo=${owner_repo#*/}

# Not an OffchainLabs remote — none of our business.
[ "$owner" = "offchainlabs" ] || exit 0
# Already the -private variant — fine.
case "$repo" in *-private) exit 0;; esac

# Enforce for the parent repo itself (pushing nitro-private → nitro) and
# for every entry in .private-submodules.conf. Anything else (e.g. a public
# submodule like brotli, safe-smart-account) isn't our concern even when
# owner=OffchainLabs — those aren't routed through a -private fork.
enforce=0
case "$repo" in
  nitro) enforce=1 ;;
  *)
    if [ -f "${super}/.private-submodules.conf" ]; then
      while IFS= read -r entry || [ -n "$entry" ]; do
        entry=${entry#"${entry%%[![:space:]]*}"}
        entry=${entry%"${entry##*[![:space:]]}"}
        case "$entry" in ''|'#'*) continue;; esac
        if [ "$entry" = "$repo" ]; then
          enforce=1
          break
        fi
      done < "${super}/.private-submodules.conf"
    fi
    ;;
esac
[ "$enforce" -eq 1 ] || exit 0

msg="pre-push: about to push from a nitro-private workspace to the PUBLIC repo 'OffchainLabs/${repo}'."
# Prompt only if a controlling terminal is actually openable. `-r/-w` on
# /dev/tty can pass on permissions while `open()` fails with ENXIO under
# `</dev/null` or a daemon context; detect that by trying to open the tty
# and falling back to refuse-outright if it errors.
if { exec 3>/dev/tty; } 2>/dev/null && { exec 4</dev/tty; } 2>/dev/null; then
  printf '%s\n' "$msg" >&2
  printf 'Type "yes" to push anyway, or anything else to abort: ' >&3
  answer=""
  IFS= read -r answer <&4 || true
  exec 3>&- 4<&-
  case "$answer" in
    yes|YES|Yes)
      printf 'Continuing push to %s as requested.\n' "$owner_repo" >&2
      exit 0
      ;;
    *)
      printf "Aborting push. Use 'git push --no-verify' or 'NITRO_ALLOW_PUBLIC_PUSH=1' to bypass.\n" >&2
      exit 1
      ;;
  esac
fi
printf "%s Refusing (no tty). Set NITRO_ALLOW_PUBLIC_PUSH=1 or use 'git push --no-verify' to bypass.\n" "$msg" >&2
exit 1
