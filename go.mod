module github.com/OffchainLabs/new-rollup-exploration

go 1.19

require (
	github.com/emicklei/dot v1.2.0
	github.com/ethereum/go-ethereum v1.10.21
	github.com/gorilla/websocket v1.5.0
	github.com/labstack/echo/v5 v5.0.0-20220717203827-74022662be4a
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.1
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/sys v0.0.0-20220722155257-8c9f86f7a55f // indirect
	golang.org/x/time v0.0.0-20211116232009-f0f3c7e86c11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Fix for nogo. See https://github.com/bazelbuild/rules_go/issues/3230
replace golang.org/x/tools => golang.org/x/tools v0.1.12
