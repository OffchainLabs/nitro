package statetransfer

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

var ArbTransferListNames = []string{"Blocks", "AddressTableContents", "RetryableData", "Accounts"}

// This is the external inteface to read ArbosInintData.
// Implementations support in-memory data or potentially huge files read element by element.
// Data is saved in lists, in order as seen in ArbTransferListNames.
// When parsing a file all lists must be included (may be empty), in the right order.
// To use:
// r.OpenTopLevel()
// r.NextList()
// for r.More() {
//  r.GetNext*
// }
// r.CloseList
// repeat NextList..CloseList for each list
// r.CloseTopLevel
type InitDataReader interface {
	NextList() (string, error)                                      // opens next list and returns it's name
	GetNextStoredBlock() (*StoredBlock, error)                      // only in list "Blocks"
	GetNextAddress() (*common.Address, error)                       // only in list "AddressTableContents"
	GetNextRetryableData() (*InitializationDataForRetryable, error) // only in list "RetryableData"
	GetNextAccountInit() (*AccountInitializationInfo, error)        // only in list "Accounts"
	CloseList() error
	More() bool
	OpenTopLevel() error
	CloseTopLevel() error
}

type MemoryInitDataReader struct {
	multiListTracker
	data *ArbosInitializationInfo
}

// skipblocks is useful for tests that only initialize a blockchain
func NewMemoryInitDataReader(data *ArbosInitializationInfo, skipBlocks bool) (InitDataReader, error) {
	res := &MemoryInitDataReader{
		multiListTracker: newMultiListTracker(ArbTransferListNames),
		data:             data,
	}
	if skipBlocks {
		if len(data.Blocks) > 0 {
			return nil, errors.New("trying to skip blocks in initdata, but they exist")
		}
		if err := res.OpenTopLevel(); err != nil {
			return nil, err
		}
		if _, err := res.NextList(); err != nil {
			return nil, err
		}
		if err := res.CloseList(); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (m *MemoryInitDataReader) NextList() (string, error) {
	if err := m.canEnterList(); err != nil {
		return "", err
	}
	return m.enterNextList()
}

func (m *MemoryInitDataReader) GetNextStoredBlock() (*StoredBlock, error) {
	if err := m.AssumeInsideList("Blocks"); err != nil {
		return nil, err
	}
	res := &m.data.Blocks[m.posInList]
	m.advInList()
	return res, nil
}

func (m *MemoryInitDataReader) GetNextAddress() (*common.Address, error) {
	if err := m.AssumeInsideList("AddressTableContents"); err != nil {
		return nil, err
	}
	res := &m.data.AddressTableContents[m.posInList]
	m.advInList()
	return res, nil
}

func (m *MemoryInitDataReader) GetNextRetryableData() (*InitializationDataForRetryable, error) {
	if err := m.AssumeInsideList("RetryableData"); err != nil {
		return nil, err
	}
	res := &m.data.RetryableData[m.posInList]
	m.advInList()
	return res, nil
}

func (m *MemoryInitDataReader) GetNextAccountInit() (*AccountInitializationInfo, error) {
	if err := m.AssumeInsideList("Accounts"); err != nil {
		return nil, err
	}
	res := &m.data.Accounts[m.posInList]
	m.advInList()
	return res, nil
}

func (m *MemoryInitDataReader) More() bool {
	listName, err := m.CurListName()
	if err != nil {
		return false
	}
	if listName == "Blocks" {
		return len(m.data.Blocks) > m.posInList
	}
	if listName == "AddressTableContents" {
		return len(m.data.AddressTableContents) > m.posInList
	}
	if listName == "RetryableData" {
		return len(m.data.RetryableData) > m.posInList
	}
	if listName == "Accounts" {
		return len(m.data.Accounts) > m.posInList
	}
	return false
}

func (m *MemoryInitDataReader) CloseList() error {
	if !m.InsideList() {
		return errors.New("not in list")
	}
	m.leaveList()
	return nil
}

func (m *MemoryInitDataReader) OpenTopLevel() error {
	if err := m.canOpenTopLevel(); err != nil {
		return err
	}
	m.openTopLevel()
	return nil
}

func (m *MemoryInitDataReader) CloseTopLevel() error {
	if err := m.canCloseTopLevel(); err != nil {
		return err
	}
	m.closeTopLevel()
	return nil
}

type IterativeInitDataReader struct {
	JsonMultiListReader
}

func NewIterativeInitDataReader(ioreader io.Reader) InitDataReader {
	return &IterativeInitDataReader{
		JsonMultiListReader: *NewJsonMultiListReader(ioreader, ArbTransferListNames),
	}
}

func (i *IterativeInitDataReader) GetNextStoredBlock() (*StoredBlock, error) {
	if err := i.AssumeInsideList("Blocks"); err != nil {
		return nil, err
	}
	var elem StoredBlock
	if err := i.GetNextElement(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (i *IterativeInitDataReader) GetNextAddress() (*common.Address, error) {
	if err := i.AssumeInsideList("AddressTableContents"); err != nil {
		return nil, err
	}
	var elem common.Address
	if err := i.GetNextElement(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (i *IterativeInitDataReader) GetNextRetryableData() (*InitializationDataForRetryable, error) {
	if err := i.AssumeInsideList("RetryableData"); err != nil {
		return nil, err
	}
	var elem InitializationDataForRetryable
	if err := i.GetNextElement(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (i *IterativeInitDataReader) GetNextAccountInit() (*AccountInitializationInfo, error) {
	if err := i.AssumeInsideList("Accounts"); err != nil {
		return nil, err
	}
	var elem AccountInitializationInfo
	if err := i.GetNextElement(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}
