package statetransfer

import (
	"encoding/json"
	"fmt"
	"io"
)

// json contains a single dictionary
// each field in the distionary is a list
// list order is known to the user

type IterativeJsonReader struct {
	input *json.Decoder
}

func NewIterativeJsonReader(in io.Reader) *IterativeJsonReader {
	var inputJson *json.Decoder
	if in != nil {
		inputJson = json.NewDecoder(in)
	}
	return &IterativeJsonReader{
		input: inputJson,
	}
}

func (i *IterativeJsonReader) eatInputDelim(expected json.Delim) error {
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

func (i *IterativeJsonReader) eatInputString(expected string) error {
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

func (i *IterativeJsonReader) OpenSubList(listName string) error {
	err := i.eatInputString(listName)
	if err != nil {
		return err
	}

	err = i.eatInputDelim('[')
	if err != nil {
		return err
	}
	return nil
}

func (i *IterativeJsonReader) GetNextElement(elem interface{}) error {
	return i.input.Decode(elem)
}

func (i *IterativeJsonReader) More() bool {
	if i == nil {
		return false
	}
	return i.input.More()
}

func (i *IterativeJsonReader) CloseList() error {
	err := i.eatInputDelim(']')
	if err != nil {
		return err
	}
	return nil
}

func (i *IterativeJsonReader) OpenTopLevel() error {
	err := i.eatInputDelim('{')
	if err != nil {
		return err
	}
	return nil
}

func (i *IterativeJsonReader) CloseTopLevel() error {
	err := i.eatInputDelim('}')
	if err != nil {
		return err
	}
	return nil
}

type IterativeJsonWriter struct {
	output io.Writer

	listcount          int
	curentListLocation int
}

func NewIterativeJsonWriter(out io.Writer) *IterativeJsonWriter {
	return &IterativeJsonWriter{
		output:             out,
		curentListLocation: -1,
	}
}

func (i *IterativeJsonWriter) CreateSubList(listName string) error {
	if i.curentListLocation >= 0 {
		return fmt.Errorf("cannot start list while in list")
	}
	if i.listcount > 0 {
		_, err := i.output.Write([]byte{','})
		if err != nil {
			return err
		}
	}
	_, err := i.output.Write([]byte{'\n'})
	if err != nil {
		return err
	}
	nameMarshalled, err := json.Marshal(listName)
	if err != nil {
		return err
	}
	_, err = i.output.Write(nameMarshalled)
	if err != nil {
		return err
	}
	_, err = i.output.Write([]byte{':', ' ', '['})
	if err != nil {
		return err
	}
	i.curentListLocation = 0
	i.listcount++
	return nil
}

func (i *IterativeJsonWriter) AddElement(elem interface{}) error {
	if i.curentListLocation < 0 {
		return fmt.Errorf("not in list")
	}
	elemMarshalled, err := json.Marshal(elem)
	if err != nil {
		return err
	}
	if i.curentListLocation > 0 {
		_, err = i.output.Write([]byte{','})
		if err != nil {
			return err
		}
	}
	_, err = i.output.Write([]byte{'\n', ' ', ' '})
	if err != nil {
		return err
	}
	_, err = i.output.Write(elemMarshalled)
	if err != nil {
		return err
	}
	i.curentListLocation++
	return nil
}

func (i *IterativeJsonWriter) CloseList() error {
	if i.curentListLocation < 0 {
		return fmt.Errorf("cannot close list while not in list")
	}
	_, err := i.output.Write([]byte{']'})
	if err != nil {
		return err
	}
	i.curentListLocation = -1
	return nil
}

func (i *IterativeJsonWriter) OpenTopLevel() error {
	_, err := i.output.Write([]byte{'{'})
	if err != nil {
		return err
	}
	i.curentListLocation = -1
	return nil
}

func (i *IterativeJsonWriter) CloseTopLevel() error {
	_, err := i.output.Write([]byte{'\n', '}', '\n'})
	if err != nil {
		return err
	}
	i.curentListLocation = -1
	return nil
}

type IterativeJsonUpdater struct {
	Reader *IterativeJsonReader
	Writer *IterativeJsonWriter
}

func (i *IterativeJsonUpdater) StartSubList(listName string) error {
	if i.Reader != nil {
		err := i.Reader.OpenSubList(listName)
		if err != nil {
			return err
		}
	}
	return i.Writer.CreateSubList(listName)
}

func (i *IterativeJsonUpdater) CloseList() error {
	if i.Reader != nil {
		err := i.Reader.CloseList()
		if err != nil {
			return err
		}
	}
	return i.Writer.CloseList()
}

func (i *IterativeJsonUpdater) OpenTopLevel() error {
	if i.Reader != nil {
		err := i.Reader.OpenTopLevel()
		if err != nil {
			return err
		}
	}
	return i.Writer.OpenTopLevel()
}

func (i *IterativeJsonUpdater) CloseTopLevel() error {
	if i.Reader != nil {
		err := i.Reader.CloseTopLevel()
		if err != nil {
			return err
		}
	}
	return i.Writer.CloseTopLevel()
}
