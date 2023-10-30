// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build go1.18

package genericconf

import "runtime/debug"

func GetVersion(definedVersion string, definedTime string, definedModified string) (string, string, string) {
	vcsVersion := "development"
	vcsTime := "development"
	vcsModified := "false"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		vcsVersion = "unknown"
	}
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			vcsVersion = v.Value
			if len(vcsVersion) > 7 {
				vcsVersion = vcsVersion[:7]
			}
		} else if v.Key == "vcs.time" {
			vcsTime = v.Value
		} else if v.Key == "vcs.modified" {
			vcsModified = v.Value
		}
	}

	// Defined values override if provided
	if len(definedVersion) > 0 {
		vcsVersion = definedVersion
	}
	if len(definedTime) > 0 {
		vcsTime = definedTime
	}
	if len(definedModified) > 0 {
		vcsModified = definedModified
	}

	if vcsModified == "true" {
		vcsVersion = vcsVersion + "-modified"
	}

	strippedVersion := vcsVersion
	if len(strippedVersion) > 0 && strippedVersion[0] == 'v' {
		strippedVersion = strippedVersion[1:]
	}

	return vcsVersion, strippedVersion, vcsTime
}
