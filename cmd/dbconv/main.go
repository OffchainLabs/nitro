package main

import (
	"os"

	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
	flag "github.com/spf13/pflag"
)

func parseDBConv(args []string) (*dbconv.DBConvConfig, error) {
	f := flag.NewFlagSet("dbconv", flag.ContinueOnError)
	dbconv.DBConvConfigAddOptions(f)
	// TODO
	return nil, nil
}

func main() {
	args := os.Args[1:]
	_, _ = parseDBConv(args)
}
