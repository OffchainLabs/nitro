// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/pkg/errors"
)

type JitMachine struct {
	binary  string
	process *exec.Cmd
	stdin   io.WriteCloser
}

func createJitMachine(config NitroMachineConfig, moduleRoot common.Hash, fatalErrChan chan error) (*JitMachine, error) {

	jitBinary, err := exec.LookPath("jit")
	if err != nil {
		jitBinary = filepath.FromSlash("./target/bin/jit")
	}
	if _, err := os.Stat(jitBinary); err != nil {
		return nil, err
	}

	binary := filepath.Join(config.getMachinePath(moduleRoot), config.ProverBinPath)
	invocation := []string{"--binary", binary, "--forks"}
	if config.JitCranelift {
		invocation = append(invocation, "--cranelift")
	}
	process := exec.Command(jitBinary, invocation...)
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, err
	}
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	go func() {
		if err := process.Run(); err != nil {
			fatalErrChan <- fmt.Errorf("Lost jit block validator process: %w", err)
		}
	}()

	machine := &JitMachine{
		binary:  binary,
		process: process,
		stdin:   stdin,
	}
	return machine, nil
}

func (machine *JitMachine) prove(
	entry *validationEntry, resolver GoPreimageResolver, delayed []byte,
) (GoGlobalState, error) {
	state := GoGlobalState{}

	timeout := time.Now().Add(60 * time.Second)
	tcp, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		return state, err
	}
	if err := tcp.SetDeadline(timeout); err != nil {
		return state, err
	}
	defer tcp.Close()
	address := fmt.Sprintf("%v\n", tcp.Addr().String())

	// Tell the spawner process about the new tcp port
	if _, err := machine.stdin.Write([]byte(address)); err != nil {
		return state, err
	}

	// Wait for the forked process to connect
	conn, err := tcp.Accept()
	if err != nil {
		return state, err
	}
	if err := conn.SetReadDeadline(timeout); err != nil {
		return state, err
	}
	if err := conn.SetWriteDeadline(timeout); err != nil {
		return state, err
	}
	defer conn.Close()

	// Tell the new process about the global state
	gsStart := entry.start()

	writeExact := func(data []byte) error {
		_, err := conn.Write(data)
		return err
	}
	writeUint8 := func(data uint8) error {
		return writeExact([]byte{data})
	}
	writeUint64 := func(data uint64) error {
		return writeExact(arbmath.UintToBytes(data))
	}
	writeBytes := func(data []byte) error {
		if err := writeUint64(uint64(len(data))); err != nil {
			return err
		}
		return writeExact(data)
	}

	// send global state
	if err := writeUint64(gsStart.Batch); err != nil {
		return state, err
	}
	if err := writeUint64(gsStart.PosInBatch); err != nil {
		return state, err
	}
	if err := writeExact(gsStart.BlockHash[:]); err != nil {
		return state, err
	}
	if err := writeExact(gsStart.SendRoot[:]); err != nil {
		return state, err
	}

	const successByte = 0x0
	const failureByte = 0x1
	const preimageByte = 0x2
	const anotherByte = 0x3
	const readyByte = 0x4

	success := []byte{successByte}
	another := []byte{anotherByte}
	ready := []byte{successByte, readyByte}

	// send inbox
	for _, batch := range entry.BatchInfo {
		if err := writeExact(another); err != nil {
			return state, err
		}
		if err := writeUint64(batch.Number); err != nil {
			return state, err
		}
		if err := writeBytes(batch.Data); err != nil {
			return state, err
		}
	}
	if err := writeExact(success); err != nil {
		return state, err
	}

	// send delayed inbox
	if entry.HasDelayedMsg {
		if err := writeExact(another); err != nil {
			return state, err
		}
		if err := writeUint64(entry.DelayedMsgNr); err != nil {
			return state, err
		}
		if err := writeBytes(delayed); err != nil {
			return state, err
		}
	}
	if err := writeExact(ready); err != nil {
		return state, err
	}

	read := func(count uint64) ([]byte, error) {
		slice := make([]byte, count)
		_, err := io.ReadFull(conn, slice)
		if err != nil {
			return nil, err
		}
		return slice, nil
	}
	readUint64 := func() (uint64, error) {
		slice, err := read(8)
		if err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint64(slice), nil
	}
	readHash := func() (common.Hash, error) {
		slice, err := read(32)
		if err != nil {
			return common.Hash{}, err
		}
		return common.BytesToHash(slice), nil
	}

	for {
		kind, err := read(1)
		if err != nil {
			return state, err
		}
		switch kind[0] {
		case preimageByte:
			hash, err := readHash()
			if err != nil {
				return state, err
			}
			preimage, err := resolver(hash)
			if err != nil {
				log.Error("Failed to resolve preimage for jit", "hash", hash)
				if err := writeUint8(failureByte); err != nil {
					return state, err
				}
				continue
			}

			// send the preimage
			if err := writeUint8(successByte); err != nil {
				return state, err
			}
			if err := writeBytes(preimage); err != nil {
				return state, err
			}

		case failureByte:
			length, err := readUint64()
			if err != nil {
				return state, err
			}
			message, err := read(length)
			if err != nil {
				return state, err
			}
			log.Error("Jit Machine Failure", "message", string(message))
			return state, errors.New(string(message))
		case successByte:
			if state.Batch, err = readUint64(); err != nil {
				return state, err
			}
			if state.PosInBatch, err = readUint64(); err != nil {
				return state, err
			}
			if state.BlockHash, err = readHash(); err != nil {
				return state, err
			}
			state.SendRoot, err = readHash()
			return state, err
		default:
			message := "inter-process communication failure"
			log.Error("Jit Machine Failure", "message", message)
			return state, errors.New("inter-process communication failure")
		}
	}
}
