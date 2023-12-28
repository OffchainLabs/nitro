package main

import (
	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
	flag "github.com/spf13/pflag"
)

func parseDBConv(args []string) (*DBConvConfig, error) {
	f := flag.NewFlagSet("dbconv", flag.ContinueOnError)
	dbconv.DBConvConfigAddOptions(f)
}

func main() {

}
