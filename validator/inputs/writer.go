package inputs

import (
	"fmt"
	"os"
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
// is being written. If the Writer is configured by calling SetSlug, then the
// path will be like:
//
//	$HOME/.arbuitrum/validation-inputs/<slug>/<YYYMMDD_HHMMSS>/block_inputs_<id>.json
//
// The inclusion of a timestamp directory is on by default to avoid conflicts which
// would result in files being overwritten. However, the Writer can be configured
// to not use a timestamp directory.  If the Writer is configured by calling
// SetUseTimestampDir(false), then the path will be like:
//
//	$HOME/.arbuitrum/validation-inputs/<slug>/block_inputs_<id>.json
//
// Finally, to give complete control to the clients, the base directory can be
// set directly with SetBaseDir. In which case, the path will be like:
//
//	<baseDir>/block_inputs_<id>.json
//	  or
//	<baseDir>/<slug>/block_inputs_<id>.json
//	  or
//	<baseDir>/<slug>/<YYYMMDD_HHMMSS>/block_inputs_<id>.json
type Writer struct {
	clock           Clock
	baseDir         string
	slug            string
	useTimestampDir bool
}

// Clock is an interface for getting the current time.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

// NewWriter creates a new Writer with default settings.
func NewWriter() (*Writer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := fmt.Sprintf("%s/.arbitrum/validation-inputs", homeDir)
	return &Writer{
		clock:           realClock{},
		baseDir:         baseDir,
		slug:            "",
		useTimestampDir: true}, nil
}

// SetClockForTesting sets the clock used by the Writer.
//
// This is only intended for testing.
func (w *Writer) SetClockForTesting(clock Clock) *Writer {
	w.clock = clock
	return w
}

// SetSlug configures the Writer to use the given slug as a directory name.
func (w *Writer) SetSlug(slug string) *Writer {
	w.slug = slug
	return w
}

// ClearSlug clears the slug configuration.
//
// This is equivalent to calling SetSlug("") but is more readable.
func (w *Writer) ClearSlug() *Writer {
	w.slug = ""
	return w
}

// SetBaseDir configures the Writer to use the given base directory.
func (w *Writer) SetBaseDir(baseDir string) *Writer {
	w.baseDir = baseDir
	return w
}

// SetUseTimestampDir controls the addition of a timestamp directory.
func (w *Writer) SetUseTimestampDir(useTimestampDir bool) *Writer {
	w.useTimestampDir = useTimestampDir
	return w
}

// Write writes the given InputJSON to a file in JSON format.
func (w *Writer) Write(inputs *server_api.InputJSON) error {
	dir := w.baseDir
	if w.slug != "" {
		dir = fmt.Sprintf("%s/%s", dir, w.slug)
	}
	if w.useTimestampDir {
		t := w.clock.Now()
		tStr := t.Format("20060102_150405")
		dir = fmt.Sprintf("%s/%s", dir, tStr)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	contents, err := inputs.Marshal()
	if err != nil {
		return err
	}
	if err = os.WriteFile(
		fmt.Sprintf("%s/block_inputs_%d.json", dir, inputs.Id),
		contents, 0600); err != nil {
		return err
	}
	return nil
}
