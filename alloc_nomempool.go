//go:build !mempool
// +build !mempool

package openlistwasiplugindriver

import (
	"unsafe"
)

var (
	// zeroSentinel 是一个全局哨兵对象，专门用于响应大小为 0 的分配请求。
	// 使用 uint64 确保其物理地址至少满足 8 字节对齐。
	zeroSentinel struct {
		_ uint64
	}
	// zeroPtr 是指向哨兵对象的通用指针。
	zeroPtr = unsafe.Pointer(&zeroSentinel)
)

// simpleAlloc 是一个简化的底层分配器，确保返回的指针满足 WebAssembly 的对齐要求。
func simpleAlloc(size, align uintptr) unsafe.Pointer {
	if size == 0 {
		return unsafe.Pointer(&zeroSentinel)
	}

	// 绝大多数 WASI 类型的对齐要求 <= 8。
	if align <= 8 {
		n := (size + 7) / 8
		return unsafe.Pointer(unsafe.SliceData(make([]uint64, n)))
	}

	// 针对极其罕见的极端对齐要求 (例如 align = 16 或更大)
	allocSize := size + align
	n := (allocSize + 7) / 8
	base := unsafe.Pointer(unsafe.SliceData(make([]uint64, n)))

	// 计算出严格满足 align 的偏移地址
	addr := (uintptr(base) + align - 1) &^ (align - 1)
	return unsafe.Pointer(addr)
}

//export cabi_free
func cabi_free(ptr unsafe.Pointer) {

}

// cabi_realloc 是 WebAssembly Component Model 核心内存分配例程。
//
//export cabi_realloc
func cabi_realloc(ptr unsafe.Pointer, oldSize, align, newSize uintptr) unsafe.Pointer {
	// 1. 初次分配场景
	if ptr == nil {
		return simpleAlloc(newSize, align)
	}

	// 2. 显式释放场景
	if newSize == 0 {
		// 返回一个对齐的有效哨兵指针
		return simpleAlloc(0, align)
	}

	// 3. 原地复用 (In-place Reallocation)
	// 在纯 GC 模式下，如果我们仅仅是想要缩小分配的内存 (newSize <= oldSize)，
	// 原始指针已经完全满足容量和对齐要求，直接返回即可，省去拷贝。
	if newSize <= oldSize {
		if uintptr(ptr)&(align-1) == 0 {
			return ptr
		}
	}

	// 4. 标准扩容重分配
	newPtr := simpleAlloc(newSize, align)
	if oldSize > 0 {
		copy(unsafe.Slice((*byte)(newPtr), oldSize), unsafe.Slice((*byte)(ptr), oldSize))
	}

	return newPtr
}
