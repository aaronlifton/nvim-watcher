package log

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ps "github.com/mitchellh/go-ps"
	"github.com/natefinch/lumberjack"

	stdlog "log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	stdLog         *stdlog.Logger
	FileLogger     *zap.SugaredLogger
	ConsoleLogger  *zap.SugaredLogger
	CombinedLogger *zap.SugaredLogger
	GitLogger      *zap.SugaredLogger
	// CombinedGitLogger *zap.SugaredLogger
	CronLogger *zap.SugaredLogger
)

// Init initializes the loggers
// CombinedLogger is for logging to both console and file
// GitLogger is for logging explicit git commands only
func Init() {
	config := zap.NewProductionEncoderConfig()
	// config := zap.NewDevelopmentEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join("logs", "nvim-watcher.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 1,
		MaxAge:     1, // days,
		LocalTime:  true,
	})
	gitWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join("logs", "git.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 1,
		MaxAge:     7, // days,
	})
	cronWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join("logs", "gocron.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 1,
		MaxAge:     1, // days,
	})
	// mw := zapcore.NewMultiWriteSyncer(
	//    zapcore.AddSync(os.Stdout),
	//    zapcore.AddSync(fileWriter),
	// )
	// core := zapcore.NewTee(
	//        zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zap.DebugLevel),
	//    1    zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
	//    )

	// core 1: FileLogger
	// core 2: ConsoleLogger
	// core 3: GitLogger
	cores := make([]zapcore.Core, 4)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(fileWriter),
		zap.DebugLevel,
	)
	FileLogger = zap.New(core).Sugar()
	cores[0] = core
	core2 := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)
	ConsoleLogger = zap.New(core2).Sugar()

	cores[1] = core2
	teeCore := zapcore.NewTee(cores...)
	CombinedLogger = zap.New(teeCore).Sugar()

	cores[2] = zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(gitWriter),
		zap.InfoLevel,
	)
	GitLogger = zap.New(cores[2]).Sugar()
	// teeCore2 := zapcore.NewTee(cores[2], cores[0])
	// CombinedGitLogger = zap.New(teeCore2).Sugar()
	cores[3] = zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(cronWriter),
		zap.InfoLevel,
	)
	CronLogger = zap.New(cores[3]).Sugar()
}

func LogGitCommand(cmd *exec.Cmd) {
	process, err := ps.FindProcess(cmd.Process.Pid)
	if err != nil {
		CombinedLogger.Fatal(
			"Failed to find process",
			zap.Int("pid", process.Pid()),
			zap.Error(err),
		)
	}
	GitLogger.Info("Ran git command",
		zap.Dict(
			"command",
			zap.String("name", process.Executable()),
			zap.String("args", strings.Join(cmd.Args, ", ")),
			zap.String("dir", cmd.Dir),
			zap.String("full_command", cmd.String())),
	)
}

// type ProcessInfo struct {
// 	Name          string
// 	Exe           string
// 	PercentMemory string
// 	PercentCpu    string
// }
//
// func LogProcesses(ps []ProcessInfo) {
// 	for _, process := range ps {
// 		FileLogger.Info("Top 10 processes",
// 			zap.Dict(
// 				"process",
// 				zap.String("name", process.Name),
// 				zap.String("exe", process.Exe),
// 				zap.String("memPercent", process.PercentMemory),
// 				zap.String("cpuPercent", process.PercentCpu),
// 			),
// 		)
// 	}
// }
