package melrecording

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

type TxsAndReceiptsDatabase struct {
	underlying *triedb.Database
	recorder   daprovider.PreimageRecorder
}

func (rdb *TxsAndReceiptsDatabase) Get(key []byte) ([]byte, error) {
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
func (rdb *TxsAndReceiptsDatabase) Has(key []byte) (bool, error) {
	hash := common.BytesToHash(key)
	_, err := rdb.underlying.Node(hash)
	return err == nil, nil
}
func (rdb *TxsAndReceiptsDatabase) Put(key []byte, value []byte) error {
	return fmt.Errorf("Put not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Delete(key []byte) error {
	return fmt.Errorf("Delete not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) DeleteRange(start, end []byte) error {
	return fmt.Errorf("DeleteRange not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, fmt.Errorf("ReadAncients not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) SyncAncient() error {
	return fmt.Errorf("SyncAncient not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) TruncateHead(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateHead not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) TruncateTail(n uint64) (uint64, error) {
	return 0, fmt.Errorf("TruncateTail not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Append(kind string, number uint64, item interface{}) error {
	return fmt.Errorf("Append not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) AppendRaw(kind string, number uint64, item []byte) error {
	return fmt.Errorf("AppendRaw not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) AncientDatadir() (string, error) {
	return "", fmt.Errorf("AncientDatadir not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, fmt.Errorf("Ancient not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, fmt.Errorf("AncientRange not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, fmt.Errorf("AncientBytes not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Ancients() (uint64, error) {
	return 0, fmt.Errorf("Ancients not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Tail() (uint64, error) {
	return 0, fmt.Errorf("Tail not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) AncientSize(kind string) (uint64, error) {
	return 0, fmt.Errorf("AncientSize not supported on recording DB")
}
func (rdb *TxsAndReceiptsDatabase) Compact(start []byte, limit []byte) error {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) SyncKeyValue() error {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) Stat() (string, error) {
	return "", nil
}
func (rdb *TxsAndReceiptsDatabase) WasmDataBase() ethdb.KeyValueStore {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) NewBatch() ethdb.Batch {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return nil
}
func (rdb *TxsAndReceiptsDatabase) Close() error {
	return nil
}
