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
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/pkg/errors"
)

type JitMachine struct {
	binary  string
	process *exec.Cmd
	stdin   io.WriteCloser
}

func createJitMachine(config NitroMachineConfig, moduleRoot common.Hash) (*JitMachine, error) {

	jitBinary := filepath.FromSlash("./arbitrator/target/release/jit")
	if _, err := os.Stat(jitBinary); err != nil {
		return nil, err
	}

	binary := filepath.Join(config.getMachinePath(moduleRoot), config.ProverBinPath)
	process := exec.Command(jitBinary, "--binary", binary, "--forks", "--cranelift")
	stdin, err := process.StdinPipe()
	if err != nil {
		return nil, err
	}
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	if err := process.Start(); err != nil {
		return nil, err
	}

	colors.PrintBlue("Created: ", jitBinary)

	machine := &JitMachine{
		binary:  binary,
		process: process,
		stdin:   stdin,
	}
	return machine, nil
}

func (machine *JitMachine) prove(entry *validationEntry, delayed []byte) (GoGlobalState, error) {
	empty := GoGlobalState{}

	timeout := time.Now().Add(10 * time.Second)

	tcp, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		return empty, err
	}
	tcp.SetDeadline(timeout)
	defer tcp.Close()
	address := fmt.Sprintf("%v\n", tcp.Addr().String())

	// Tell the spawner process about the new tcp port
	colors.PrintBlue("Writing to standard in ", address)
	if _, err := machine.stdin.Write([]byte(address)); err != nil {
		return empty, err
	}

	// Wait for the forked process to connect
	colors.PrintBlue("Waiting for tcp on ", address)
	conn, err := tcp.Accept()
	if err != nil {
		return empty, err
	}
	conn.SetReadDeadline(timeout)
	conn.SetWriteDeadline(timeout)
	defer conn.Close()

	// Tell the new process about the global state
	gsStart := entry.start()
	colors.PrintBlue("Sending global state")

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
		return empty, err
	}
	if err := writeUint64(gsStart.PosInBatch); err != nil {
		return empty, err
	}
	if err := writeExact(gsStart.BlockHash[:]); err != nil {
		return empty, err
	}
	if err := writeExact(gsStart.SendRoot[:]); err != nil {
		return empty, err
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
			return empty, err
		}
		if err := writeUint64(batch.Number); err != nil {
			return empty, err
		}
		if err := writeBytes(batch.Data); err != nil {
			return empty, err
		}
	}
	if _, err := conn.Write(success); err != nil {
		return empty, err
	}

	// send delayed inbox
	if entry.HasDelayedMsg {
		if err := writeExact(another); err != nil {
			return empty, err
		}
		if err := writeUint64(entry.DelayedMsgNr); err != nil {
			return empty, err
		}
		if err := writeBytes(delayed); err != nil {
			return empty, err
		}
	}
	if _, err := conn.Write(ready); err != nil {
		return empty, err
	}

	read := func(count uint64) ([]byte, error) {
		slice := make([]byte, count)
		_, err := conn.Read(slice)
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

	for {
		kind, err := read(1)
		if err != nil {
			return empty, err
		}
		switch kind[0] {
		case preimageByte:
			colors.PrintBlue("Got preimage request")
			hash, err := read(32)
			if err != nil {
				return empty, err
			}
			colors.PrintBlue("Supplying preimage 0x", common.Bytes2Hex(hash), " ", len(hash))
			_ = hash

			// no hash found
			if err := writeUint8(0x01); err != nil {
				return empty, err
			}
		case failureByte:
			colors.PrintRed("Machine failed")
			length, err := readUint64()
			if err != nil {
				return empty, err
			}
			message, err := read(length)
			if err != nil {
				return empty, err
			}
			log.Error("Jit Machine Failure", "message", string(message))
			return empty, errors.New(string(message))
		case successByte:
			colors.PrintMint("Got success")
		default:
			message := "inter-process communication failure"
			log.Error("Jit Machine Failure", "message", message)
			return empty, errors.New("inter-process communication failure")
		}
	}
}
