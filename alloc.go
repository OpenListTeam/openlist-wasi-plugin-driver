package openlistwasiplugindriver

import "unsafe"

//export cabi_realloc
// realloc 分配或重新分配内存，用于 Component Model ABI (cabi) 调用。
// 它被导出为 "cabi_realloc"，供 Wasm 模块（guest）调用。
//
//   - ptr: 指向现有内存块的指针 (如果首次分配则为 0 或 nil)。
//   - size: 现有内存块的大小（字节）。
//   - align: 内存所需的对齐方式（字节）。
//   - newsize: 请求的新内存块大小（字节）。
//
// 如果 newsize 小于或等于旧的 size，函数会复用现有的缓冲区。
// 它返回 *原始* 指针 ptr，但会将其调整（向前推进）以满足 align 要求。
//
// 如果 newsize 大于旧的 size，函数会分配一个具有新大小和对齐方式的新内存块，
// 将旧块中的数据 (size 字节) 复制到新块中，然后返回指向新块的指针。
func realloc(ptr unsafe.Pointer, size, align, newsize uintptr) unsafe.Pointer {
	if newsize <= size {
		// 复用现有缓冲区。返回原始指针，但会根据需要将其向前移动
		// 以满足对齐要求。s
		return unsafe.Add(ptr, offset(uintptr(ptr), align))
	}
	// 分配一个新的、更大的缓冲区
	newptr := alloc(newsize, align)
	if size > 0 {
		// 将旧数据从 ptr 复制到 newptr
		copy(unsafe.Slice((*byte)(newptr), newsize), unsafe.Slice((*byte)(ptr), size))
	}
	return newptr
}

// offset 返回将指针 ptr 向上对齐到 align 所需的字节增量。
//
//   - ptr: 要对齐的指针地址。
//   - align: 期望的对齐值 (必须是 2 的幂)。
//
// 函数计算需要添加到 ptr 上的最小字节数，以使其成为 align 的倍数。
// 返回值保证为非负数 (>= 0)，适用于 unsafe.Add。
func offset(ptr, align uintptr) uintptr {
	// 1. (ptr + align - 1) 确保即使 ptr 已经对齐，
	//    它也会被“推”到下一个对齐块的范围内（或者保持在当前块的末尾）。
	// 2. &^ (align - 1) 是一个位掩码技巧，用于将地址“向下”舍入
	//    到最近的 align 倍数。 (align - 1) 是一个像 0b...0111 这样的掩码,
	//    &^ (按位清除) 会将末尾的几位置零。
	newptr := (ptr + align - 1) &^ (align - 1)

	// 3. newptr - ptr 就是从原始地址到对齐后地址所需的“填充”字节数。
	return newptr - ptr
}

// alloc 分配一个 size 字节的内存块。
// 它尝试通过分配一个与所需对齐方式相匹配的类型的切片来对齐所分配的内存。
// 对于 1、2、4 或 8 之外的 align 值，它将分配 16 字节大小的块。
func alloc(size, align uintptr) unsafe.Pointer {
	// 如果请求的大小为 0，我们仍然需要返回一个有效的、非 nil 的指针，
	// 因为 unsafe.SliceData(make([]T, 0)) 会返回一个非 nil 的哨兵指针。
	if size == 0 {
		return unsafe.Pointer(unsafe.SliceData(make([]uint8, 0)))
	}

	switch align {
	case 1:
		// 需要 'size' 个 uint8 元素
		s := make([]uint8, size)
		return unsafe.Pointer(unsafe.SliceData(s))
	case 2:
		// 需要 n 个 uint16 元素，使得 n * 2 >= size
		n := (size + align - 1) / align // (size + 1) / 2
		s := make([]uint16, n)
		return unsafe.Pointer(unsafe.SliceData(s))
	case 4:
		// 需要 n 个 uint32 元素，使得 n * 4 >= size
		n := (size + align - 1) / align // (size + 3) / 4
		s := make([]uint32, n)
		return unsafe.Pointer(unsafe.SliceData(s))
	case 8:
		// 需要 n 个 uint64 元素，使得 n * 8 >= size
		n := (size + align - 1) / align // (size + 7) / 8
		s := make([]uint64, n)
		return unsafe.Pointer(unsafe.SliceData(s))
	default:
		// 注释表明这里对齐到 16 字节。
		// [2]uint64 的 *大小* 是 16 字节。
		// 我们计算需要多少个 16 字节的块。
		const elementSize = 16
		n := (size + elementSize - 1) / elementSize
		s := make([][2]uint64, n)
		return unsafe.Pointer(unsafe.SliceData(s))
	}
}
