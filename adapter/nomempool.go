//go:build !mempool
// +build !mempool

package adapter

//go:inline
func freeWasiSlice[T any](s []T) {
	// Do nothing
}

//go:inline
func freeWasiString(s string) {
	// Do nothing
}

//go:inline
func cloneStringAndFree(s string) string { return s }

//go:inline
func cloneSliceAndFree[T any](s []T) []T { return s }
