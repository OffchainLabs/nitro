package melrecording

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

type TxAndLogsDatabase struct {
	underlying *triedb.Database
	recorder   daprovider.PreimageRecorder
}

func (rdb *TxAndLogsDatabase) Get(key []byte) ([]byte, error) {
	hash := common.BytesToHash(key)
	value, err := rdb.underlying.Node(hash)
	if err != nil {
		return nil, err
	}
	if rdb.recorder != nil {
		rdb.recorder(hash, value, arbutil.Keccak256PreimageType)
	}

	return value, nil
}
func (rdb *TxAndLogsDatabase) Has(key []byte) (bool, error) {
	hash := common.BytesToHash(key)
	_, err := rdb.underlying.Node(hash)
	return err == nil, nil
}
func (rdb *TxAndLogsDatabase) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) SyncAncient() error {
	return fmt.Errorf("SyncAncient not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) TruncateHead(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateHead not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) TruncateTail(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateTail not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Append(kind string, number uint64, item interface{}) error {
	return fmt.Errorf("Append not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) AppendRaw(kind string, number uint64, item []byte) error {
	return fmt.Errorf("AppendRaw not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) AncientDatadir() (string, error) {
	return "", fmt.Errorf("AncientDatadir not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, fmt.Errorf("Ancient not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, fmt.Errorf("AncientRange not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, fmt.Errorf("AncientBytes not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Ancients() (uint64, error) {
	return 0, fmt.Errorf("Ancients not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Tail() (uint64, error) {
	return 0, fmt.Errorf("Tail not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) AncientSize(kind string) (uint64, error) {
	return 0, fmt.Errorf("AncientSize not supported on recording DB")
}
func (rdb *TxAndLogsDatabase) Compact(start []byte, limit []byte) error {
	return nil
}
func (rdb *TxAndLogsDatabase) SyncKeyValue() error {
	return nil
}
func (rdb *TxAndLogsDatabase) Stat() (string, error) {
	return "", nil
}
func (rdb *TxAndLogsDatabase) WasmDataBase() ethdb.KeyValueStore {
	return nil
}
func (rdb *TxAndLogsDatabase) NewBatch() ethdb.Batch {
	return nil
}
func (rdb *TxAndLogsDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return nil
}
func (rdb *TxAndLogsDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return nil
}
func (rdb *TxAndLogsDatabase) Close() error {
	return nil
}
