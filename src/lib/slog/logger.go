package slog

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	config     *Config
	logger     *zap.Logger
	mux        sync.Mutex
	debugLevel int
)

// Config allows slog to be configured.
type Config struct {
	Disabled bool
	Colorful bool
}

func init() {
	d := strings.ToLower(os.Getenv("DEBUG"))

	if d == "true" {
		debugLevel = 9
		return
	}

	debugLevel, _ = strconv.Atoi(d)

	if debugLevel > 0 {
		Debug(LogOpts{
			Msg: "debug mode enabled",
			Payload: []zap.Field{
				zap.Int("level", debugLevel),
			},
		})
	} else {
		Info("debug mode is disabled")
	}
}

// getConfig is a helper function to return the current config.
func getConfig() *Config {
	if config == nil {
		config = &Config{
			Disabled: strings.Contains(os.Args[0], "_test") || strings.Contains(os.Args[0], ".test"),
		}
	}

	return config
}

// getLogger returns a configured zap logger instance
func getLogger() *zap.Logger {
	mux.Lock()
	defer mux.Unlock()

	if logger == nil {
		cfg := getConfig()

		// Create a custom encoder config for colorful output
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Use color encoding if colorful is enabled
		if cfg.Colorful {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		// Configure logging level
		level := zapcore.InfoLevel

		if debug := os.Getenv("DEBUG"); debug != "" {
			level = zapcore.DebugLevel
		}

		// Create core with console encoder
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		)

		// Disable logging if configured
		if cfg.Disabled {
			core = zapcore.NewNopCore()
		}

		logger = zap.New(core, zap.AddCallerSkip(1))
	}

	return logger
}

// SetConfig sets the configuration for slog.
func SetConfig(conf *Config) {
	config = conf
	// Reset logger to pick up new config
	logger = nil
}

const DL1 = 1 // Debug level 1 is used for general debug messages.
const DL2 = 2 // Debug level 2 is used for more detailed debug messages, such as function calls and variable values.
const DL3 = 3 // Debug level 3 is used for noisy messages, such as request and response logging.
const DL4 = 4 // Debug level 4 is used for extremely noisy messages, such as logging every line of code execution (not recommended for regular use).

type LogOpts struct {
	Msg     string
	MsgArgs []any
	Payload []zap.Field
	Level   int
}

// Debug logs debug level stuff.
func Debug(opts LogOpts) {
	// Debugging is disabled
	if debugLevel == 0 {
		return
	}

	// Debugging is enabled, but the level is not sufficient
	if debugLevel > 0 && opts.Level > debugLevel {
		return
	}

	msg := opts.Msg

	if len(opts.MsgArgs) > 0 {
		msg = fmt.Sprintf(msg, opts.MsgArgs...)
	}

	getLogger().Debug(msg, opts.Payload...)
}

// Info logs info level stuff.
func Info(v ...any) {
	getLogger().Info(fmt.Sprint(v...))
}

// Infof accepts a formatted string and calls Info function.
func Infof(msg string, args ...any) {
	Info(fmt.Sprintf(msg, args...))
}

// Error logs error level stuff.
func Error(v ...any) {
	_, file, no, ok := runtime.Caller(1)

	// caller is Errorf go one step deeper
	if strings.HasSuffix(file, "logger.go") {
		_, file, no, ok = runtime.Caller(2)
	}

	msg := fmt.Sprint(v...)

	fields := []zap.Field{}

	if ok {
		fields = append(fields, zap.String("caller", fmt.Sprintf("%s#%d", file, no)))
	}

	getLogger().Error(msg, fields...)
}

// Errorf accepts a formatted string and calls Error function.
func Errorf(msg string, args ...any) {
	Error(fmt.Sprintf(msg, args...))
}
