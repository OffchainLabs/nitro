package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/wavmio"
)

func main() {
	wavmio.StubInit()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlError)
	log.Root().SetHandler(glogger)

	code := wavmio.ReadInboxMessage(0)
	fmt.Printf("Executing EVM bytecode: %v\n", hex.EncodeToString(code))
	output, _, err := runtime.Execute(code, []byte{}, nil)
	if err != nil {
		panic(fmt.Sprintf("Error executing EVM: %v", err.Error()))
	}
	fmt.Printf("Output: %v\n", hex.EncodeToString(output))

	wavmio.StubFinal()
}
