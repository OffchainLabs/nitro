package genericconf

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalFileHandlerFactory = fileHandlerFactory{}

type fileHandlerFactory struct {
	writer  *lumberjack.Logger
	records chan *log.Record
	cancel  context.CancelFunc
}

// newHandler is not threadsafe
func (l *fileHandlerFactory) newHandler(logFormat log.Format, config *FileLoggingConfig, pathResolver func(string) string) log.Handler {
	l.close()
	l.writer = &lumberjack.Logger{
		Filename:   pathResolver(config.File),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
	// capture copy of the pointer
	writer := l.writer
	// lumberjack.Logger already locks on Write, no need for SyncHandler proxy which is used in StreamHandler
	unsafeStreamHandler := log.LazyHandler(log.FuncHandler(func(r *log.Record) error {
		_, err := writer.Write(logFormat.Format(r))
		return err
	}))
	l.records = make(chan *log.Record, config.BufSize)
	// capture copy
	records := l.records
	var consumerCtx context.Context
	consumerCtx, l.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case r := <-records:
				_ = unsafeStreamHandler.Log(r)
			case <-consumerCtx.Done():
				return
			}
		}
	}()
	return log.FuncHandler(func(r *log.Record) error {
		select {
		case records <- r:
			return nil
		default:
			return fmt.Errorf("Buffer overflow, dropping record")
		}
	})
}

// close is not threadsafe
func (l *fileHandlerFactory) close() error {
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
func InitLog(logType string, logLevel log.Lvl, fileLoggingConfig *FileLoggingConfig, pathResolver func(string) string) error {
	logFormat, err := ParseLogType(logType)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log type: %w", err)
	}
	var glogger *log.GlogHandler
	// always close previous instance of file logger
	if err := globalFileHandlerFactory.close(); err != nil {
		return fmt.Errorf("failed to close file writer: %w", err)
	}
	if fileLoggingConfig.Enable {
		glogger = log.NewGlogHandler(
			log.MultiHandler(
				log.StreamHandler(os.Stderr, logFormat),
				// on overflow records are dropped silently as MultiHandler ignores errors
				globalFileHandlerFactory.newHandler(logFormat, fileLoggingConfig, pathResolver),
			))
	} else {
		glogger = log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	}
	glogger.Verbosity(logLevel)
	log.Root().SetHandler(glogger)
	return nil
}
