package logger

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global structured logger abstraction.
var (
    zl *zap.Logger
    slogger *slog.Logger
    levelAtomic zap.AtomicLevel
    inited atomic.Bool
)

type Config struct {
    Level  string
    Format string // text|json
}

func Init(cfg Config) {
    if inited.Load() { return }
    level := zap.InfoLevel
    switch cfg.Level {
    case "debug": level = zap.DebugLevel
    case "warn": level = zap.WarnLevel
    case "error": level = zap.ErrorLevel
    }
    levelAtomic = zap.NewAtomicLevelAt(level)
    encCfg := zapcore.EncoderConfig{
        TimeKey:        "ts",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stack",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     func(t time.Time, pae zapcore.PrimitiveArrayEncoder) { pae.AppendString(t.UTC().Format(time.RFC3339Nano)) },
        EncodeDuration: zapcore.StringDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
    var enc zapcore.Encoder
    if cfg.Format == "json" { enc = zapcore.NewJSONEncoder(encCfg) } else { enc = zapcore.NewConsoleEncoder(encCfg) }
    core := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), levelAtomic)
    zl = zap.New(core, zap.AddCaller())
    slogger = slog.New(zapslogHandler{core: core, addSource: true})
    inited.Store(true)
}

// zapslogHandler implements slog.Handler using zap.
type zapslogHandler struct {
    core zapcore.Core
    attrs []slog.Attr
    addSource bool
}

func (h zapslogHandler) Enabled(_ context.Context, level slog.Level) bool {
    switch level {
    case slog.LevelDebug: return levelAtomic.Enabled(zap.DebugLevel)
    case slog.LevelInfo: return levelAtomic.Enabled(zap.InfoLevel)
    case slog.LevelWarn: return levelAtomic.Enabled(zap.WarnLevel)
    case slog.LevelError: return levelAtomic.Enabled(zap.ErrorLevel)
    default: return true
    }
}

func (h zapslogHandler) Handle(_ context.Context, r slog.Record) error {
    fields := make([]zapcore.Field, 0, len(h.attrs)+r.NumAttrs())
    for _, a := range h.attrs { fields = append(fields, attrToField(a)) }
    r.Attrs(func(a slog.Attr) bool { fields = append(fields, attrToField(a)); return true })
    msg := r.Message
    switch {
    case r.Level >= slog.LevelError:
        return h.core.Write(zapcore.Entry{Level: zap.ErrorLevel, Time: r.Time, Message: msg}, fields)
    case r.Level >= slog.LevelWarn:
        return h.core.Write(zapcore.Entry{Level: zap.WarnLevel, Time: r.Time, Message: msg}, fields)
    case r.Level >= slog.LevelInfo:
        return h.core.Write(zapcore.Entry{Level: zap.InfoLevel, Time: r.Time, Message: msg}, fields)
    default:
        return h.core.Write(zapcore.Entry{Level: zap.DebugLevel, Time: r.Time, Message: msg}, fields)
    }
}

func (h zapslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { nh := h; nh.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...); return nh }
func (h zapslogHandler) WithGroup(name string) slog.Handler { return h.WithAttrs([]slog.Attr{slog.Group(name)}) }

func attrToField(a slog.Attr) zapcore.Field {
    a.Value = a.Value.Resolve()
    switch a.Value.Kind() {
    case slog.KindString: return zap.String(a.Key, a.Value.String())
    case slog.KindInt64: return zap.Int64(a.Key, a.Value.Int64())
    case slog.KindUint64: return zap.Uint64(a.Key, a.Value.Uint64())
    case slog.KindFloat64: return zap.Float64(a.Key, a.Value.Float64())
    case slog.KindBool: return zap.Bool(a.Key, a.Value.Bool())
    case slog.KindTime: return zap.Time(a.Key, a.Value.Time())
    default: return zap.Any(a.Key, a.Value.Any())
    }
}

func Zap() *zap.Logger { return zl }
func Slog() *slog.Logger { return slogger }

// SetLevel updates the atomic log level at runtime (debug|info|warn|error).
func SetLevel(level string) {
    switch level {
    case "debug": levelAtomic.SetLevel(zap.DebugLevel)
    case "info": levelAtomic.SetLevel(zap.InfoLevel)
    case "warn": levelAtomic.SetLevel(zap.WarnLevel)
    case "error": levelAtomic.SetLevel(zap.ErrorLevel)
    }
}
