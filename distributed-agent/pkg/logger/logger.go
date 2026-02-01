package logger

import (
	"io"
	"os"

	"log/slog"

	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *slog.Logger

func Init(logFilePath string) {
	rotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,  // MB
		MaxBackups: 0,   // only one file
		MaxAge:     0,   // ignore age
		Compress:   false,
	}
	writer := io.MultiWriter(os.Stdout, rotator)
	Log = slog.New(slog.NewJSONHandler(writer,nil))
	slog.SetDefault(Log)
}
