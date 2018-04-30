package logger

import (
	"os"

	"github.com/op/go-logging"
)

var format = logging.MustStringFormatter(
	`%{color}level=%{level:.4s}%{color:reset} time=%{time:15:04:05.000} file=%{shortfile} %{color}▶%{color:reset} %{message}`,
)

type Level logging.Level

const (
	INFO    Level = Level(logging.INFO)
	DEBUG         = Level(logging.DEBUG)
	ERROR         = Level(logging.ERROR)
	WARNING       = Level(logging.WARNING)
)

func newBackend() *logging.LogBackend {
	return logging.NewLogBackend(os.Stderr, "", 0)
}

func Init() {
	SetLogLevel(INFO)
}

func SetLogLevel(lvl Level) {
	level := logging.Level(lvl)
	backend := newBackend()
	backendFormatted := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormatted)
	backendLeveled.SetLevel(level, "")
	logging.SetBackend(backendLeveled)
}

func NewLogger(name string) *logging.Logger {
	return logging.MustGetLogger(name)
}

func FormatLogMessage(labelValues ...string) string {
	var msg string
	for i, str := range labelValues {
		if i%2 == 0 {
			msg += str + "="
		} else {
			msg += "\"" + str + "\" "
		}
	}
	return msg
}
