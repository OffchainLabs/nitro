#!/usr/bin/env bash
# Same as ./download-machine.sh but for the espresso integration.
#
# The url_base has been changed to point to the espresso integration repo such
# that it downloads the replay wasm binary for the integration instead.
#
# For this to work there needs to be a tagged github release that exported the
# wasm machine. Then, run
#
# ./download-machine-espresso.sh GIT_RELEASE_TAG WASM_MACHINE_ROOT
# ./download-machine-espresso.sh 20231211        0xb2ec17fe4ae788f2c81cd1d28242dfa47696598ea0f18cd78f64c7e2e8b75434
set -euxo pipefail

mkdir "$2"
ln -sfT "$2" latest
cd "$2"

url_base="https://github.com/EspressoSystems/nitro-espresso-integration/releases/download/$1"

# Download the module root from the release page
wget "$url_base/module-root.txt"

# Check that the module root specified matches the release
grep -q "$2" module-root.txt ||
    (echo "Module root mismatch: specified $2 != release $(cat module-root.txt)" && exit 1)

wget "$url_base/machine.wavm.br"

status_code="$(curl -LI "$url_base/replay.wasm" -so /dev/null -w '%{http_code}')"
if [ "$status_code" -ne 404 ]; then
	wget "$url_base/replay.wasm"
fi
