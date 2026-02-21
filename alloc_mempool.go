//go:build mempool
// +build mempool

package openlistwasiplugindriver

import (
	"sync/atomic"
	"unsafe"
)

const (
	// 分配器的尺寸分级 (Size Classes) 参数。
	// minBlockShift = 5 表示最小池化块为 2^5 = 32 字节。
	// maxBlockShift = 17 表示最大池化块为 2^17 = 128 KB。
	minBlockShift = 5
	maxBlockShift = 17
	numClasses    = maxBlockShift - minBlockShift + 1

	// headerSize 隐藏在有效载荷 (Payload) 前方的元数据大小。
	headerSize = 8

	// 魔数 (Magic Numbers)，用于运行时校验指针合法性及防止释放后使用 (Use-After-Free)。
	magicPooled   uint32 = 0x424C4F4B // "BLOK" - 内存块属于无锁对象池
	magicUnpooled uint32 = 0x4E4F4E45 // "NONE" - 内存块由 Go GC 直接管理
)

var (
	// freeLists 是按尺寸分级的无锁空闲链表。
	// 索引 i 对应的块大小为 2^(minBlockShift + i) 字节。
	freeLists [numClasses]atomic.Pointer[memoryBlock]

	// zeroSentinel 是一个全局哨兵对象，专门用于响应大小为 0 的分配请求。
	// 使用 uint64 确保其物理地址至少满足 8 字节对齐。
	zeroSentinel struct {
		_ uint64
	}
	// zeroPtr 是指向哨兵对象的通用指针。
	zeroPtr = unsafe.Pointer(&zeroSentinel)
)

// blockHeader 是位于分配指针前方 headerSize 字节处的元数据。
// 它必须是 8 字节大小以确保后续 Payload 的自然对齐。
type blockHeader struct {
	magic    atomic.Uint32 // 原子操作保护，防止并发 Double-Free
	classIdx int32         // 对应的尺寸分级索引；-1 表示未池化 (回退分配)
}

// memoryBlock 代表空闲链表中的一个节点，在块被回收后直接覆盖在 Payload 区域之上。
type memoryBlock struct {
	next atomic.Pointer[memoryBlock]
}

// classIndex 根据请求大小计算其对应的尺寸分级索引。
// 如果请求超出了池化支持的最大阈值 (128 KB)，则返回 -1。
func classIndex(size uintptr) int {
	total := size + headerSize
	for i := range numClasses {
		if total <= (uintptr(1) << (minBlockShift + i)) {
			return i
		}
	}
	return -1
}

// pushBlock 使用 CAS (Compare-And-Swap) 原子操作将内存块压入无锁栈。
func pushBlock(idx int, node *memoryBlock) {
	for {
		head := freeLists[idx].Load()
		node.next.Store(head)
		if freeLists[idx].CompareAndSwap(head, node) {
			break
		}
	}
}

// popBlock 使用 CAS 操作从无锁栈中弹出一个内存块，若栈为空则返回 nil。
func popBlock(idx int) *memoryBlock {
	for {
		head := freeLists[idx].Load()
		if head == nil {
			return nil
		}
		next := head.next.Load()
		if freeLists[idx].CompareAndSwap(head, next) {
			// 断开 next 指针引用，避免保守式 GC 扫描时产生错误的存活判断。
			head.next.Store(nil)
			return head
		}
	}
}

// sysAlloc 封装底层的物理内存分配，利用 Go 切片机制隐式提供底层支持。
func sysAlloc(size uintptr) unsafe.Pointer {
	// 请求向上取整到 uint64 的倍数，在语言层面获取天然的 8 字节对齐内存。
	n := (size + 7) / 8
	return unsafe.Pointer(unsafe.SliceData(make([]uint64, n)))
}

// fallbackAlloc 处理巨大对象或具有极端对齐要求的分配请求。
// 分配出的内存不进入对象池，其生命周期将隐式交还给 Go 垃圾回收器。
func fallbackAlloc(size, align uintptr) unsafe.Pointer {
	if align < headerSize {
		align = headerSize
	}

	// 额外申请足够的空间，用于吸纳由于对齐产生的偏移量 (Padding) 以及 Header。
	allocSize := size + align + headerSize
	base := sysAlloc(allocSize)

	// 计算能够完整容纳 Header 的最小载荷基址。
	minPayload := uintptr(base) + headerSize

	// 将载荷地址严格向上对齐到 align 的整数倍。
	payloadAddr := (minPayload + align - 1) &^ (align - 1)
	payload := unsafe.Pointer(payloadAddr)

	// 在严格对齐的 Payload 前方预留的 8 字节处写入元数据。
	hdr := (*blockHeader)(unsafe.Add(payload, -headerSize))
	hdr.magic.Store(magicUnpooled)
	hdr.classIdx = -1

	return payload
}

// poolAlloc 尝试从特定尺寸的无锁池中分配内存；若池已空则向系统申请新切片。
func poolAlloc(size, align uintptr) unsafe.Pointer {
	idx := classIndex(size)

	// 逃逸判定：若超出池容量，或对齐要求高于基础对齐 (8 字节)，走大内存回退逻辑。
	if idx == -1 || align > headerSize {
		return fallbackAlloc(size, align)
	}

	// 快速路径 (Fast Path)：O(1) 无锁命中。
	if node := popBlock(idx); node != nil {
		payload := unsafe.Pointer(node)
		hdr := (*blockHeader)(unsafe.Add(payload, -headerSize))
		hdr.magic.Store(magicPooled) // 原子重置合法状态
		return payload
	}

	// 慢速路径 (Slow Path)：池为空，分配一个全新的物理块。
	blockSize := uintptr(1) << (minBlockShift + idx)
	base := sysAlloc(blockSize)

	payload := unsafe.Add(base, headerSize)
	hdr := (*blockHeader)(base)

	hdr.magic.Store(magicPooled)
	hdr.classIdx = int32(idx)

	return payload
}

// cabi_free 回收由 cabi_realloc 分配的指针。
//
//export cabi_free
func cabi_free(ptr unsafe.Pointer) {
	if ptr == nil || ptr == zeroPtr {
		return
	}

	hdr := (*blockHeader)(unsafe.Add(ptr, -headerSize))
	currentMagic := hdr.magic.Load()

	switch currentMagic {
	case magicPooled:
		if hdr.magic.CompareAndSwap(magicPooled, 0) {
			idx := hdr.classIdx
			if idx >= 0 && int(idx) < numClasses {
				node := (*memoryBlock)(ptr)
				pushBlock(int(idx), node)
			}
		}
	case magicUnpooled:
		hdr.magic.CompareAndSwap(magicUnpooled, 0)
	}
}

// cabi_realloc 是 WebAssembly Component Model 内存生命周期管理的核心。
//
//export cabi_realloc
func cabi_realloc(ptr unsafe.Pointer, oldSize, align, newSize uintptr) unsafe.Pointer {
	// 场景 1：首次分配 (Malloc)
	if ptr == nil {
		if newSize == 0 && align <= headerSize {
			return zeroPtr // 命中全局零大小哨兵
		}
		return poolAlloc(newSize, align)
	}

	// 场景 2：释放操作 (Free)，需返回一个有效的零大小哨兵
	if newSize == 0 {
		cabi_free(ptr)
		if align <= headerSize {
			return zeroPtr
		}
		return poolAlloc(0, align) // 应对极端要求的零大小高对齐分配
	}

	// 读取元数据进行扩容/缩容判断
	hdr := (*blockHeader)(unsafe.Add(ptr, -headerSize))
	currentMagic := hdr.magic.Load()

	// 场景 3：原地复用 (In-place Reallocation) - 零分配、零拷贝快速路径
	switch currentMagic {
	case magicPooled:
		blockSize := uintptr(1) << (minBlockShift + hdr.classIdx)
		usableSize := blockSize - headerSize

		// 若新容量仍在当前物理块限制内，且底层对齐依然满足要求，则直接返回。
		if newSize <= usableSize && align <= headerSize {
			return ptr
		}
	case magicUnpooled:
		// 未池化对象的原地复用：仅在降级缩容时直接返回。
		if newSize <= oldSize && align <= headerSize {
			return ptr
		}
	}

	// 场景 4：常规重分配 (分配新块 -> 数据迁移 -> 释放旧块)
	newPtr := poolAlloc(newSize, align)
	copySize := min(newSize, oldSize)
	if copySize > 0 {
		// 使用 Go 底层 runtime 提供的高效 slice memory copy
		copy(unsafe.Slice((*byte)(newPtr), copySize), unsafe.Slice((*byte)(ptr), copySize))
	}

	cabi_free(ptr)

	return newPtr
}

//go:linkname cabiFreeNet github.com/OpenListTeam/go-wasi-socket/wasip2_net.cabiFree
func cabiFreeNet(ptr unsafe.Pointer) {
	cabi_free(ptr)
}

//go:linkname cabiFreeHTTP github.com/OpenListTeam/go-wasi-http/wasihttp.cabiFree
func cabiFreeHTTP(ptr unsafe.Pointer) {
	cabi_free(ptr)
}

//go:linkname cabiFreeDriver github.com/OpenListTeam/openlist-wasi-plugin-driver/adapter.cabiFree
func cabiFreeDriver(ptr unsafe.Pointer) {
	cabi_free(ptr)
}
