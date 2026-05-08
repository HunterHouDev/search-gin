package utils

import (
	"io"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	
	f, err := os.OpenFile("gin.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		logger.Error(err)
	}
	
	logger.SetOutput(io.MultiWriter(f, os.Stdout))
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	logger.SetLevel(logrus.DebugLevel)
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

func DebugFormat(format string, v ...any) {
	logger.Debugf(format, v...)
}

func DebugNormal(v ...any) {
	logger.Debugf("%v", v...)
}

func RecoverPanic() {
	if r := recover(); r != nil {
		ErrorNormal("系统发生异常:", r)
		ErrorNormal("堆栈信息:", string(debug.Stack()))
	}
}
