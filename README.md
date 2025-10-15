# OpenList WASI Plugin Driver

该库是一个用于简化 `OpenList` 驱动插件开发的中间层，它封装了 `WASI`（WebAssembly System Interface）的复杂性，让开发者可以更专注于驱动本身的业务逻辑。

## WASI 是什么？

WASI (WebAssembly System Interface) 是一种为 `WebAssembly` 设计的系统接口。它允许 `WebAssembly` 模块以一种标准化的方式与宿主环境（Host）进行交互，例如访问文件系统、网络、时钟等系统资源。

在 `WebAssembly` 最初的设计中，它本身只是一个虚拟机，并没有直接访问外部世界的能力。WASI 的出现，为 `WebAssembly` 打开了通往服务器端、桌面应用等更广阔领域的大门，使其不再局限于浏览器环境。通过 WASI，开发者可以使用 Go、Rust 等语言编写可移植、安全、高性能的插件，并在支持 WASI 的宿主环境中运行。

## 这个库的作用

本库的核心作用是作为**中间层**，它处理了与 `wasi:http`、`wasi:io` 等底层接口的交互细节，并提供了一套更易于使用的 Go 接口。开发者在编写 `OpenList` 驱动时，无需深入了解 `WIT` (WebAssembly Interface Type) 和复杂的 `WASI` 调用，只需实现本库中定义的 `Driver` 接口即可。

这极大地简化了插件的开发流程，让开发者可以：

  * **聚焦业务逻辑**：无需关心底层的 `WASI` 通信细节。
  * **简化编程模型**：以更符合 Go 语言习惯的方式进行开发。
  * **提高开发效率**：减少了学习和使用 `WASI` 的成本。

## 已知的问题

  * [使用time.Sleep()抛出错误](https://github.com/tinygo-org/tinygo/issues/3798)
    解决方法：
      * Guest端使用 `//go:wasmexport` 导出并其Host端设置 `wazero WithStartFunctions("_initialize")`
      * Guest端使用 `-buildmode=c-shared` 编译选项

  * [task.Pause() nilPanic()](https://github.com/tinygo-org/tinygo/issues/4867)
    解决方法：
      * 指定更大的栈 `-stack-size=1MB`

  * 使用 json 时出现 unimplemented: AssignableTo with interface
    解决方法：
      * 避免使用 `reflect.Implements` ,已知使用 errors.As 会触发
      * 避免使用 `github.com/hashicorp/go-multierror` 库，会直接导致错误触发

## 编译指令

```bash
tinygo build -target=wasip1 -no-debug -buildmode=c-shared -o plugin.wasm .
```