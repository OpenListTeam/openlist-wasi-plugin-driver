package openlistwasiplugindriver

import (
	"fmt"

	driverimports "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/host"
)

// Debugln 记录一条 Debug 级别的日志，参数间用空格分隔，结尾添加换行。
func Debugln(v ...any) {
	message := fmt.Sprintln(v...)
	driverimports.Log(driverimports.LogLevelDebug, message)
}

// Infoln 记录一条 Info 级别的日志，参数间用空格分隔，结尾添加换行。
func Infoln(v ...any) {
	message := fmt.Sprintln(v...)
	driverimports.Log(driverimports.LogLevelInfo, message)
}

// Warnln 记录一条 Warn 级别的日志，参数间用空格分隔，结尾添加换行。
func Warnln(v ...any) {
	message := fmt.Sprintln(v...)
	driverimports.Log(driverimports.LogLevelWarn, message)
}

// Errorln 记录一条 Error 级别的日志，参数间用空格分隔，结尾添加换行。
func Errorln(v ...any) {
	message := fmt.Sprintln(v...)
	driverimports.Log(driverimports.LogLevelError, message)
}

// Debugf 格式化并记录一条 Debug 级别的日志。
func Debugf(format string, v ...any) {
	message := fmt.Sprintf(format, v...)
	driverimports.Log(driverimports.LogLevelDebug, message)
}

// Infof 格式化并记录一条 Info 级别的日志。
func Infof(format string, v ...any) {
	message := fmt.Sprintf(format, v...)
	driverimports.Log(driverimports.LogLevelInfo, message)
}

// Warnf 格式化并记录一条 Warn 级别的日志。
func Warnf(format string, v ...any) {
	message := fmt.Sprintf(format, v...)
	driverimports.Log(driverimports.LogLevelWarn, message)
}

// Errorf 格式化并记录一条 Error 级别的日志。
func Errorf(format string, v ...any) {
	message := fmt.Sprintf(format, v...)
	driverimports.Log(driverimports.LogLevelError, message)
}
