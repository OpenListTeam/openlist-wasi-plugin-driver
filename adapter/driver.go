package adapter

import (
	"maps"

	driverexports "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/exports"
	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"

	"go.bytecodealliance.org/cm"
)

type ResultOptionObject = cm.Result[driverexports.OptionObjectShape, cm.Option[driverexports.Object], driverexports.DriverErrors]
type ResultObject = cm.Result[driverexports.ObjectShape, driverexports.Object, driverexports.DriverErrors]
type ResultObjects = cm.Result[driverexports.DriverErrorsShape, cm.List[driverexports.Object], driverexports.DriverErrors]
type Result = cm.Result[driverexports.DriverErrors, struct{}, driverexports.DriverErrors]

//go:inline
func ErrRef(err driverexports.DriverErrors) *driverexports.DriverErrors {
	return &err
}

func ErrorToErrRef(err error) *driverexports.DriverErrors {
	e := drivertypes.DriverErrorsGeneric(err.Error())
	return &e
}

func OptionObject(obj *drivertypes.Object) cm.Option[drivertypes.Object] {
	if obj == nil {
		return cm.None[driverexports.Object]()
	}
	return cm.Some(*obj)
}

// result<option<object>, err-code>
//
//go:inline
func ReturnOkOptionObject(obj *drivertypes.Object) (result ResultOptionObject) {
	return cm.OK[ResultOptionObject](OptionObject(obj))
}

// result<option<object>, err-code>
//
//go:inline
func ReturnErrCodeOptionObject(err driverexports.DriverErrors) (result ResultOptionObject) {
	return cm.Err[ResultOptionObject](err)
}

// result<option<object>, err-code>
//
//go:inline
func ReturnErrOptionObject(err error) (result ResultOptionObject) {
	return cm.Err[ResultOptionObject](drivertypes.DriverErrorsGeneric(err.Error()))
}

// result<object, err-code>
//
//go:inline
func ReturnOkObject(obj driverexports.Object) (result ResultObject) {
	return cm.OK[ResultObject](obj)
}

// result<object, err-code>
//
//go:inline
func ReturnErrCodeObject(err driverexports.DriverErrors) (result ResultObject) {
	return cm.Err[ResultObject](err)
}

// result<object, err-code>
//
//go:inline
func ReturnErrObject(err error) (result ResultObject) {
	return cm.Err[ResultObject](drivertypes.DriverErrorsGeneric(err.Error()))
}

// result<object, err-code>
//
//go:inline
func ReturnOkObjects(obj []driverexports.Object) (result ResultObjects) {
	return cm.OK[ResultObjects](cm.ToList(obj))
}

// result<object, err-code>
//
//go:inline
func ReturnErrCodeObjects(err driverexports.DriverErrors) (result ResultObjects) {
	return cm.Err[ResultObjects](err)
}

// result<object, err-code>
//
//go:inline
func ReturnErrObjects(err error) (result ResultObjects) {
	return cm.Err[ResultObjects](drivertypes.DriverErrorsGeneric(err.Error()))
}

// result<_, err-code>
//
//go:inline
func ReturnErr(err error) (result Result) {
	return cm.Err[Result](drivertypes.DriverErrorsGeneric(err.Error()))
}

// result<_, err-code>
//
//go:inline
func ReturnOk() (result Result) {
	return cm.OK[Result](struct{}{})
}

func ExtraGet(extra cm.List[[2]string], key string) (string, bool) {
	for _, v := range extra.Slice() {
		if v[0] == key {
			return v[1], true
		}
	}
	return "", false
}

func ExtraGetDefable(extra cm.List[[2]string], key string) string {
	for _, v := range extra.Slice() {
		if v[0] == key {
			return v[1]
		}
	}
	return ""
}

func ExtraFormMap(extraMap map[string]string) cm.List[[2]string] {
	var extra [][2]string
	for k, v := range extraMap {
		extra = append(extra, [2]string{k, v})
	}
	return cm.ToList(extra)
}

func ExtraToMap(extra cm.List[[2]string]) map[string]string {
	m := make(map[string]string, len(extra.Slice())) // 预分配容量，减少扩容
	for _, pair := range extra.Slice() {
		key, value := pair[0], pair[1]
		if _, exists := m[key]; !exists { // 只保留第一个出现的key
			m[key] = value
		}
	}
	return m
}

func ExtraAppend(extra cm.List[[2]string], newPairs ...[2]string) cm.List[[2]string] {
	mergedMap := ExtraToMap(extra)
	for _, pair := range newPairs {
		mergedMap[pair[0]] = pair[1]
	}
	return ExtraFormMap(mergedMap)
}

func ExtraAppendMap(extra cm.List[[2]string], newExtra map[string]string) cm.List[[2]string] {
	mergedMap := ExtraToMap(extra)
	maps.Copy(mergedMap, newExtra)
	return ExtraFormMap(mergedMap)
}
