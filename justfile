build:
    make build
    make build-replay-env

espresso-tests: build
    gotestsum --format standard-verbose --packages="\$packages" -- -v -timeout 15m -p 1 ./system_tests/... -run 'TestEspressoE2E'
