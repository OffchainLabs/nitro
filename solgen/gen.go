// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
)

type HardHatArtifact struct {
	Format       string        `json:"_format"`
	ContractName string        `json:"contractName"`
	SourceName   string        `json:"sourceName"`
	Abi          []interface{} `json:"abi"`
	Bytecode     string        `json:"bytecode"`
}

type FoundryBytecode struct {
	Object string `json:"object"`
}

type FoundryArtifact struct {
	Abi      []interface{}   `json:"abi"`
	Bytecode FoundryBytecode `json:"bytecode"`
}

type moduleInfo struct {
	contractNames []string
	abis          []string
	bytecodes     []string
}

func (m *moduleInfo) addArtifact(artifact HardHatArtifact) {
	abi, err := json.Marshal(artifact.Abi)
	if err != nil {
		log.Fatal(err)
	}
	m.contractNames = append(m.contractNames, artifact.ContractName)
	m.abis = append(m.abis, string(abi))
	m.bytecodes = append(m.bytecodes, artifact.Bytecode)
}

func (m *moduleInfo) exportABIs(dest string) {
	for i, name := range m.contractNames {
		path := filepath.Join(dest, name+".abi")
		abi := m.abis[i] + "\n"

		// #nosec G306
		err := os.WriteFile(path, []byte(abi), 0o644)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("bad path")
	}
	root := filepath.Dir(filename)
	parent := filepath.Dir(root)
	filePaths, err := filepath.Glob(filepath.Join(parent, "contracts", "build", "contracts", "src", "*", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}

	filePathsInternal, err := filepath.Glob(filepath.Join(parent, "contracts-legacy", "build", "contracts", "src", "*", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}

	filePathsSafeSmartAccount, err := filepath.Glob(filepath.Join(parent, "safe-smart-account", "build", "artifacts", "contracts", "*", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	filePathsSafeSmartAccountOuter, err := filepath.Glob(filepath.Join(parent, "safe-smart-account", "build", "artifacts", "contracts", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}

	filePaths = append(filePaths, filePathsInternal...)
	filePaths = append(filePaths, filePathsSafeSmartAccount...)
	filePaths = append(filePaths, filePathsSafeSmartAccountOuter...)

	modules := make(map[string]*moduleInfo)

	for _, path := range filePaths {
		if strings.Contains(path, ".dbg.json") {
			continue
		}

		if strings.Contains(path, "precompiles") {
			continue
		}

		dir, file := filepath.Split(path)
		dir, _ = filepath.Split(dir[:len(dir)-1])
		_, module := filepath.Split(dir[:len(dir)-1])
		module = strings.ReplaceAll(module, "-", "_")

		if strings.Contains(path, "contracts-legacy") {
			module += "_legacy_"
		}

		module += "gen"

		name := file[:len(file)-5]

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}

		artifact := HardHatArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		modInfo := modules[module]
		if modInfo == nil {
			modInfo = &moduleInfo{}
			modules[module] = modInfo
		}
		modInfo.addArtifact(artifact)
	}

	yulFilePaths, err := filepath.Glob(filepath.Join(parent, "contracts", "out", "*", "*.yul", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	yulFilePathsGasDimensions, err := filepath.Glob(filepath.Join(parent, "contracts-local", "out", "gas-dimensions-yul", "*.yul", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	yulFilePaths = append(yulFilePaths, yulFilePathsGasDimensions...)
	yulModInfo := modules["yulgen"]
	if yulModInfo == nil {
		yulModInfo = &moduleInfo{}
		modules["yulgen"] = yulModInfo
	}
	for _, path := range yulFilePaths {
		_, file := filepath.Split(path)
		name := file[:len(file)-5]

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}

		artifact := FoundryArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		yulModInfo.addArtifact(HardHatArtifact{
			ContractName: name,
			Abi:          artifact.Abi,
			Bytecode:     artifact.Bytecode.Object,
		})
	}

	gasDimensionsFilePaths, err := filepath.Glob(filepath.Join(parent, "contracts-local", "out", "gas-dimensions", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	gasDimensionsModInfo := modules["gas_dimensionsgen"]
	if gasDimensionsModInfo == nil {
		gasDimensionsModInfo = &moduleInfo{}
		modules["gas_dimensionsgen"] = gasDimensionsModInfo
	}
	for _, path := range gasDimensionsFilePaths {
		_, file := filepath.Split(path)
		name := file[:len(file)-5]

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}
		artifact := FoundryArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		gasDimensionsModInfo.addArtifact(HardHatArtifact{
			ContractName: name,
			Abi:          artifact.Abi,
			Bytecode:     artifact.Bytecode.Object,
		})
	}

	localFilePaths, err := filepath.Glob(filepath.Join(parent, "contracts-local", "out", "src", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	localModInfo := modules["localgen"]
	if localModInfo == nil {
		localModInfo = &moduleInfo{}
		modules["localgen"] = localModInfo
	}
	for _, path := range localFilePaths {
		_, file := filepath.Split(path)
		name := file[:len(file)-5]

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}
		artifact := FoundryArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		localModInfo.addArtifact(HardHatArtifact{
			ContractName: name,
			Abi:          artifact.Abi,
			Bytecode:     artifact.Bytecode.Object,
		})
	}

	precompilesFilePaths, err := filepath.Glob(filepath.Join(parent, "contracts-local", "out", "precompiles", "*.sol", "*.json"))
	if err != nil {
		log.Fatal(err)
	}
	precompilesModInfo := modules["precompilesgen"]
	if precompilesModInfo == nil {
		precompilesModInfo = &moduleInfo{}
		modules["precompilesgen"] = precompilesModInfo
	}
	for _, path := range precompilesFilePaths {
		_, file := filepath.Split(path)
		name := file[:len(file)-5]

		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}
		artifact := FoundryArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		precompilesModInfo.addArtifact(HardHatArtifact{
			ContractName: name,
			Abi:          artifact.Abi,
			Bytecode:     artifact.Bytecode.Object,
		})
	}

	// add upgrade executor module which is not compiled locally, but imported from 'nitro-contracts' dependencies
	upgExecutorPath := filepath.Join(parent, "contracts", "node_modules", "@offchainlabs", "upgrade-executor", "build", "contracts", "src", "UpgradeExecutor.sol", "UpgradeExecutor.json")
	_, err = os.Stat(upgExecutorPath)
	if !os.IsNotExist(err) {
		data, err := os.ReadFile(upgExecutorPath)
		if err != nil {
			// log.Fatal(string(output))
			log.Fatal("could not read", upgExecutorPath, "for contract", "UpgradeExecutor", err)
		}
		artifact := HardHatArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", "UpgradeExecutor", err)
		}
		modInfo := modules["upgrade_executorgen"]
		if modInfo == nil {
			modInfo = &moduleInfo{}
			modules["upgrade_executorgen"] = modInfo
		}
		modInfo.addArtifact(artifact)
	}

	for module, info := range modules {

		code, err := abigen.Bind(
			info.contractNames,
			info.abis,
			info.bytecodes,
			nil,
			module,
			nil,
			nil,
		)
		if err != nil {
			log.Fatal(err)
		}

		folder := filepath.Join(root, "go", module)

		err = os.MkdirAll(folder, 0o755)
		if err != nil {
			log.Fatal(err)
		}

		/*
			#nosec G306
			This file contains no private information so the permissions can be lenient
		*/
		err = os.WriteFile(filepath.Join(folder, module+".go"), []byte(code), 0o644)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("successfully generated go abi files")

	blockscout := filepath.Join(parent, "nitro-testnode", "blockscout", "init", "data")
	if _, err := os.Stat(blockscout); err != nil {
		fmt.Println("skipping abi export since blockscout is not present")
	} else {
		modules["precompilesgen"].exportABIs(blockscout)
		modules["node_interfacegen"].exportABIs(blockscout)
		fmt.Println("successfully exported abi files")
	}
}
