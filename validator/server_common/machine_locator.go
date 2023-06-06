package server_common

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type MachineLocator struct {
	rootPath string
	latest   common.Hash
}

var ErrMachineNotFound = errors.New("machine not found")

func NewMachineLocator(rootPath string) (*MachineLocator, error) {
	var places []string

	if rootPath != "" {
		places = append(places, rootPath)
	} else {
		// Check the project dir: <project>/arbnode/node.go => ../../target/machines
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("failed to find root path")
		}
		projectDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
		projectPath := filepath.Join(filepath.Join(projectDir, "target"), "machines")
		places = append(places, projectPath)

		// Check the working directory: ./machines and ./target/machines
		workDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		workPath1 := filepath.Join(workDir, "machines")
		workPath2 := filepath.Join(filepath.Join(workDir, "target"), "machines")
		places = append(places, workPath1)
		places = append(places, workPath2)

		// Check above the executable: <binary> => ../../machines
		execfile, err := os.Executable()
		if err != nil {
			return nil, err
		}
		execPath := filepath.Join(filepath.Dir(filepath.Dir(execfile)), "machines")
		places = append(places, execPath)
	}

	for _, place := range places {
		if _, err := os.Stat(place); err == nil {
			var latestModuleRoot common.Hash
			latestModuleRootPath := filepath.Join(place, "latest", "module-root.txt")
			fileBytes, err := os.ReadFile(latestModuleRootPath)
			if err == nil {
				s := strings.TrimSpace(string(fileBytes))
				latestModuleRoot = common.HexToHash(s)
			}
			return &MachineLocator{place, latestModuleRoot}, nil
		}
	}
	return nil, ErrMachineNotFound
}

func (l MachineLocator) MachinePath(moduleRoot common.Hash) string {
	if moduleRoot == (common.Hash{}) || moduleRoot == l.latest {
		return filepath.Join(l.rootPath, "latest")
	} else {
		return filepath.Join(l.rootPath, moduleRoot.String())
	}
}

func (l MachineLocator) LatestWasmModuleRoot() common.Hash {
	return l.latest
}

func (l MachineLocator) RootPath() string {
	return l.rootPath
}
