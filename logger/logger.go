package logger

import (
	"os"

	ps "github.com/mitchellh/go-ps"
	"github.com/natefinch/lumberjack"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
 log *zap.Logger
)

type ProcessAction struct {
	ActionType string
	Process *ps.Process
}

func (a ProcessAction) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("action_type", a.ActionType)
	process := *a.Process
	enc.AddInt("pid", process.Pid())
	enc.AddString("executable", process.Executable())
	return nil
}

// func (r *request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
// 	enc.AddString("url", r.URL)
// 	zap.Inline(r.Listen).AddTo(enc)
// 	return enc.AddObject("remote", r.Remote)
// }

func New() {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "nvim-watcher.log",
		MaxSize:    5, // megabytes
		MaxBackups: 1,
		MaxAge:     7, // days,
		LocalTime:  true,
	})
	mw := zapcore.NewMultiWriteSyncer(
    zapcore.AddSync(os.Stdout),
    zapcore.AddSync(fileWriter),
	)
	core := zapcore.NewCore(
	 	zapcore.NewJSONEncoder(config),
	  mw,
	  zap.InfoLevel,//zap.NewAtomicLevelAt(zapcore.DebugLevel),
  )
	log = zap.New(core)
}
