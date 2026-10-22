// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nitroversion

import (
	"fmt"
	"os"
	"runtime/debug"
)

// These values are populated through the Go linker's -X option. They must be
// strings and package variables for the linker to address them.
var (
	tag      = ""
	branch   = ""
	commit   = ""
	datetime = ""
	modified = ""
)

type linkerProvenance struct {
	tag      string
	branch   string
	commit   string
	datetime string
	modified string
}

var current = mustLoadCurrent()

// Current returns the running binary's immutable, once-initialized version.
func Current() Version {
	return current
}

func mustLoadCurrent() Version {
	embedded := Provenance{Commit: "unknown"}
	if info, ok := debug.ReadBuildInfo(); ok {
		embedded = provenanceFromBuildSettings(info.Settings)
	}
	version, err := resolveProvenance(linkerProvenance{
		tag:      tag,
		branch:   branch,
		commit:   commit,
		datetime: datetime,
		modified: modified,
	}, embedded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid Nitro build provenance: %v\n", err)
		os.Exit(1)
	}
	return version
}

func provenanceFromBuildSettings(settings []debug.BuildSetting) Provenance {
	provenance := Provenance{Commit: "unknown"}
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			provenance.Commit = setting.Value
			if len(provenance.Commit) > 7 {
				provenance.Commit = provenance.Commit[:7]
			}
		case "vcs.time":
			provenance.Timestamp = setting.Value
		case "vcs.modified":
			provenance.Modified = setting.Value == "true"
		}
	}
	return provenance
}

func resolveProvenance(stamped linkerProvenance, embedded Provenance) (Version, error) {
	provenance := Provenance{
		Tag:       stamped.tag,
		Branch:    stamped.branch,
		Commit:    stamped.commit,
		Timestamp: stamped.datetime,
		Modified:  stamped.modified == "true",
	}
	if stamped.commit == "" {
		provenance.Commit = embedded.Commit
	}
	if stamped.datetime == "" {
		provenance.Timestamp = embedded.Timestamp
	}
	if stamped.modified == "" {
		provenance.Modified = embedded.Modified
	}
	return New(provenance)
}
