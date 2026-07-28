// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package confighelpers

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

// ErrUnsupportedVersion is returned when this binary's version falls outside the
// inclusive range a configuration declares through conf.min-version and
// conf.max-version.
var ErrUnsupportedVersion = errors.New("unsupported nitro version")

// commitSuffixRegexp matches the short commit hash the build appends to the
// version tag: .github/buildspec.yml sets
// NITRO_VERSION=${VERSION_TAG}-${COMMIT_HASH} with
// COMMIT_HASH=$(git rev-parse --short=7 HEAD). git emits more than 7 characters
// when a 7-character prefix is ambiguous, so accept 7..40.
var commitSuffixRegexp = regexp.MustCompile(`-[0-9a-f]{7,40}$`)

// comparableVersion converts the version baked into this binary into something
// semver can rank, and reports false when the range check should not apply.
//
// Semver reads anything after a hyphen as a prerelease, which ranks below the
// bare tag. That is wrong for the commit hash the build appends -- a released
// v3.9.9 binary reports v3.9.9-<hash> and would otherwise fail
// conf.min-version v3.9.9 -- but right for a genuine "-rc.N", which really does
// come before its release. So only the hash and the "-modified" GetVersion adds
// for a dirty tree are removed; a release candidate keeps its precedence.
//
// The suffixes come off in that order because GetVersion appends "-modified"
// after the build appended the hash. Stripping before validating also matters:
// an all-digit hash with a leading zero ("v3.9.9-0123456") is an invalid
// prerelease identifier, so validating first would reject a real release build.
//
// Enforcement is deliberately limited to builds the release pipeline produced
// from a semantic version tag. An empty version means the ldflags were never
// set, so this is a local build or a test binary. A version that then fails
// semver validation means VERSION_TAG was a branch name or the literal "dev".
// Both skip the check rather than blocking the node.
func comparableVersion() (string, bool) {
	if version == "" {
		return "", false
	}
	v, _, _ := GetVersion()
	v = strings.TrimSuffix(v, "-modified")
	v = commitSuffixRegexp.ReplaceAllString(v, "")
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "", false
	}
	return v, true
}

// CheckVersionRange tests this binary's version against the inclusive range
// [minVersion, maxVersion] declared by the configuration. Either bound may be
// empty, which disables that side; both empty disables the check entirely.
func CheckVersionRange(minVersion, maxVersion string) error {
	if minVersion == "" && maxVersion == "" {
		return nil
	}
	// Reject malformed bounds explicitly. semver.Compare ranks an invalid string
	// below every valid one, so an unchecked min-version of "3.9.0" (no leading
	// v) would silently be a no-op while the same max-version would reject
	// unconditionally.
	if minVersion != "" && !semver.IsValid(minVersion) {
		return fmt.Errorf("invalid conf.min-version %q, expected a semantic version such as v3.9.0", minVersion)
	}
	if maxVersion != "" && !semver.IsValid(maxVersion) {
		return fmt.Errorf("invalid conf.max-version %q, expected a semantic version such as v3.10.0", maxVersion)
	}
	if minVersion != "" && maxVersion != "" && semver.Compare(minVersion, maxVersion) > 0 {
		return fmt.Errorf("conf.min-version %s is greater than conf.max-version %s", minVersion, maxVersion)
	}

	nodeVersion, _, _ := GetVersion()
	v, ok := comparableVersion()
	if !ok {
		// Logging is not set up yet at this point in startup: geth's root logger
		// discards everything until genericconf.InitLog runs, which happens after
		// the configuration has been parsed.
		fmt.Fprintf(os.Stderr, "WARN: nitro version %q is not a release build, "+
			"skipping the conf.min-version/conf.max-version check\n", nodeVersion)
		return nil
	}
	if minVersion != "" && semver.Compare(v, minVersion) < 0 {
		return fmt.Errorf("%w: this binary is nitro %s, but the configuration declares conf.min-version %s",
			ErrUnsupportedVersion, nodeVersion, minVersion)
	}
	if maxVersion != "" && semver.Compare(v, maxVersion) > 0 {
		return fmt.Errorf("%w: this binary is nitro %s, but the configuration declares conf.max-version %s",
			ErrUnsupportedVersion, nodeVersion, maxVersion)
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
