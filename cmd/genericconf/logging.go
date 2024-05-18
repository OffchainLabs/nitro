package genericconf

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalFileLoggerFactory = fileLoggerFactory{}

type fileLoggerFactory struct {
	// writerMutex is to avoid parallel writes to the file-logger
	writerMutex sync.Mutex
	writer      *lumberjack.Logger

	cancel context.CancelFunc

	// writeStartPing and writeDonePing are used to simulate sending of data via a buffered channel
	// when Write is called and receiving it on another go-routine to write it to the io.Writer.
	writeStartPing chan struct{}
	writeDonePing  chan struct{}
}

// Write is essentially a wrapper for filewriter or lumberjack.Logger's Write method to implement
// config.BufSize functionality, data is dropped when l.writeStartPing channel (of size config.BuffSize) is full
func (l *fileLoggerFactory) Write(p []byte) (n int, err error) {
	select {
	case l.writeStartPing <- struct{}{}:
		// Write data to the filelogger
		l.writerMutex.Lock()
		_, _ = l.writer.Write(p)
		l.writerMutex.Unlock()
		l.writeDonePing <- struct{}{}
	default:
	}
	return len(p), nil
}

// newFileWriter is not threadsafe
func (l *fileLoggerFactory) newFileWriter(config *FileLoggingConfig, filename string) io.Writer {
	l.close()
	l.writer = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
	l.writeStartPing = make(chan struct{}, config.BufSize)
	l.writeDonePing = make(chan struct{}, config.BufSize)
	// capture copy
	writeStartPing := l.writeStartPing
	writeDonePing := l.writeDonePing
	var consumerCtx context.Context
	consumerCtx, l.cancel = context.WithCancel(context.Background())
	go func() {
		// writeStartPing channel signals Write operations to correctly implement config.BufSize functionality
		for {
			select {
			case <-writeStartPing:
				<-writeDonePing
			case <-consumerCtx.Done():
				return
			}
		}
	}()
	return l
}

// close is not threadsafe
func (l *fileLoggerFactory) close() error {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	if l.writer != nil {
		if err := l.writer.Close(); err != nil {
			return err
		}
		l.writer = nil
	}
	return nil
}

// initLog is not threadsafe
func InitLog(logType string, logLevel string, fileLoggingConfig *FileLoggingConfig, pathResolver func(string) string) error {
	var glogger *log.GlogHandler
	// always close previous instance of file logger
	if err := globalFileLoggerFactory.close(); err != nil {
		return fmt.Errorf("failed to close file writer: %w", err)
	}
	var output io.Writer
	if fileLoggingConfig.Enable {
		output = io.MultiWriter(
			io.Writer(os.Stderr),
			// on overflow writeStartPing are dropped silently
			globalFileLoggerFactory.newFileWriter(fileLoggingConfig, pathResolver(fileLoggingConfig.File)),
		)
	} else {
		output = io.Writer(os.Stderr)
	}
	handler, err := HandlerFromLogType(logType, output)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log type when creating handler: %w", err)
	}
	slogLevel, err := ToSlogLevel(logLevel)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log level: %w", err)
	}

	glogger = log.NewGlogHandler(handler)
	glogger.Verbosity(slogLevel)
	log.SetDefault(log.NewLogger(glogger))
	return nil
}
