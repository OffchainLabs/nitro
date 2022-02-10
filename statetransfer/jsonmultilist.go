package statetransfer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Json contains a single top-level dictionary
// Each field in the top-level dictionary is a list
// Order and type of fields is predetermined (listNames)

type multiListTracker struct {
	listNames  []string
	listNumber int
	posInList  int
}

func newMultiListTracker(listNames []string) multiListTracker {
	return multiListTracker{
		listNames:  listNames,
		listNumber: -1,
		posInList:  -1,
	}
}

func (l *multiListTracker) canOpenTopLevel() error {
	if l.listNumber >= 0 {
		return errors.New("toplevel already open")
	}
	return nil
}

func (l *multiListTracker) openTopLevel() {
	l.listNumber = 0
}

func (l *multiListTracker) canCloseTopLevel() error {
	if l.listNumber < 0 {
		return errors.New("toplevel not open")
	}
	return nil
}

func (l *multiListTracker) closeTopLevel() {
	l.listNumber = -1
}

func (l *multiListTracker) nextListName() string {
	return l.listNames[l.listNumber]
}

func (l *multiListTracker) CurListName() (string, error) {
	if l.posInList < 0 {
		return "", errors.New("not in list")
	}
	return l.listNames[l.listNumber-1], nil
}

func (l *multiListTracker) AssumeInsideList(listName string) error {
	curName, err := l.CurListName()
	if err != nil {
		return err
	}
	if curName != listName {
		return fmt.Errorf("not in list %v", listName)
	}
	return nil
}

func (l *multiListTracker) canEnterList() error {
	if l.listNumber < 0 {
		return errors.New("didn't open toplevel")
	}
	if l.listNumber >= len(l.listNames) {
		return errors.New("passed all lists")
	}
	if l.posInList >= 0 {
		return fmt.Errorf("prev list not closed")
	}
	return nil
}

func (l *multiListTracker) enterNextList() (string, error) {
	l.posInList = 0
	l.listNumber++
	return l.CurListName()
}

func (l *multiListTracker) InsideList() bool {
	return (l.posInList >= 0)
}

func (l *multiListTracker) leaveList() {
	l.posInList = -1
}

func (l *multiListTracker) advInList() {
	l.posInList++
}

type JsonMultiListReader struct {
	multiListTracker
	input *json.Decoder
}

func NewJsonMultiListReader(in io.Reader, listNames []string) *JsonMultiListReader {
	var inputJson *json.Decoder
	if in != nil {
		inputJson = json.NewDecoder(in)
	}
	return &JsonMultiListReader{
		multiListTracker: newMultiListTracker(listNames),
		input:            inputJson,
	}
}

func (i *JsonMultiListReader) eatInputDelim(expected json.Delim) error {
	token, err := i.input.Token()
	if err != nil {
		return err
	}
	foundString := "<Not Delim>"
	delim, match := token.(json.Delim)
	if match {
		foundString = delim.String()
	}
	if foundString != expected.String() {
		return fmt.Errorf("expected %s, found: %s", expected, foundString)
	}
	return nil
}

func (i *JsonMultiListReader) eatInputString(expected string) error {
	token, err := i.input.Token()
	if err != nil {
		return err
	}
	foundString, match := token.(string)
	if !match {
		foundString = fmt.Sprintf("<%T: %v>", token, token)
	}
	if foundString != expected {
		return fmt.Errorf("expected %s, found: %s", expected, foundString)
	}
	return nil
}

func (i *JsonMultiListReader) NextList() (string, error) {
	if err := i.canEnterList(); err != nil {
		return "", err
	}
	if err := i.eatInputString(i.nextListName()); err != nil {
		return "", err
	}
	if err := i.eatInputDelim('['); err != nil {
		return "", err
	}
	return i.enterNextList()
}

func (i *JsonMultiListReader) GetNextElement(elem interface{}) error {
	if !i.InsideList() {
		return errors.New("list not open")
	}
	if err := i.input.Decode(elem); err != nil {
		return err
	}
	i.advInList()
	return nil
}

func (i *JsonMultiListReader) More() bool {
	if i == nil || !i.InsideList() {
		return false
	}
	return i.input.More()
}

func (i *JsonMultiListReader) CloseList() error {
	if !i.InsideList() {
		return errors.New("list not open")
	}
	if err := i.eatInputDelim(']'); err != nil {
		return err
	}
	i.leaveList()
	return nil
}

func (i *JsonMultiListReader) OpenTopLevel() error {
	if err := i.canOpenTopLevel(); err != nil {
		return err
	}
	if err := i.eatInputDelim('{'); err != nil {
		return err
	}
	i.openTopLevel()
	return nil
}

func (i *JsonMultiListReader) CloseTopLevel() error {
	if err := i.canCloseTopLevel(); err != nil {
		return err
	}
	if err := i.eatInputDelim('}'); err != nil {
		return err
	}
	i.closeTopLevel()
	return nil
}

type JsonMultiListWriter struct {
	multiListTracker
	output io.Writer
}

func NewJsonMultiListWriter(out io.Writer, listNames []string) *JsonMultiListWriter {
	return &JsonMultiListWriter{
		output:           out,
		multiListTracker: newMultiListTracker(listNames),
	}
}

func (i *JsonMultiListWriter) NextList() (string, error) {
	if err := i.canEnterList(); err != nil {
		return "", err
	}
	if i.listNumber > 0 {
		if _, err := i.output.Write([]byte{','}); err != nil {
			return "", err
		}
	}
	if _, err := i.output.Write([]byte{'\n'}); err != nil {
		return "", err
	}
	nameMarshalled, err := json.Marshal(i.nextListName())
	if err != nil {
		return "", err
	}
	if _, err = i.output.Write(nameMarshalled); err != nil {
		return "", err
	}
	if _, err = i.output.Write([]byte{':', ' ', '['}); err != nil {
		return "", err
	}
	return i.enterNextList()
}

func (i *JsonMultiListWriter) AddElement(elem interface{}) error {
	if !i.InsideList() {
		return fmt.Errorf("not in list")
	}
	elemMarshalled, err := json.Marshal(elem)
	if err != nil {
		return err
	}
	if i.posInList > 0 {
		_, err = i.output.Write([]byte{','})
		if err != nil {
			return err
		}
	}
	if _, err = i.output.Write([]byte{'\n', ' ', ' '}); err != nil {
		return err
	}
	if _, err = i.output.Write(elemMarshalled); err != nil {
		return err
	}
	i.advInList()
	return nil
}

func (i *JsonMultiListWriter) CloseList() error {
	if !i.InsideList() {
		return fmt.Errorf("cannot close list while not in list")
	}
	if _, err := i.output.Write([]byte{']'}); err != nil {
		return err
	}
	i.leaveList()
	return nil
}

func (i *JsonMultiListWriter) OpenTopLevel() error {
	if err := i.canOpenTopLevel(); err != nil {
		return err
	}
	if _, err := i.output.Write([]byte{'{'}); err != nil {
		return err
	}
	i.openTopLevel()
	return nil
}

func (i *JsonMultiListWriter) CloseTopLevel() error {
	if err := i.canCloseTopLevel(); err != nil {
		return err
	}
	if _, err := i.output.Write([]byte{'\n', '}', '\n'}); err != nil {
		return err
	}
	i.closeTopLevel()
	return nil
}

type JsonMultiListUpdater struct {
	Reader *JsonMultiListReader
	Writer *JsonMultiListWriter
}

func (i *JsonMultiListUpdater) NextList() (string, error) {
	var readerList string
	if i.Reader != nil {
		var err error
		readerList, err = i.Reader.NextList()
		if err != nil {
			return "", err
		}
	}
	writerList, err := i.Writer.NextList()
	if err != nil {
		return "", err
	}
	if i.Reader != nil && readerList != writerList {
		return "", fmt.Errorf("list names mismatch reader: %v writer: %v", readerList, writerList)
	}
	return writerList, nil
}

func (i *JsonMultiListUpdater) CloseList() error {
	if i.Reader != nil {
		err := i.Reader.CloseList()
		if err != nil {
			return err
		}
	}
	return i.Writer.CloseList()
}

func (i *JsonMultiListUpdater) OpenTopLevel() error {
	if i.Reader != nil {
		err := i.Reader.OpenTopLevel()
		if err != nil {
			return err
		}
	}
	return i.Writer.OpenTopLevel()
}

func (i *JsonMultiListUpdater) CloseTopLevel() error {
	if i.Reader != nil {
		err := i.Reader.CloseTopLevel()
		if err != nil {
			return err
		}
	}
	return i.Writer.CloseTopLevel()
}
