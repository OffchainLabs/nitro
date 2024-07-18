package github

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-github/v62/github"
)

var wasmRootExp = regexp.MustCompile(`\*\*WAVM Module Root\*\*: (0x[a-f0-9]{64})`)

type ConsensusRelease struct {
	WavmModuleRoot string
	MachineWavmURL url.URL
	ReplayWasmURL  url.URL
}

// NitroReleases returns the most recent 50 releases of the Nitro repository.
func NitroReleases(ctx context.Context) ([]*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
	opts := &github.ListOptions{
		PerPage: 50,
	}
	releases, _, err := client.Repositories.ListReleases(ctx, "OffchainLabs", "nitro", opts)
	return releases, err
}

// LatestConsensusRelease returns data about the latest consensus release.
func LatestConsensusRelease(ctx context.Context) (*ConsensusRelease, error) {
	releases, err := NitroReleases(ctx)
	if err != nil {
		return nil, err
	}
	var found *ConsensusRelease
	for _, release := range releases {
		if strings.HasPrefix(release.GetTagName(), "consensus") {
			if found, err = fromRelease(release); err != nil {
				return nil, err
			}
			break
		}
	}
	if found == nil {
		return nil, errors.New("no consensus release found")
	}
	return found, nil
}

func fromRelease(release *github.RepositoryRelease) (*ConsensusRelease, error) {
	// TODO(eljobe): Consider making the module-root.txt a release asset.
	// This is currently brittle because it relies on the release body format.
	matches := wasmRootExp.FindStringSubmatch(release.GetBody())
	if len(matches) != 2 {
		return nil, errors.New("no WAVM module root found in release body")
	}
	wavmModuleRoot := matches[1]
	var machineWavmURL url.URL
	var replayWasmURL url.URL
	for _, asset := range release.Assets {
		if asset.GetName() == "machine.wavm.br" {
			wURL, err := url.Parse(asset.GetBrowserDownloadURL())
			if err != nil {
				return nil, err
			}
			machineWavmURL = *wURL
		}
		if asset.GetName() == "replay.wasm" {
			rURL, err := url.Parse(asset.GetBrowserDownloadURL())
			if err != nil {
				return nil, err
			}
			replayWasmURL = *rURL
		}
	}
	return &ConsensusRelease{
		WavmModuleRoot: wavmModuleRoot,
		MachineWavmURL: machineWavmURL,
		ReplayWasmURL:  replayWasmURL,
	}, nil
}
