package server_common

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type MachineLocator struct {
	rootPath    string
	latest      common.Hash
	moduleRoots []common.Hash
}

var ErrMachineNotFound = errors.New("machine not found")

func NewMachineLocator(rootPath string) (*MachineLocator, error) {
	dirs := []string{rootPath}
	if rootPath == "" {
		// Check the project dir: <project>/arbnode/node.go => ../../target/machines
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("failed to find root path")
		}
		projectDir := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
		projectPath := filepath.Join(filepath.Join(projectDir, "target"), "machines")
		dirs = append(dirs, projectPath)

		// Check the working directory: ./machines and ./target/machines
		workDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		workPath1 := filepath.Join(workDir, "machines")
		workPath2 := filepath.Join(filepath.Join(workDir, "target"), "machines")
		dirs = append(dirs, workPath1)
		dirs = append(dirs, workPath2)

		// Check above the executable: <binary> => ../../machines
		execfile, err := os.Executable()
		if err != nil {
			return nil, err
		}
		execPath := filepath.Join(filepath.Dir(filepath.Dir(execfile)), "machines")
		dirs = append(dirs, execPath)
	}

	var (
		moduleRoots      = make(map[common.Hash]bool)
		latestModuleRoot common.Hash
	)

	for _, dir := range dirs {
		fInfo, err := os.Stat(dir)
		if err != nil {
			log.Warn("Getting file info", "error", err)
			continue
		}
		if !fInfo.IsDir() {
			// Skip files that are not directories.
			continue
		}
		files, err := os.ReadDir(dir)
		if err != nil {
			log.Warn("Reading directory", "dir", dir, "error", err)
		}
		for _, file := range files {
			mrFile := filepath.Join(dir, file.Name(), "module-root.txt")
			if _, err := os.Stat(mrFile); err != nil {
				// Skip if module-roots file does not exist.
				continue
			}
			mrContent, err := os.ReadFile(mrFile)
			if err != nil {
				log.Warn("Reading module roots file", "file path", mrFile, "error", err)
				continue
			}
			moduleRoot := common.HexToHash(strings.TrimSpace(string(mrContent)))
			if file.Name() != "latest" && file.Name() != moduleRoot.Hex() {
				continue
			}
			moduleRoots[moduleRoot] = true
			if file.Name() == "latest" {
				latestModuleRoot = moduleRoot
			}
			rootPath = dir
		}
		if rootPath != "" {
			break
		}
	}
	var roots []common.Hash
	for k := range moduleRoots {
		roots = append(roots, k)
	}
	return &MachineLocator{
		rootPath:    rootPath,
		latest:      latestModuleRoot,
		moduleRoots: roots,
	}, nil
}

func (l MachineLocator) GetMachinePath(moduleRoot common.Hash) string {
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

func (l MachineLocator) ModuleRoots() []common.Hash {
	return l.moduleRoots
}
