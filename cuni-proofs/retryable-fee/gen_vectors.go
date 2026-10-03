// Reference reimplementation of Arbitrum retryable submission pricing,
// used ONLY to generate test vectors for the CuNi proof.
// Every formula below is transcribed from OffchainLabs/nitro v3.11.4
// (tag v3.11.4, commit 7d5ac27); the nitro source file + line for each
// step is cited. This file depends only on math/big — it does not import
// nitro, so it can run without nitro's module dependencies.
//
// What it is: an independent transcription of the pure fee arithmetic,
// cross-checked against the CuNi law. What it is not: the nitro code
// itself (see the cited lines for that).

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

type Vec struct {
	Fn   string   `json:"fn"`
	Args []string `json:"args"` // decimal strings (exact)
	Want string   `json:"want"` // decimal string (exact)
	Note string   `json:"note"`
}

var out []Vec

func emit(fn string, note string, want *big.Int, args ...*big.Int) {
	sargs := make([]string, len(args))
	for i, a := range args {
		sargs[i] = a.String()
	}
	out = append(out, Vec{fn, sargs, want.String(), note})
}

func bi(v int64) *big.Int { return big.NewInt(v) }
func bu(v uint64) *big.Int { return new(big.Int).SetUint64(v) }

// nitro arbos/retryables/retryable.go:394-398
//   func RetryableSubmissionFee(calldataLengthInBytes int, l1BaseFee *big.Int) *big.Int {
//       return arbmath.BigMulByUint(l1BaseFee, 1400+6*uint64(calldataLengthInBytes))
//   }
// arbmath.BigMulByUint: exact big-int multiply (util/arbmath/math.go:213-215).
func refSubmissionFee(calldataLen uint64, l1BaseFee *big.Int) *big.Int {
	return new(big.Int).Mul(l1BaseFee, bu(1400+6*calldataLen))
}

// nitro arbos/tx_processor.go:102-113 takeFunds(pool, take):
// panics if take < 0; else returns min(pool, take).
// tx_processor.go:377: submissionFeeRefund = takeFunds(availableRefund, MaxSubmissionFee - submissionFee)
// (valid only after the tx_processor.go:358 check maxSubmissionFee >= submissionFee).
func refSubmissionRefund(available, maxFee, fee *big.Int) *big.Int {
	excess := new(big.Int).Sub(maxFee, fee)
	if available.Cmp(excess) < 0 {
		return new(big.Int).Set(available)
	}
	return excess
}

// nitro arbos/tx_processor.go:457
//   gascost := arbmath.BigMulByUint(effectiveBaseFee, usergas)
func refL2GasCost(effectiveBaseFee *big.Int, gasLimit uint64) *big.Int {
	return new(big.Int).Mul(effectiveBaseFee, bu(gasLimit))
}

// nitro arbos/tx_processor.go:462-464 (ArbOS >= 11)
//   infraFee  := arbmath.BigMin(minBaseFee, effectiveBaseFee)
//   infraCost := arbmath.BigMulByUint(infraFee, usergas)
//   infraCost  = takeFunds(networkCost, infraCost)   // networkCost == gascost here
func refInfraFeeCost(networkCost, minBaseFee, effectiveBaseFee *big.Int, gasLimit uint64) *big.Int {
	fee := new(big.Int).Set(minBaseFee)
	if effectiveBaseFee.Cmp(fee) < 0 {
		fee.Set(effectiveBaseFee)
	}
	take := new(big.Int).Mul(fee, bu(gasLimit))
	if networkCost.Cmp(take) < 0 {
		return new(big.Int).Set(networkCost)
	}
	return take
}

// nitro arbos/tx_processor.go:477-481
//   gasPriceRefund := arbmath.BigMulByUint(arbmath.BigSub(tx.GasFeeCap, effectiveBaseFee), tx.Gas)
//   gasPriceRefund = takeFunds(availableRefund, gasPriceRefund)
// (non-estimation path: tx_processor.go:443-444 guarantees gasFeeCap >= effectiveBaseFee)
func refGasPriceRefund(available, gasFeeCap, effectiveBaseFee *big.Int, gas uint64) *big.Int {
	diff := new(big.Int).Sub(gasFeeCap, effectiveBaseFee)
	want := new(big.Int).Mul(diff, bu(gas))
	if available.Cmp(want) < 0 {
		return new(big.Int).Set(available)
	}
	return want
}

func main() {
	gwei := bi(1_000_000_000)

	// ---- submission_fee vectors ----
	// (len, fee) pairs chosen so fee*(1400+6*len) <= 2^53-1 (law domain)
	sfCases := []struct {
		len  uint64
		fee  *big.Int
		note string
	}{
		{0, bi(0), "zero len, zero fee -> 0"},
		{0, bi(1), "zero len: fixed 1400 overhead only"},
		{0, new(big.Int).Mul(gwei, bi(15)), "zero len at 15 gwei"},
		{1, bi(1), "one byte: 1406 * 1"},
		{2, bi(1), "two bytes"},
		{100, gwei, "100 bytes at 1 gwei"},
		{1024, new(big.Int).Mul(gwei, bi(100)), "1KiB at 100 gwei"},
		{32768, new(big.Int).Mul(gwei, bi(20)), "32KiB at 20 gwei"},
		{131072, new(big.Int).Mul(gwei, bi(10)), "128KiB at 10 gwei (large len, moderate fee)"},
		{100000, bi(10000000000), "100KB at 10 gwei (large fee, moderate len)"},
		{0, bi(6433713753386), "max fee at zero len: floor((2^53-1)/1400), domain edge accepted"},
	}
	for _, c := range sfCases {
		fee := new(big.Int).Mul(c.fee, bi(1))
		emit("submission_fee", c.note, refSubmissionFee(c.len, fee), bu(c.len), fee)
	}

	// ---- submission_refund vectors ----
	srCases := []struct {
		avail, maxFee, fee *big.Int
		note              string
	}{
		{bi(1000), bi(1000), bi(700), "avail covers excess exactly at kink"},
		{bi(200), bi(1000), bi(700), "avail binds: refund capped by deposit"},
		{bi(5000), bi(1000), bi(700), "avail ample: refund = maxFee - fee"},
		{bi(0), bi(1000), bi(700), "zero avail -> zero refund"},
		{bi(1000), bi(700), bi(700), "maxFee == fee -> zero refund"},
		{bi(1), bi(1000000), bi(1), "dust avail, large excess"},
	}
	for _, c := range srCases {
		emit("submission_refund", c.note, refSubmissionRefund(c.avail, c.maxFee, c.fee),
			c.avail, c.maxFee, c.fee)
	}

	// ---- l2_gas_cost vectors ----
	gcCases := []struct {
		baseFee *big.Int
		gas     uint64
		note    string
	}{
		{bi(0), 21000, "zero base fee"},
		{bi(1), 21000, "1 wei base fee at TxGas"},
		{gwei, 21000, "1 gwei at TxGas"},
		{new(big.Int).Mul(gwei, bi(100)), 90000, "100 gwei, 90k gas (domain edge)"},
		{bi(90000000000), 100000, "90 gwei, 100k gas (domain edge: floor((2^53-1)/100001))"},
		{bi(0), 0, "zero gas"},
	}
	for _, c := range gcCases {
		emit("l2_gas_cost", c.note, refL2GasCost(c.baseFee, c.gas), c.baseFee, bu(c.gas))
	}

	// ---- infra_fee_cost vectors ----
	icCases := []struct {
		netCost, minFee, baseFee *big.Int
		gas                    uint64
		note                   string
	}{
		{bi(21000000), gwei, new(big.Int).Mul(gwei, bi(2)), 21000, "minFee < baseFee: infra takes minFee*gas"},
		{bi(42000000), new(big.Int).Mul(gwei, bi(5)), new(big.Int).Mul(gwei, bi(2)), 21000, "minFee > baseFee: infra takes baseFee*gas"},
		{bi(21000000), gwei, gwei, 21000, "kink: minFee == baseFee"},
		{bi(1000), gwei, gwei, 21000, "netCost binds: infra capped by gascost"},
		{bi(0), gwei, gwei, 21000, "zero netCost -> zero"},
		{bi(21000000), bi(0), gwei, 21000, "zero minFee -> zero take"},
	}
	for _, c := range icCases {
		emit("infra_fee_cost", c.note,
			refInfraFeeCost(c.netCost, c.minFee, c.baseFee, c.gas),
			c.netCost, c.minFee, c.baseFee, bu(c.gas))
	}

	// ---- gas_price_refund vectors ----
	grCases := []struct {
		avail, cap, baseFee *big.Int
		gas               uint64
		note              string
	}{
		{bi(1000000), new(big.Int).Mul(gwei, bi(2)), gwei, 21000, "cap 2x baseFee, ample avail"},
		{bi(100), new(big.Int).Mul(gwei, bi(2)), gwei, 21000, "avail binds"},
		{bi(1000000), gwei, gwei, 21000, "kink: cap == baseFee -> zero"},
		{bi(0), new(big.Int).Mul(gwei, bi(2)), gwei, 21000, "zero avail -> zero"},
		{bi(1000000), new(big.Int).Mul(gwei, bi(3)), gwei, 0, "zero gas -> zero"},
	}
	for _, c := range grCases {
		emit("gas_price_refund", c.note,
			refGasPriceRefund(c.avail, c.cap, c.baseFee, c.gas),
			c.avail, c.cap, c.baseFee, bu(c.gas))
	}

	w := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "vectors: %d\n", len(out))
}
