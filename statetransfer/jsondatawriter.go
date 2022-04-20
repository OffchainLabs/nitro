// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"encoding/json"
	"os"
)

type JsonListWriter struct {
	file       *os.File
	elementNum int
}

func NewJsonListWriter(filePath string, append bool) (*JsonListWriter, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	}
	file, err := os.OpenFile(filePath, flags, 0664)
	if err != nil {
		return nil, err
	}
	return &JsonListWriter{
		file:       file,
		elementNum: 0,
	}, nil
}

func (l *JsonListWriter) Write(elem interface{}) error {
	jsonElem, err := json.Marshal(elem)
	if err != nil {
		return err
	}
	jsonElem = append(jsonElem, '\n')
	if _, err := l.file.Write(jsonElem); err != nil {
		return err
	}
	l.elementNum++
	return nil
}

func (l *JsonListWriter) Close() error {
	return l.file.Close()
}
