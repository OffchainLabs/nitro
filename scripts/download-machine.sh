#!/usr/bin/env bash
set -euo pipefail

tag="$1"
module_root="$2"
repo="${3:-OffchainLabs/nitro}"

mkdir "$module_root"
ln -sfT "$module_root" latest
cd "$module_root"
echo "$module_root" > module-root.txt

if [ "$repo" = "OffchainLabs/nitro" ]; then
	url_base="https://github.com/$repo/releases/download/$tag"
	wget "$url_base/machine.v2.wavm.br"

	status_code="$(curl -LI "$url_base/replay.wasm" -so /dev/null -w '%{http_code}')"
	if [ "$status_code" -ne 404 ]; then
		wget "$url_base/replay.wasm"
	fi
else
	token="$(cat "${GH_TOKEN_FILE:-/run/secrets/gh_token}" 2>/dev/null | tr -d '[:space:]' || true)"
	if [ -z "$token" ]; then
		echo "ERROR: $repo requires a GitHub token; mount one as a BuildKit secret with id=gh_token (e.g., --mount=type=secret,id=gh_token,required=false)" >&2
		exit 1
	fi
	api="https://api.github.com/repos/$repo/releases"

	release_json="$(curl -fsSL --retry 3 \
		-H "Authorization: Bearer $token" \
		-H "Accept: application/vnd.github+json" \
		"$api/tags/$tag")"

	asset_id_for() {
		printf '%s' "$release_json" | jq -r --arg n "$1" '(.assets // [])[] | select(.name==$n) | .id' | head -n1
	}

	download_asset() {
		local name="$1" asset_id="$2"
		curl -fsSL --retry 3 \
			-H "Authorization: Bearer $token" \
			-H "Accept: application/octet-stream" \
			-o "$name" \
			"$api/assets/$asset_id"
	}

	machine_id="$(asset_id_for machine.v2.wavm.br)"
	if [ -z "$machine_id" ]; then
		echo "ERROR: machine.v2.wavm.br not found in $repo release $tag" >&2
		exit 1
	fi
	download_asset machine.v2.wavm.br "$machine_id"

	replay_id="$(asset_id_for replay.wasm)"
	if [ -n "$replay_id" ]; then
		download_asset replay.wasm "$replay_id"
	fi
fi
