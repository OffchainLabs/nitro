//go:build go1.18

package conf

import "runtime/debug"

func GetVersion() (string, string) {
	vcsRevision := "development"
	vcsTime := "development"
	vcsModified := "false"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		vcsRevision = "unknown"
	}
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			vcsRevision = v.Value
		} else if v.Key == "vcs.time" {
			vcsTime = v.Value
		} else if v.Key == "vcs.modified" {
			vcsModified = v.Value
		}
	}
	if vcsModified == "true" {
		vcsRevision = vcsRevision + "-modified"
	}

	return vcsRevision, vcsTime
}
