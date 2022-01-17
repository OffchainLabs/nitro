FROM debian:bullseye-20211220

RUN apt-get update && apt-get install -y \
	# these must be pinned to a version as they build the actual binaries
	build-essential=12.9 clang=1:11.0-51+nmu5 lld=1:11.0-51+nmu5 wabt=1.0.20-1 \
	# these can change versions without affecting the build output
	curl nodejs npm

RUN npm install -g yarn
RUN curl -L https://golang.org/dl/go1.17.linux-amd64.tar.gz | tar -C /usr/local -xzf -
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain 1.58.0 --target x86_64-unknown-linux-gnu wasm32-unknown-unknown wasm32-wasi
RUN . ~/.cargo/env && cargo install cbindgen --version =0.20.0

WORKDIR /usr/src/nitro
COPY . /usr/src/nitro
RUN . ~/.cargo/env && PATH="$PATH:/usr/local/go/bin" make build-replay-env
RUN sha256sum arbitrator/target/env/lib/*.wasm
