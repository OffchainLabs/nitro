// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package confighelpers

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/mod/semver"
)

// ErrUnsupportedVersion is returned when this binary's version falls outside the
// inclusive range a configuration declares through conf.min-version and
// conf.max-version.
var ErrUnsupportedVersion = errors.New("unsupported nitro version")

// CheckVersionRange tests this binary's version against the inclusive range
// [minVersion, maxVersion] declared by the configuration. Either bound may be
// empty, which disables that side; both empty disables the check entirely.
func CheckVersionRange(minVersion, maxVersion string) error {
	if minVersion == "" && maxVersion == "" {
		return nil
	}
	// Require the configured spelling itself to be canonical. Canonical SemVer
	// keeps prereleases, but expands shorthand and removes build metadata, so an
	// equality check rejects both. It also prevents semver.Compare from silently
	// ordering an invalid bound below every valid version.
	if minVersion != "" && semver.Canonical(minVersion) != minVersion {
		return fmt.Errorf("invalid conf.min-version %q, expected a semantic version such as v3.9.0", minVersion)
	}
	if maxVersion != "" && semver.Canonical(maxVersion) != maxVersion {
		return fmt.Errorf("invalid conf.max-version %q, expected a semantic version such as v3.10.0", maxVersion)
	}
	if minVersion != "" && maxVersion != "" && semver.Compare(minVersion, maxVersion) > 0 {
		return fmt.Errorf("conf.min-version %s is greater than conf.max-version %s", minVersion, maxVersion)
	}

	versionInfo := GetVersion()
	comparableVersion, err := versionInfo.ComparableVersion()
	if err != nil {
		// Logging is not set up yet at this point in startup: geth's root logger
		// discards everything until genericconf.InitLog runs, which happens after
		// the configuration has been parsed.
		fmt.Fprintf(os.Stderr, "WARN: nitro version %q cannot be compared (%v), "+
			"skipping the conf.min-version/conf.max-version check\n", versionInfo.RawVersion, err)
		return nil
	}
	if minVersion != "" && semver.Compare(comparableVersion, minVersion) < 0 {
		return fmt.Errorf("%w: this binary is nitro %s, but the configuration declares conf.min-version %s",
			ErrUnsupportedVersion, versionInfo.RawVersion, minVersion)
	}
	if maxVersion != "" && semver.Compare(comparableVersion, maxVersion) > 0 {
		return fmt.Errorf("%w: this binary is nitro %s, but the configuration declares conf.max-version %s",
			ErrUnsupportedVersion, versionInfo.RawVersion, maxVersion)
	}
	return nil
}

// SetVersionForTesting overrides the version normally injected by the linker and
// returns a function restoring the previous value. Tests need this because
// `go test` never sets those ldflags, which would make every range check skip.
func SetVersionForTesting(v string) func() {
	oldVersion, oldModified := version, modified
	version, modified = v, "false"
	return func() {
		version, modified = oldVersion, oldModified
	}
}
