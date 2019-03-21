package core

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

func Info(format string, v ...interface{}) {
	printf(" INFO", format, v...)
}

func Warn(format string, v ...interface{}) {
	printf(" WARN", format, v...)
}

func Fatal(format string, v ...interface{}) {
	printf("FATAL", format, v...)
	os.Exit(1)
}

func Debug(format string, v ...interface{}) {
	if Config().Core.Debug == 1 {
		printf("DEBUG", format, v...)
	}
}

var logger = log.New(os.Stdout, "", log.LstdFlags)

func printf(prefix string, format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	place := fmt.Sprintf("%s:%d", strings.TrimPrefix(file, ProjectDir()), line)
	logger.SetPrefix(fmt.Sprintf("%s == ", prefix))
	logger.Printf(fmt.Sprintf("%s: %s", place, format), v...)
}
