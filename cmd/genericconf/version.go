// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package genericconf

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var (
	errVersionNotStamped    = errors.New("nitro version was not stamped by the linker")
	errVersionNotComparable = errors.New("nitro version is not valid semantic version")
)

// VersionInfo describes the version embedded in the running binary. RawVersion
// is suitable for display and protocol client identifiers; ComparableVersion
// converts linker-stamped release builds to SemVer for ordering.
type VersionInfo struct {
	RawVersion string
	Timestamp  string
	Stamped    bool
}

// WithoutV returns the displayed version without a leading "v".
func (v VersionInfo) WithoutV() string {
	return strings.TrimPrefix(v.RawVersion, "v")
}

// stampedCommitSuffix matches the commit hash appended by the build pipeline,
// optionally followed by the dirty-tree marker appended by GetVersion.
var stampedCommitSuffix = regexp.MustCompile(`-([0-9a-f]{7,40})(-modified)?$`)

// ComparableVersion returns a valid SemVer representation of a linker-stamped
// Nitro version. The build pipeline separates its commit hash with "-", which
// SemVer interprets as prerelease data; convert only that separator to "+" so
// the hash remains visible but does not affect ordering.
func (v VersionInfo) ComparableVersion() (string, error) {
	if !v.Stamped {
		return "", errVersionNotStamped
	}

	comparable := v.RawVersion
	if !strings.Contains(comparable, "+") {
		if stampedCommitSuffix.MatchString(comparable) {
			comparable = stampedCommitSuffix.ReplaceAllString(comparable, `+$1$2`)
		} else if strings.HasSuffix(comparable, "-modified") {
			comparable = strings.TrimSuffix(comparable, "-modified") + "+modified"
		}
	}
	if !strings.HasPrefix(comparable, "v") {
		comparable = "v" + comparable
	}
	if !semver.IsValid(comparable) {
		return "", fmt.Errorf("%w: %q", errVersionNotComparable, v.RawVersion)
	}
	return comparable, nil
}
