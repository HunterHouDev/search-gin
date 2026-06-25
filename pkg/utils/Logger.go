package utils

import (
	"io"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var isProd bool

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
}

// 日志文件大小上限：超 5MB 时保留尾部 3MB
const logFileMaxSize = 5 * 1024 * 1024
const logFileKeepSize = 3 * 1024 * 1024

// rotateWriter 包装 *os.File，每次写入前检查大小并自动裁剪
type rotateWriter struct {
	file *os.File
	path string
}

func (w *rotateWriter) Write(p []byte) (int, error) {
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
		logger.Errorf("无法打开 gin.log: %v，仅输出至 stdout", err)
	} else {
		logger.SetOutput(io.MultiWriter(&rotateWriter{file: f, path: "gin.log"}, os.Stdout))
	}
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "time",
			logrus.FieldKeyMsg:  "message",
		},
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
	logger.Infof(format, v...)
}

func InfoNormal(v ...any) {
	logger.Infof("%v", v...)
}

func ErrorFormat(format string, v ...any) {
	logger.Errorf(format, v...)
}

func ErrorNormal(v ...any) {
	logger.Errorf("%v", v...)
}

func PanicFormat(format string, v ...any) {
	logger.Errorf(format, v...)
	logger.Errorf("Stack trace:\n%s", debug.Stack())
}

func PanicNormal(v ...any) {
	logger.Errorf("%v", v...)
	logger.Errorf("Stack trace:\n%s", debug.Stack())
}

func RecoverPanic() {
	if r := recover(); r != nil {
		ErrorNormal("系统发生异常:", r)
		ErrorNormal("堆栈信息:", string(debug.Stack()))
	}
}
