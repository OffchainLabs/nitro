//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
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

func main() {
	filePaths, err := filepath.Glob("./precompiles/artifacts/src/*/*.json")
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range filePaths {
		if strings.Contains(path, ".dbg.json") {
			continue
		}

		name := path[strings.LastIndex(path, "/")+1 : len(path)-5]

		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal("could not read", path, "for contract", name, err)
		}

		artifact := HardHatArtifact{}
		if err := json.Unmarshal(data, &artifact); err != nil {
			log.Fatal("failed to parse contract", name, err)
		}
		abi, err := json.Marshal(artifact.Abi)
		if err != nil {
			log.Fatal(err)
		}

		code, err := bind.Bind(
			[]string{artifact.ContractName},
			[]string{string(abi)},
			[]string{artifact.Bytecode},
			nil,
			"precompiles",
			bind.LangGo,
			nil,
			nil,
		)
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile("./precompiles/go/"+name+".go", []byte(code), 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("successfully generated", len(filePaths)/2, "precompiles")
}
