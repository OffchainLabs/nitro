package server_api

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func TestPreimagesMapJson(t *testing.T) {
	t.Parallel()
	for _, preimages := range []PreimagesMapJson{
		{},
		{make(map[common.Hash][]byte)},
		{map[common.Hash][]byte{
			{}: {},
		}},
		{map[common.Hash][]byte{
			{1}: {1},
			{2}: {1, 2},
			{3}: {1, 2, 3},
		}},
	} {
		t.Run(fmt.Sprintf("%v preimages", len(preimages.Map)), func(t *testing.T) {
			// These test cases are fast enough that t.Parallel() probably isn't worth it
			serialized, err := preimages.MarshalJSON()
			Require(t, err, "Failed to marshal preimagesj")

			// Make sure that `serialized` is a valid JSON map
			stringMap := make(map[string]string)
			err = json.Unmarshal(serialized, &stringMap)
			Require(t, err, "Failed to unmarshal preimages as string map")
			if len(stringMap) != len(preimages.Map) {
				t.Errorf("Got %v entries in string map but only had %v preimages", len(stringMap), len(preimages.Map))
			}

			var deserialized PreimagesMapJson
			err = deserialized.UnmarshalJSON(serialized)
			Require(t, err)

			if (len(preimages.Map) > 0 || len(deserialized.Map) > 0) && !reflect.DeepEqual(preimages, deserialized) {
				t.Errorf("Preimages map %v serialized to %v but then deserialized to different map %v", preimages, string(serialized), deserialized)
			}
		})
	}
}

func mapParity(input map[common.Hash][]byte) int64 {
	valTot := int64(0)
	for k, v := range input {
		valBuf := int64(0)
		for _, b := range v {
			valBuf += int64(b)
		}
		valKey := int64(0)
		for _, b := range k {
			valKey += int64(b)
		}
		valKey += valKey * valBuf
	}
	return valTot
}

func mapToBytes(input map[common.Hash][]byte) []byte {
	length := 8
	for _, b := range input {
		length += len(b) + 32 + 8
	}
	output := make([]byte, length)
	i := 0
	binary.BigEndian.PutUint64(output[i:], uint64(len(input)))
	i += 8
	for hash, buf := range input {
		copy(output[i:], hash[:])
		i += 32
		binary.BigEndian.PutUint64(output[i:], uint64(len(buf)))
		i += 8
		copy(output[i:], buf[:])
		i += len(buf)
	}
	return output
}

func bytesTpMap(input []byte) (map[common.Hash][]byte, error) {
	mapLength := int(binary.BigEndian.Uint64(input[0:8]))
	i := 8
	output := make(map[common.Hash][]byte, mapLength)
	for i < len(input) {
		if len(input)-i < 40 {
			return nil, errors.New("unexpected bytes left")
		}
		hash := common.Hash{}
		copy(hash[:], input[i:i+32])
		i += 32
		length := int(binary.BigEndian.Uint64(input[i : i+8]))
		i += 8
		if len(input)-i < length {
			return nil, errors.New("bad buffer size")
		}
		output[hash] = input[i : i+length]
		i += length
	}
	return output, nil
}

type mockAPI struct {
}

func (m *mockAPI) Return1(ctx context.Context) (int64, error) {
	return 1, nil
}

func (m *mockAPI) HandRolled(ctx context.Context, preimages *PreimagesMapJson) (int64, error) {
	parity := mapParity(preimages.Map)
	return parity, nil
}

func (m *mockAPI) Simple(ctx context.Context, preimages map[common.Hash][]byte) (int64, error) {
	parity := mapParity(preimages)
	return parity, nil
}

func (m *mockAPI) B64Map(ctx context.Context, preimages_b64 map[string]string) (int64, error) {
	preimages := make(map[common.Hash][]byte)
	for encHash, encData := range preimages_b64 {
		hash, err := base64.StdEncoding.DecodeString(encHash)
		if err != nil {
			return 0, err
		}
		data, err := base64.StdEncoding.DecodeString(encData)
		if err != nil {
			return 0, err
		}
		preimages[common.BytesToHash(hash)] = data
	}
	parity := mapParity(preimages)
	return parity, nil
}

func (m *mockAPI) MapAsBytes(ctx context.Context, input []byte) (int64, error) {
	preimages, err := bytesTpMap(input)
	if err != nil {
		return 0, err
	}
	parity := mapParity(preimages)
	return parity, nil
}

func createMockAPINode(ctx context.Context) (*node.Node, error) {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{"test"}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	if err != nil {
		return nil, err
	}

	valAPIs := []rpc.API{{
		Namespace:     "test",
		Version:       "1.0",
		Service:       &mockAPI{},
		Public:        true,
		Authenticated: false,
	}}
	stack.RegisterAPIs(valAPIs)

	err = stack.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		stack.Close()
	}()

	return stack, nil
}

func setupPreimages(n int) ([]map[common.Hash][]byte, int) {
	totalSize := 0
	currentImageTotal := 0
	prand := testhelpers.NewPseudoRandomDataSource(nil, 1)
	res := make([]map[common.Hash][]byte, 0)
	preimages := make(map[common.Hash][]byte)
	for i := 0; i < n; i++ {
		if currentImageTotal > 1<<21 {
			res = append(res, preimages)
			preimages = make(map[common.Hash][]byte)
			currentImageTotal = 0
			// if len(res) >= n {
			// 	break
			// }
		}
		size := prand.GetIntRange(32, 1024)
		preimages[prand.GetHash()] = prand.GetData(size)
		totalSize += size + 32
		currentImageTotal += size + 32
	}
	res = append(res, preimages)
	return res, totalSize
}

func BenchmarkHandRolled(b *testing.B) {
	benchmarkSends(b, func(ctx context.Context, client *rpc.Client, preimages map[common.Hash][]byte) error {
		var res int64
		err := client.CallContext(ctx, &res, "test_handRolled", &PreimagesMapJson{preimages})
		if err != nil {
			return nil
		}
		parity := mapParity(preimages)
		if parity != res {
			return errors.New("unexpected parity")
		}
		return nil
	})
}

func BenchmarkSimple(b *testing.B) {
	benchmarkSends(b, func(ctx context.Context, client *rpc.Client, preimages map[common.Hash][]byte) error {
		var res int64
		err := client.CallContext(ctx, &res, "test_simple", preimages)
		if err != nil {
			return nil
		}
		parity := mapParity(preimages)
		if parity != res {
			return errors.New("unexpected parity")
		}
		return nil
	})
}

func BenchmarkMapToB64(b *testing.B) {
	benchmarkSends(b, func(ctx context.Context, client *rpc.Client, preimages map[common.Hash][]byte) error {
		mapOut := make(map[string]string)
		for hash, data := range preimages {
			encHash := base64.StdEncoding.EncodeToString(hash.Bytes())
			encData := base64.StdEncoding.EncodeToString(data)
			mapOut[encHash] = encData
		}
		var res int64
		err := client.CallContext(ctx, &res, "test_b64Map", mapOut)
		if err != nil {
			return err
		}
		parity := mapParity(preimages)
		if parity != res {
			return errors.New("unexpected parity")
		}
		return nil
	})
}

func BenchmarkMapToBytes(b *testing.B) {
	benchmarkSends(b, func(ctx context.Context, client *rpc.Client, preimages map[common.Hash][]byte) error {
		var res int64
		err := client.CallContext(ctx, &res, "test_mapAsBytes", mapToBytes(preimages))
		if err != nil {
			return err
		}
		parity := mapParity(preimages)
		if parity != res {
			return errors.New("unexpected parity")
		}
		return nil
	})
}

func benchmarkSends(b *testing.B, oneSend func(context.Context, *rpc.Client, map[common.Hash][]byte) error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	preimageArr, totalSize := setupPreimages(b.N)

	server, err := createMockAPINode(ctx)
	if err != nil {
		b.Fatal(err)
	}
	wsclient, err := rpc.Dial(server.WSEndpoint())
	if err != nil {
		b.Fatal(err)
	}
	var res int64
	err = wsclient.CallContext(ctx, &res, "test_return1")
	if err != nil {
		b.Fatal(err)
	}
	if res != 1 {
		b.Fatal("not 1: ", res)
	}
	b.ResetTimer()
	for _, preimages := range preimageArr {
		err := oneSend(ctx, wsclient, preimages)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(preimageArr))/float64(b.N), "sends/op")
	b.ReportMetric(float64(totalSize)/float64(b.N), "bytes/op")
}

func TestMain(m *testing.M) {
	logLevelEnv := os.Getenv("TEST_LOGLEVEL")
	if logLevelEnv != "" {
		logLevel, err := strconv.ParseUint(logLevelEnv, 10, 32)
		if err != nil || logLevel > uint64(log.LvlTrace) {
			log.Warn("TEST_LOGLEVEL exists but out of bound, ignoring", "logLevel", logLevelEnv, "max", log.LvlTrace)
		}
		glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
		glogger.Verbosity(log.Lvl(logLevel))
		log.Root().SetHandler(glogger)
	}
	code := m.Run()
	os.Exit(code)
}
