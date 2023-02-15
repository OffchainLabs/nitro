# Challenge Protocol V2

## Generating Contract Bindings

* Install nodejs and npm
* Install yarn with `npm i -g yarn`
* In the `contracts/` directory, run `yarn install` then `yarn --cwd contracts build`
* In the **top-level directory**, run `go run ./solgen/main.go`
* You should now have Go bindings inside of `solgen/go`
