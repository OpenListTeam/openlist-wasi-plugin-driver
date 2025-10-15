package adapter

import (
	"errors"

	driverexports "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/exports"
	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"
)

// 定义与wit中driver-errors对应的全局错误变量
var (
	ErrInvalidHandle  = errors.New("invalid handle")
	ErrNotImplemented = errors.New("not implemented")
	ErrNotSupport     = errors.New("not support")
	ErrNotFound       = errors.New("not found")
	ErrNotFolder      = errors.New("not folder")
	ErrNotFile        = errors.New("not file")
	ErrUnauthorized   = errors.New("unauthorized")
)

// ErrorToErrRef 将error转换为driverexports.DriverErrors
// 使用errors.Is进行错误类型判断
func ErrorToDriverError(err error) driverexports.DriverErrors {
	switch {
	case errors.Is(err, ErrInvalidHandle):
		return drivertypes.DriverErrorsInvalidHandle()
	case errors.Is(err, ErrNotImplemented):
		return drivertypes.DriverErrorsNotImplemented()
	case errors.Is(err, ErrNotFound):
		return drivertypes.DriverErrorsNotFound()
	case errors.Is(err, ErrNotFolder):
		return drivertypes.DriverErrorsNotFolder()
	case errors.Is(err, ErrNotFile):
		return drivertypes.DriverErrorsNotFile()
	case errors.Is(err, ErrUnauthorized):
		return drivertypes.DriverErrorsUnauthorized(err.Error())
	default:
		return drivertypes.DriverErrorsGeneric(err.Error())
	}
}
