// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"fmt"
	"math/big"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ethereum/go-ethereum/params"
)

type Result struct {
	baseFee  *big.Int
	gasRatio float64
}

func printOutput(config Config, results []Result) {
	if config.ShouldExportCSV() {
		printCsvOutput(results)
	} else {
		printTextOutput(config, results)
	}
}

func printCsvOutput(results []Result) {
	fmt.Println("i,baseFee,gasRatio")
	for i, result := range results {
		fmt.Printf("%v,%v,%.3f\n", i, toGwei(result.baseFee), result.gasRatio)
	}
}

func printTextOutput(config Config, results []Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	config.Print(w)
	w.Flush()

	fmt.Println()

	w = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "i\tbaseFee\tgasRatio\t")
	for i, result := range results {
		if !config.ShouldPrintLine(i) {
			continue
		}
		fmt.Fprintf(w, "%v\t", i)
		fmt.Fprintf(w, "%v\t", toGwei(result.baseFee))
		fmt.Fprintf(w, "%.3f\t", result.gasRatio)
		fmt.Fprintln(w)
	}
	w.Flush()
}

func toGwei(wei *big.Int) string {
	gweiDivisor := big.NewInt(params.GWei)
	weiRat := new(big.Rat).SetInt(wei)
	gweiDivisorRat := new(big.Rat).SetInt(gweiDivisor)
	gweiRat := new(big.Rat).Quo(weiRat, gweiDivisorRat)
	return gweiRat.FloatString(3)
}

func toPrettyUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	parts := []string{}
	for v >= 1000 {
		parts = append(parts, fmt.Sprintf("%03d", v%1000))
		v = v / 1000
	}
	if v > 0 {
		parts = append(parts, fmt.Sprint(v))
	}
	slices.Reverse(parts)
	return strings.Join(parts, ",")
}

func toPrettyInt(v int64) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		return "-" + toPrettyUint(uint64(-v))
	}
	return toPrettyUint(uint64(v))
}
