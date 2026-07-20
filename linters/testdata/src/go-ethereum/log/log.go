// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package log

type Logger interface {
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
	With(ctx ...interface{}) Logger
	New(ctx ...interface{}) Logger
}

func Trace(msg string, ctx ...interface{}) {}
func Debug(msg string, ctx ...interface{}) {}
func Info(msg string, ctx ...interface{})  {}
func Warn(msg string, ctx ...interface{})  {}
func Error(msg string, ctx ...interface{}) {}
func Crit(msg string, ctx ...interface{})  {}
func New(ctx ...interface{}) Logger        { return nil }
func Root() Logger                         { return nil }
