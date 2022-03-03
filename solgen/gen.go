//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type HardHatArtifact struct {
	Format       string        `json:"_format"`
	ContractName string        `json:"contractName"`
	SourceName   string        `json:"sourceName"`
	Abi          []interface{} `json:"abi"`
	Bytecode     string        `json:"bytecode"`
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

func main() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("bad path")
	}
	root := filepath.Dir(filename)
	filePaths, err := filepath.Glob(filepath.Join(root, "build", "contracts", "src", "*", "*", "*.json"))
	if err != nil {
		log.Fatal(err)
	}

	modules := make(map[string]*moduleInfo)

	for _, path := range filePaths {
		if strings.Contains(path, ".dbg.json") {
			continue
		}

		dir, file := filepath.Split(path)
		dir, _ = filepath.Split(dir[:len(dir)-1])
		_, module := filepath.Split(dir[:len(dir)-1])
		module += "gen"

		name := file[:len(file)-5]

		data, err := ioutil.ReadFile(path)
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

	for module, info := range modules {
		code, err := bind.Bind(
			info.contractNames,
			info.abis,
			info.bytecodes,
			nil,
			module,
			bind.LangGo,
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
		err = ioutil.WriteFile(filepath.Join(folder, module+".go"), []byte(code), 0o644)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("successfully generated go abi files")
}
