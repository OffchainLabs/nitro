// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/offchainlabs/nitro/util/containers"
)

var (
	flagTests       = flag.String("systest.tests", "", "comma-separated test name patterns (glob); default: all")
	flagCategories  = flag.String("systest.categories", defaultCategory, "comma-separated categories to run; tests without WithCategory are in the default category")
	flagMaxWeight   = flag.Int("systest.max-weight", 0, "max weighted concurrency in this binary (0 = GOMAXPROCS)")
	flagArbOS       = flag.Uint64("systest.arbos", 0, "pin ArbOS version (0 = test default)")
	flagStateScheme = flag.String("systest.state-scheme", "", "pin state scheme (hash|path)")
	flagDBEngine    = flag.String("systest.db-engine", "", "pin db engine (pebble|leveldb|in-memory)")

	flagMatrixArbOS  = flag.String("systest.matrix.arbos", "", "matrix ArbOS versions, e.g. 30,40,50")
	flagMatrixStates = flag.String("systest.matrix.state-scheme", "", "matrix state schemes, e.g. hash,path")
	flagMatrixDBs    = flag.String("systest.matrix.db-engine", "", "matrix db engines")

	flagDryRun = flag.Bool("systest.dry-run", false, "print resolved specs and exit without running")

	flagTestTimeout = flag.Duration("systest.test-timeout", defaultTestTimeout, "per-scenario wall-clock backstop against hangs (0 = none); keep below the package -timeout so a hang fails one test, not the whole binary")
)

// parseCLI parses the command-line flags and returns the corresponding scheduleParams.
func parseCLI() scheduleParams {
	if !flag.Parsed() {
		flag.Parse()
	}
	p, err := buildCLIParams(
		*flagTests, *flagCategories, *flagArbOS, *flagStateScheme, *flagDBEngine,
		*flagMatrixArbOS, *flagMatrixStates, *flagMatrixDBs)
	if err != nil {
		panic(err.Error())
	}
	p.DefaultStateScheme = envDefaultScheme()
	return p
}

// buildCLIParams assembles and validates scheduleParams from raw flag values.
func buildCLIParams(tests, categories string, arbOS uint64, stateScheme, dbEngine, matrixArbOS, matrixStates, matrixDBs string) (scheduleParams, error) {
	p := scheduleParams{
		Tests:      splitSet(tests),
		Categories: splitSet(categories),
	}
	for pattern := range p.Tests {
		if _, err := path.Match(pattern, ""); err != nil {
			return p, fmt.Errorf("systest: invalid -systest.tests pattern %q: %w", pattern, err)
		}
	}
	if arbOS != 0 {
		p.ArbOS = containers.Some(arbOS)
	}
	if stateScheme != "" {
		s := StateScheme(stateScheme)
		if !s.Valid() {
			return p, fmt.Errorf("systest: invalid -systest.state-scheme %q", stateScheme)
		}
		p.StateScheme = containers.Some(s)
	}
	if dbEngine != "" {
		e := DBEngine(dbEngine)
		if !e.Valid() {
			return p, fmt.Errorf("systest: invalid -systest.db-engine %q", dbEngine)
		}
		p.DBEngine = containers.Some(e)
	}
	for _, s := range splitCSV(matrixArbOS) {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil || v == 0 {
			return p, fmt.Errorf("systest: invalid -systest.matrix.arbos value %q: must be a positive version", s)
		}
		if slices.Contains(p.MatrixArbOS, v) {
			return p, fmt.Errorf("systest: duplicate -systest.matrix.arbos value %d", v)
		}
		p.MatrixArbOS = append(p.MatrixArbOS, v)
	}
	for _, s := range splitCSV(matrixStates) {
		ss := StateScheme(s)
		if !ss.Valid() {
			return p, fmt.Errorf("systest: invalid -systest.matrix.state-scheme %q", s)
		}
		if slices.Contains(p.MatrixStates, ss) {
			return p, fmt.Errorf("systest: duplicate -systest.matrix.state-scheme value %q", s)
		}
		p.MatrixStates = append(p.MatrixStates, ss)
	}
	for _, s := range splitCSV(matrixDBs) {
		de := DBEngine(s)
		if !de.Valid() {
			return p, fmt.Errorf("systest: invalid -systest.matrix.db-engine %q", s)
		}
		if slices.Contains(p.MatrixDBs, de) {
			return p, fmt.Errorf("systest: duplicate -systest.matrix.db-engine value %q", s)
		}
		p.MatrixDBs = append(p.MatrixDBs, de)
	}
	return p, nil
}

func isDryRun() bool { return *flagDryRun }

// testTimeout is the default per-scenario wall-clock backstop, from
// -systest.test-timeout. A test's own WithTimeout overrides it; 0 means no deadline.
func testTimeout() time.Duration { return *flagTestTimeout }

// baseCapacity returns -systest.max-weight, or GOMAXPROCS if unset.
func baseCapacity() int {
	c := *flagMaxWeight
	if c <= 0 {
		c = runtime.GOMAXPROCS(0)
	}
	return c
}

func printDryRun(items []scheduledTest) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tWEIGHT\tARBOS\tSTATE\tDB\tCATEGORY\tNOTE")
	for _, it := range items {
		note := ""
		if it.SkipReason != "" {
			note = "SKIP: " + it.SkipReason
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
			it.Spec.Name, it.Spec.Weight, dashIfNone(it.Spec.ArbOSVersion),
			dashIfNone(it.Spec.StateScheme), dashIfNone(it.Spec.DBEngine),
			it.Spec.Category, note)
	}
	w.Flush()
}

func dashIfNone[T any](o containers.Option[T]) string {
	if o.IsNone() {
		return "-"
	}
	return fmt.Sprint(o.Unwrap())
}

func splitCSV(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitSet(s string) map[string]bool {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return nil
	}
	out := make(map[string]bool, len(parts))
	for _, p := range parts {
		out[p] = true
	}
	return out
}
