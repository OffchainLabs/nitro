// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package nitroversion describes the source provenance of a Nitro binary.
package nitroversion

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

var (
	// ErrInvalidCanonicalVersion identifies a value that is not a canonical
	// semantic version.
	ErrInvalidCanonicalVersion = errors.New("invalid canonical nitro version")
	// ErrInvalidVersionRange identifies malformed or contradictory version bounds.
	ErrInvalidVersionRange = errors.New("invalid nitro version range")
	// ErrUnsupportedVersion identifies a binary version outside valid configured bounds.
	ErrUnsupportedVersion = errors.New("unsupported nitro version")
)

// CanonicalVersion is a validated canonical semantic version. Its zero value
// represents an absent optional version.
type CanonicalVersion struct {
	value string
}

// ParseCanonicalVersion validates a non-empty canonical semantic version.
func ParseCanonicalVersion(value string) (CanonicalVersion, error) {
	if value == "" || semver.Canonical(value) != value {
		return CanonicalVersion{}, fmt.Errorf("%w: %q, expected a canonical semantic version", ErrInvalidCanonicalVersion, value)
	}
	return CanonicalVersion{value: value}, nil
}

// String returns the canonical semantic version, or an empty string for the
// zero value.
func (v CanonicalVersion) String() string {
	return v.value
}

// IsZero reports whether no version was specified.
func (v CanonicalVersion) IsZero() bool {
	return v.value == ""
}

// Provenance is the raw source information used to construct a Version.
type Provenance struct {
	Tag       string
	Branch    string
	Commit    string
	Timestamp string
	Modified  bool
}

// Version describes the build provenance embedded in a Nitro binary. Its tag
// and commit time have already been validated at the provenance boundary.
type Version struct {
	tag        CanonicalVersion
	branch     string
	commit     string
	commitTime time.Time
	modified   bool
}

var _ slog.LogValuer = Version{}
var _ fmt.Stringer = Version{}

// New validates explicit provenance and constructs a Version without consulting
// the running binary's linker stamps or Go build information.
func New(provenance Provenance) (Version, error) {
	var tag CanonicalVersion
	var err error
	if provenance.Tag != "" {
		tag, err = ParseCanonicalVersion(provenance.Tag)
		if err != nil {
			return Version{}, fmt.Errorf("tag: %w", err)
		}
	}

	var commitTime time.Time
	if provenance.Timestamp != "" && provenance.Timestamp != "unknown" {
		commitTime, err = time.Parse(time.RFC3339, provenance.Timestamp)
		if err != nil {
			return Version{}, fmt.Errorf("commit timestamp %q: %w", provenance.Timestamp, err)
		}
	}
	if !tag.IsZero() && commitTime.IsZero() {
		return Version{}, fmt.Errorf("tagged version %s has no commit timestamp", provenance.Tag)
	}

	return Version{
		tag:        tag,
		branch:     provenance.Branch,
		commit:     provenance.Commit,
		commitTime: commitTime,
		modified:   provenance.Modified,
	}, nil
}

// revision derives the displayed identity of the build: the tag for release
// builds, the branch for other CI builds, or "local" when no provenance was
// stamped by the linker; the commit and compact UTC commit time follow as
// build metadata.
func (v Version) revision() string {
	prefix := v.tag.String()
	if prefix == "" {
		prefix = v.branch
	}
	if prefix == "" {
		prefix = "local"
	}
	commitTime := "unknown"
	if !v.commitTime.IsZero() {
		commitTime = v.commitTime.UTC().Format("20060102T150405Z")
	}
	revision := prefix + "+" + v.commit + "-" + commitTime
	if v.modified {
		revision += "-modified"
	}
	return revision
}

func (v Version) commitTimeString() string {
	if v.commitTime.IsZero() {
		return "unknown"
	}
	return v.commitTime.Format(time.RFC3339)
}

// String returns the displayed revision.
func (v Version) String() string {
	return v.revision()
}

// LogValue returns the version as structured logging attributes.
func (v Version) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("revision", v.revision()),
		slog.String("vcs.time", v.commitTimeString()),
	)
}

// GethVersion returns the displayed revision without a leading "v".
func (v Version) GethVersion() string {
	return strings.TrimPrefix(v.revision(), "v")
}

// IsTagged reports whether the binary was built from a release tag.
func (v Version) IsTagged() bool {
	return !v.tag.IsZero()
}

func (v Version) IsVersionOlderThan(targetVersion CanonicalVersion) bool {
	if !v.IsTagged() {
		return false
	}
	return semver.Compare(v.tag.String(), targetVersion.String()) < 0
}

func (v Version) IsCommitTimestampOlderThan(ts time.Time) bool {
	return v.commitTime.Before(ts)
}

// CheckVersionBounds validates the inclusive version bounds and checks whether
// this Nitro version falls within them. A zero min or max disables that side of
// the range. Untagged builds satisfy valid bounds because they do not have an
// ordered semantic version.
func (v Version) CheckVersionBounds(minVersion, maxVersion CanonicalVersion) error {
	if !minVersion.IsZero() && !maxVersion.IsZero() &&
		semver.Compare(minVersion.String(), maxVersion.String()) > 0 {
		return fmt.Errorf("%w: conf.min-version %s is greater than conf.max-version %s",
			ErrInvalidVersionRange, minVersion, maxVersion)
	}
	if !v.IsTagged() {
		return nil
	}
	tag := v.tag.String()
	if (!minVersion.IsZero() && semver.Compare(tag, minVersion.String()) < 0) ||
		(!maxVersion.IsZero() && semver.Compare(tag, maxVersion.String()) > 0) {
		return fmt.Errorf("%w: this binary is nitro %s, but the configuration declares conf.min-version %q and conf.max-version %q",
			ErrUnsupportedVersion, v.revision(), minVersion.String(), maxVersion.String())
	}
	return nil
}
