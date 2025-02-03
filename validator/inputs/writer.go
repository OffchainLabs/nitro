package inputs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/offchainlabs/nitro/validator/server_api"
)

// Writer is a configurable writer of InputJSON files.
//
// The default Writer will write to a path like:
//
//	$HOME/.arbuitrum/validation-inputs/<YYYMMDD_HHMMSS>/block_inputs_<id>.json
//
// The path can be nested under a slug directory so callers can provide a
// recognizable name to differentiate various contexts in which the InputJSON
// is being written. If the Writer is configured by calling WithSlug, then the
// path will be like:
//
//	$HOME/.arbuitrum/validation-inputs/<slug>/<YYYMMDD_HHMMSS>/block_inputs_<id>.json
//
// The inclusion of BlockId in the file's name is on by default, however that can be disabled
// by calling WithBlockIdInFileNameEnabled(false). In which case, the path will be like:
//
//	$HOME/.arbuitrum/validation-inputs/<slug>/<YYYMMDD_HHMMSS>/block_inputs.json
//
// The inclusion of a timestamp directory is on by default to avoid conflicts which
// would result in files being overwritten. However, the Writer can be configured
// to not use a timestamp directory.  If the Writer is configured by calling
// WithTimestampDirEnabled(false), then the path will be like:
//
//	$HOME/.arbuitrum/validation-inputs/<slug>/block_inputs_<id>.json
//
// Finally, to give complete control to the clients, the base directory can be
// set directly with WithBaseDir. In which case, the path will be like:
//
//	<baseDir>/block_inputs_<id>.json
//	  or
//	<baseDir>/<slug>/block_inputs_<id>.json
//	  or
//	<baseDir>/<slug>/<YYYMMDD_HHMMSS>/block_inputs_<id>.json
type Writer struct {
	clock                Clock
	baseDir              string
	slug                 string
	useTimestampDir      bool
	useBlockIdInFileName bool
}

// WriterOption is a function that configures a Writer.
type WriterOption func(*Writer)

// Clock is an interface for getting the current time.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

// NewWriter creates a new Writer with default settings.
func NewWriter(options ...WriterOption) (*Writer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(homeDir, ".arbitrum", "validation-inputs")
	w := &Writer{
		clock:                realClock{},
		baseDir:              baseDir,
		slug:                 "",
		useTimestampDir:      true,
		useBlockIdInFileName: true,
	}
	for _, o := range options {
		o(w)
	}
	return w, nil
}

// withTestClock configures the Writer to use the given clock.
//
// This is only intended for testing.
func withTestClock(clock Clock) WriterOption {
	return func(w *Writer) {
		w.clock = clock
	}
}

// WithSlug configures the Writer to use the given slug as a directory name.
func WithSlug(slug string) WriterOption {
	return func(w *Writer) {
		w.slug = slug
	}
}

// WithoutSlug clears the slug configuration.
//
// This is equivalent to the WithSlug("") option but is more readable.
func WithoutSlug() WriterOption {
	return WithSlug("")
}

// WithBaseDir configures the Writer to use the given base directory.
func WithBaseDir(baseDir string) WriterOption {
	return func(w *Writer) {
		w.baseDir = baseDir
	}
}

// WithTimestampDirEnabled controls the addition of a timestamp directory.
func WithTimestampDirEnabled(useTimestampDir bool) WriterOption {
	return func(w *Writer) {
		w.useTimestampDir = useTimestampDir
	}
}

// WithBlockIdInFileNameEnabled controls the inclusion of Block Id in the input json file's name
func WithBlockIdInFileNameEnabled(useBlockIdInFileName bool) WriterOption {
	return func(w *Writer) {
		w.useBlockIdInFileName = useBlockIdInFileName
	}
}

// Write writes the given InputJSON to a file in JSON format.
func (w *Writer) Write(json *server_api.InputJSON) error {
	dir := w.baseDir
	if w.slug != "" {
		dir = filepath.Join(dir, w.slug)
	}
	if w.useTimestampDir {
		t := w.clock.Now()
		tStr := t.Format("20060102_150405")
		dir = filepath.Join(dir, tStr)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	contents, err := json.Marshal()
	if err != nil {
		return err
	}
	fileName := "block_inputs.json"
	if w.useBlockIdInFileName {
		fileName = fmt.Sprintf("block_inputs_%d.json", json.Id)
	}
	if err = os.WriteFile(filepath.Join(dir, fileName), contents, 0600); err != nil {
		return err
	}
	return nil
}
