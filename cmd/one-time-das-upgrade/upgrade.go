// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/colors"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("Usage: upgrade <path>")
	}

	path := filepath.FromSlash(args[1])
	info, err := os.Stat(path)
	if err != nil {
		panic(fmt.Sprintf("failed to open directory: %v\n%v", path, err))
	}
	if !info.IsDir() {
		panic(fmt.Sprintf("path %v is not a directory", path))
	}

	println("upgrading das files in directory", path)

	renames := make(map[string]string)

	err = filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			colors.PrintRed("skipping ", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		stem := filepath.Dir(path) + "/"
		name := info.Name()
		zero := false
		if name[:2] == "0x" {
			name = name[2:]
			zero = true
		}

		hashbytes, err := hex.DecodeString(name)
		if err != nil || len(hashbytes) != 32 {
			panic(fmt.Sprintf("filename %v isn't a hash", path))
		}
		hash := *(*common.Hash)(hashbytes)
		tree := dastree.FlatHashToTreeHash(hash)

		contents, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("failed to read file %v %v", path, err))
		}
		if crypto.Keccak256Hash(contents) != hash {
			panic(fmt.Sprintf("file hash %v does not match its contents", path))
		}

		newName := tree.Hex()
		if !zero {
			newName = newName[2:]
		}
		renames[path] = stem + newName
		return nil
	})
	if err != nil {
		panic(err)
	}

	for name, rename := range renames {
		println(name, colors.Grey, "=>", colors.Clear, rename)
		err := os.Rename(name, rename)
		if err != nil {
			panic("failed to mv file")
		}
	}
}
