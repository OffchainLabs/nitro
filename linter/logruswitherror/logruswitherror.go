// Static analyzer to ensure that log statements do not use errors in
// templated log statements. Authors should use logrus.WithError().
package main

import (
	"github.com/prysmaticlabs/prysm/v4/tools/analyzers/logruswitherror"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(logruswitherror.Analyzer)
}
