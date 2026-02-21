//go:build mempool
// +build mempool

package adapter

import "unsafe"

func cabiFree(ptr unsafe.Pointer)

//go:inline
func freeWasiSlice[T any](s []T) {
	if len(s) > 0 {
		cabiFree(unsafe.Pointer(&s[0]))
	}
}

// freeWasiString 释放 Wasm 分配的字符串
//go:inline
func freeWasiString(s string) {
	if len(s) > 0 {
		cabiFree(unsafe.Pointer(unsafe.StringData(s)))
	}
}

// cloneStringAndFree 深拷贝字符串到 Go 堆内存，并释放原有的 Wasm 内存池块
//go:inline
func cloneStringAndFree(s string) string {
	if len(s) == 0 {
		return ""
	}

	b := make([]byte, len(s))
	copy(b, s)
	res := string(b)

	// 释放底层 Wasm 内存
	freeWasiString(s)
	return res
}

//go:inline
func cloneSliceAndFree[T any](s []T) []T {
	if s == nil {
		return nil
	}
	if len(s) == 0 {
		return []T{}
	}
	res := make([]T, len(s))
	copy(res, s)
	freeWasiSlice(s)
	return res
}
