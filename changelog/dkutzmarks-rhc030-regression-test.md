### Internal
- Add a regression test ensuring the Stylus wasm parser rejects oversized local
  declarations before materializing them, with a bounded peak-allocation check.
