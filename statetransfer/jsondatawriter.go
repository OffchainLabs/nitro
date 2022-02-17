package statetransfer

import (
	"encoding/json"
	"os"
)

type JsonListWriter struct {
	file       *os.File
	elementNum int
}

func NewJsonListWriter(filePath string) (*JsonListWriter, error) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return nil, err
	}
	if _, err := file.Write([]byte{'['}); err != nil {
		return nil, err
	}
	return &JsonListWriter{
		file:       file,
		elementNum: 0,
	}, nil
}

func (l *JsonListWriter) Write(elem interface{}) error {
	if l.elementNum > 0 {
		if _, err := l.file.Write([]byte{','}); err != nil {
			return err
		}
	}
	if _, err := l.file.Write([]byte{'\n', ' '}); err != nil {
		return err
	}
	jsonElem, err := json.Marshal(elem)
	if err != nil {
		return err
	}
	if _, err := l.file.Write(jsonElem); err != nil {
		return err
	}
	l.elementNum++
	return nil
}

func (l *JsonListWriter) Close() error {
	if _, err := l.file.Write([]byte{']', '\n'}); err != nil {
		return err
	}
	return l.file.Close()
}
