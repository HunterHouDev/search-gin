package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var isProd bool

// dualFormatterHook 让 stdout 和 gin.log 使用不同格式：终端彩色文本，文件 JSON
type dualFormatterHook struct {
	stdFormatter  logrus.Formatter
	fileFormatter logrus.Formatter
	fileWriter    io.Writer
}

func (h *dualFormatterHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *dualFormatterHook) Fire(entry *logrus.Entry) error {
	if b, err := h.stdFormatter.Format(entry); err == nil {
		os.Stdout.Write(b)
	}
	if b, err := h.fileFormatter.Format(entry); err == nil {
		h.fileWriter.Write(b)
	}
	return nil
}

func jsonFormatter() logrus.Formatter {
	return &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "time",
			logrus.FieldKeyMsg:  "message",
		},
	}
}

// customTextFormatter 输出顺序：级别 时间 文件:行 内容
type customTextFormatter struct {
	withColor bool
}

func levelColor(level logrus.Level) string {
	switch level {
	case logrus.InfoLevel:
		return "\033[32m" // 绿
	case logrus.WarnLevel:
		return "\033[33m" // 黄
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return "\033[31m" // 红
	case logrus.DebugLevel:
		return "\033[36m" // 青
	default:
		return "\033[37m" // 白
	}
}

func (f *customTextFormatter) Format(e *logrus.Entry) ([]byte, error) {
	level := strings.ToUpper(e.Level.String())
	ts := e.Time.Format("2006-01-02 15:04:05")
	src, _ := e.Data["src"].(string)
	if src == "" {
		src = "-"
	}
	var b strings.Builder
	if f.withColor {
		b.WriteString(levelColor(e.Level))
	}
	b.WriteString(level)
	if f.withColor {
		b.WriteString("\033[0m")
	}
	fmt.Fprintf(&b, " %s %s | %s\n", ts, src, e.Message)
	return []byte(b.String()), nil
}

// callerField 取出真正打日志的业务代码位置（文件:行），作为 src 字段。
// 封装层级：业务代码 → InfoFormat(等) → callerField → runtime.Caller，
// 故 skip=3 才能跳过两层封装到达业务调用处。
func callerField() logrus.Fields {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return logrus.Fields{"src": "-"}
	}
	return logrus.Fields{"src": fmt.Sprintf("%s:%d", filepath.Base(file), line)}
}

func textFormatter() logrus.Formatter {
	return &customTextFormatter{withColor: true}
}

func SetLogLevel(prod bool) {
	isProd = prod
	if logger == nil {
		return
	}
	if prod {
		logger.SetLevel(logrus.ErrorLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	if h, ok := logger.Hooks[logrus.InfoLevel]; ok && len(h) > 0 {
		if df, ok := h[0].(*dualFormatterHook); ok {
			if prod {
				df.stdFormatter = jsonFormatter()
			} else {
				df.stdFormatter = textFormatter()
			}
		}
	}
}

// 日志文件大小上限：超 5MB 时保留尾部 3MB
const logFileMaxSize = 5 * 1024 * 1024
const logFileKeepSize = 3 * 1024 * 1024

// rotateWriter 包装 *os.File，每次写入前检查大小并自动裁剪
type rotateWriter struct {
	file *os.File
	path string
	mu   sync.Mutex // 保护裁剪操作（Close→Rename→OpenFile 原子化）
}

func (w *rotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 写入前检查文件大小，超限则保留尾部 keepSize 字节
	if fi, err := w.file.Stat(); err == nil && fi.Size() > logFileMaxSize {
		w.truncateTail()
	}
	return w.file.Write(p)
}

func (w *rotateWriter) truncateTail() {
	src, err := os.Open(w.path)
	if err != nil {
		return
	}
	defer src.Close()

	fi, _ := src.Stat()
	fileSize := fi.Size()
	if fileSize <= logFileKeepSize {
		return
	}

	// 定位到文件尾部 keepSize 字节处开始读取
	offset := fileSize - logFileKeepSize
	buf := make([]byte, logFileKeepSize)
	_, err = src.ReadAt(buf, offset)
	if err != nil {
		return
	}

	// 找第一个完整的换行符作为起始，避免截断行
	start := 0
	for start < len(buf) && buf[start] != '\n' {
		start++
	}
	if start < len(buf) {
		buf = buf[start+1:]
	}

	// 原子替换：写临时文件 → 重命名
	tmpPath := w.path + ".tmp"
	if err := os.WriteFile(tmpPath, buf, 0600); err != nil {
		return
	}

	w.file.Close()
	newFile, err := os.OpenFile(tmpPath, os.O_RDWR, 0644)
	if err != nil {
		// 回退：重新打开原文件
		w.file, _ = os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		return
	}
	// 重命名 tmp → 原文件名
	if err := os.Rename(tmpPath, w.path); err != nil {
		newFile.Close()
		w.file, _ = os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		return
	}
	w.file = newFile
}

func init() {
	logger = logrus.New()

	f, err := os.OpenFile("gin.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		logger.SetOutput(os.Stdout)
		logger.SetFormatter(textFormatter())
		logger.Errorf("无法打开 gin.log: %v，仅输出至 stdout", err)
		return
	}

	// stdout 与文件分别用不同格式；后续 SetLogLevel 可切换 stdout 格式
	stdFmt := textFormatter()
	if isProd {
		stdFmt = jsonFormatter()
	}
	logger.SetOutput(io.Discard)
	logger.AddHook(&dualFormatterHook{
		stdFormatter:  stdFmt,
		fileFormatter: jsonFormatter(),
		fileWriter:    &rotateWriter{file: f, path: "gin.log"},
	})
	if isProd {
		logger.SetLevel(logrus.ErrorLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

func NewLogger() *logrus.Logger {
	return logger
}

func InfoFormat(format string, v ...any) {
	logger.WithFields(callerField()).Infof(format, v...)
}

func InfoNormal(v ...any) {
	logger.WithFields(callerField()).Infof("%v", v...)
}

func ErrorFormat(format string, v ...any) {
	logger.WithFields(callerField()).Errorf(format, v...)
}

func ErrorNormal(v ...any) {
	logger.WithFields(callerField()).Errorf("%v", v...)
}

func PanicFormat(format string, v ...any) {
	logger.WithFields(callerField()).Errorf(format, v...)
	logger.WithFields(callerField()).Errorf("Stack trace:\n%s", debug.Stack())
}

func PanicNormal(v ...any) {
	logger.WithFields(callerField()).Errorf("%v", v...)
	logger.WithFields(callerField()).Errorf("Stack trace:\n%s", debug.Stack())
}

func RecoverPanic() {
	if r := recover(); r != nil {
		ErrorNormal("系统发生异常:", r)
		ErrorNormal("堆栈信息:", string(debug.Stack()))
	}
}
