//go:generate wit-bindgen-go -v generate --world plugin --out binding ./wit

package openlistwasiplugindriver

import (
	"context"
	"io"

	"go.bytecodealliance.org/cm"

	"github.com/OpenListTeam/openlist-wasi-plugin-driver/adapter"
	"github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/exports"
	drivertypes "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/types"
)

type LinkArgs = exports.LinkArgs

type Driver interface {
	GetHandle() uint32
	SetHandle(handle uint32)

	// 获取驱动功能描述
	GetProperties() drivertypes.DriverProps
	// 获取配置表单
	GetFormMeta() []drivertypes.FormField
	Init(ctx context.Context) error
	Drop(ctx context.Context) error
	Reader
}

type Reader interface {
	GetRoot(ctx context.Context) (*drivertypes.Object, error)
	ListFiles(ctx context.Context, dir drivertypes.Object) ([]drivertypes.Object, error)
	LinkFile(ctx context.Context, file drivertypes.Object, args LinkArgs) (*drivertypes.LinkResource, *drivertypes.Object, error)
}

type StreamReader interface {
	LinkRange(ctx context.Context, file drivertypes.Object, args LinkArgs, _range drivertypes.RangeSpec, w io.WriteCloser) error
}

// 用于优化List，用于单个文件根据路径查找
type Getter interface {
	Get(ctx context.Context, path string) (*drivertypes.Object, error)
}

type Mkdir interface {
	MakeDir(ctx context.Context, parentDir drivertypes.Object, dirName string) (*drivertypes.Object, error)
}
type Move interface {
	Move(ctx context.Context, srcObj, dstDir drivertypes.Object) (*drivertypes.Object, error)
}

type Rename interface {
	Rename(ctx context.Context, srcObj drivertypes.Object, newName string) (*drivertypes.Object, error)
}

type Copy interface {
	Copy(ctx context.Context, srcObj, dstDir drivertypes.Object) (*drivertypes.Object, error)
}

type Remove interface {
	Remove(ctx context.Context, obj drivertypes.Object) error
}

type Put interface {
	Put(ctx context.Context, dstDir drivertypes.Object, file adapter.UploadRequest) (*drivertypes.Object, error)
}

var DriverManager *adapter.ResourceManager[Driver] = adapter.NewResourceManager[Driver](nil)

type RangeReaderFn = func(offset, size uint64) io.ReadCloser

var RangeReaderManager *adapter.ResourceManager[RangeReaderFn] = adapter.NewResourceManager[RangeReaderFn](nil)

var CreateDriver func() Driver

func init() {
	exports.Exports.Driver.Constructor = func() (result exports.Driver) {
		driver := CreateDriver()
		driverHandle := DriverManager.Add(driver)
		driver.SetHandle(driverHandle)
		return exports.Driver(driverHandle)
	}

	exports.Exports.Driver.Destructor = func(self cm.Rep) {
		DriverManager.Remove(uint32(self))
	}

	exports.Exports.Driver.GetProperties = func(self cm.Rep) (result exports.DriverProps) {
		driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			panic("DriverErrorsInvalidHandle")
		}

		properties := driver.GetProperties()

		// 未配置时自动识别
		if properties.Capabilitys == 0 {
			var flags drivertypes.Capability
			// 检查是否实现 Getter 接口
			if _, ok := driver.(Getter); ok {
				flags |= drivertypes.CapabilityGetFile
			}

			// 检查是否实现 Reader 接口
			if _, ok := driver.(Reader); ok {
				flags |= drivertypes.CapabilityLinkFile
				flags |= drivertypes.CapabilityListFile
			}

			// 检查是否实现 Mkdir 接口
			if _, ok := driver.(Mkdir); ok {
				flags |= drivertypes.CapabilityMkdirFile
			}

			// 检查是否实现 Move 接口
			if _, ok := driver.(Move); ok {
				flags |= drivertypes.CapabilityMoveFile
			}

			// 检查是否实现 Copy 接口
			if _, ok := driver.(Copy); ok {
				flags |= drivertypes.CapabilityCopyFile
			}

			// 检查是否实现 Rename 接口
			if _, ok := driver.(Rename); ok {
				flags |= drivertypes.CapabilityRenameFile
			}

			// 检查是否实现 Remove 接口
			if _, ok := driver.(Remove); ok {
				flags |= drivertypes.CapabilityRemoveFile
			}

			// 检查是否实现 Put 接口
			if _, ok := driver.(Put); ok {
				flags |= drivertypes.CapabilityUploadFile
			}
			properties.Capabilitys = flags
		}

		return properties
	}

	exports.Exports.Driver.GetFormMeta = func(self cm.Rep) (result cm.List[exports.FormField]) {
		driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			panic("DriverErrorsInvalidHandle")
		}
		return cm.ToList(driver.GetFormMeta())
	}

	exports.Exports.Driver.Init = func(self, pctx cm.Rep) (result adapter.Result) {
		if driver, ok := DriverManager.Get(uint32(self)); ok {
			ctx, cancel := adapter.WarpCancellable(pctx)
			defer cancel()

			if err := driver.Init(ctx); err != nil {
				return cm.Err[adapter.Result](adapter.ErrorToDriverError(err))
			}
			return cm.OK[adapter.Result](struct{}{})
		}
		return cm.Err[adapter.Result](drivertypes.DriverErrorsInvalidHandle())
	}

	exports.Exports.Driver.Drop = func(self, pctx cm.Rep) (result adapter.Result) {
		if driver, ok := DriverManager.Get(uint32(self)); ok {
			ctx, cancel := adapter.WarpCancellable(pctx)
			defer cancel()

			if err := driver.Drop(ctx); err != nil {
				return cm.Err[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](adapter.ErrorToDriverError(err))
			}
		}
		return cm.OK[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](struct{}{})
	}

	exports.Exports.Driver.GetFile = func(self, pctx cm.Rep, path string) (result adapter.ResultObject) {
		if driver, ok := DriverManager.Get(uint32(self)); ok {
			if driver, ok := driver.(Getter); ok {
				ctx, cancel := adapter.WarpCancellable(pctx)
				defer cancel()
				obj, err := driver.Get(ctx, path)
				if err != nil {
					return cm.Err[adapter.ResultObject](adapter.ErrorToDriverError(err))
				}
				return cm.OK[adapter.ResultObject](*obj)
			}
			return cm.Err[adapter.ResultObject](drivertypes.DriverErrorsNotImplemented())
		}
		return cm.Err[adapter.ResultObject](drivertypes.DriverErrorsInvalidHandle())
	}

	exports.Exports.Driver.GetRoot = func(self, pctx cm.Rep) adapter.ResultObject {
		driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultObject](drivertypes.DriverErrorsInvalidHandle())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		root, err := driver.GetRoot(ctx)
		if err != nil {
			return cm.Err[adapter.ResultObject](adapter.ErrorToDriverError(err))
		}
		return cm.OK[adapter.ResultObject](*root)
	}

	exports.Exports.Driver.ListFiles = func(self, pctx cm.Rep, dir exports.Object) (result adapter.ResultObjects) {
		driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultObjects](drivertypes.DriverErrorsInvalidHandle())
		}
		// driver, ok := _driver.(Reader)
		// if !ok {
		// 	return cm.Err[adapter.ResultObjects](drivertypes.DriverErrorsNotImplemented())
		// }

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		objs, err := driver.ListFiles(ctx, dir)
		if err != nil {
			return cm.Err[adapter.ResultObjects](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkObjects(objs)
	}

	exports.Exports.Driver.LinkFile = func(self, pctx cm.Rep, file exports.Object, args exports.LinkArgs) (result cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]) {
		driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]](drivertypes.DriverErrorsInvalidHandle())
		}
		// driver, ok := _driver.(Reader)
		// if !ok {
		// 	return cm.Err[cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]](drivertypes.DriverErrorsNotImplemented())
		// }

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		link, newFile, err := driver.LinkFile(ctx, file, args)
		if err != nil {
			return cm.Err[cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]](adapter.ErrorToDriverError(err))
		}

		return cm.OK[cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]](exports.LinkResult{
			File:     adapter.OptionObject(newFile),
			Resource: *link,
		})
	}

	exports.Exports.Driver.LinkRange = func(self, pctx cm.Rep, file exports.Object, args exports.LinkArgs, range_ exports.RangeSpec) (result cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](drivertypes.DriverErrorsInvalidHandle())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		driver, ok := _driver.(StreamReader)
		if !ok {
			return cm.Err[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](drivertypes.DriverErrorsNotImplemented())
		}

		stream := adapter.NewOutputStream(range_.Stream)
		err := driver.LinkRange(ctx, file, args, range_, &stream)
		if err != nil {
			return cm.Err[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](adapter.ErrorToDriverError(err))
		}

		return cm.OK[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](struct{}{})
	}

	exports.Exports.Driver.MakeDir = func(self, pctx cm.Rep, dir exports.Object, name string) (result adapter.ResultOptionObject) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Mkdir)
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		obj, err := driver.MakeDir(ctx, dir, name)
		if err != nil {
			return cm.Err[adapter.ResultOptionObject](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkOptionObject(obj)
	}

	exports.Exports.Driver.RenameFile = func(self cm.Rep, pctx cm.Rep, file exports.Object, newName string) (result adapter.ResultOptionObject) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Rename)
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		obj, err := driver.Rename(ctx, file, newName)
		if err != nil {
			return cm.Err[adapter.ResultOptionObject](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkOptionObject(obj)
	}

	exports.Exports.Driver.MoveFile = func(self, pctx cm.Rep, file, toDir exports.Object) (result adapter.ResultOptionObject) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Move)
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		obj, err := driver.Move(ctx, file, toDir)
		if err != nil {
			return cm.Err[adapter.ResultOptionObject](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkOptionObject(obj)
	}

	exports.Exports.Driver.RemoveFile = func(self, pctx cm.Rep, file exports.Object) (result adapter.Result) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.Result](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Remove)
		if !ok {
			return cm.Err[adapter.Result](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		err := driver.Remove(ctx, file)
		if err != nil {
			return cm.Err[adapter.Result](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOk()
	}

	exports.Exports.Driver.CopyFile = func(self, pctx cm.Rep, file, toDir exports.Object) (result adapter.ResultOptionObject) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Copy)
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		obj, err := driver.Copy(ctx, file, toDir)
		if err != nil {
			return cm.Err[adapter.ResultOptionObject](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkOptionObject(obj)
	}

	exports.Exports.Driver.UploadFile = func(self, pctx cm.Rep, dir exports.Object, req exports.UploadRequest) (result adapter.ResultOptionObject) {
		_driver, ok := DriverManager.Get(uint32(self))
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsInvalidHandle())
		}
		driver, ok := _driver.(Put)
		if !ok {
			return cm.Err[adapter.ResultOptionObject](drivertypes.DriverErrorsNotImplemented())
		}

		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		obj, err := driver.Put(ctx, dir, adapter.UploadRequest{UploadRequest: req})
		if err != nil {
			return cm.Err[adapter.ResultOptionObject](adapter.ErrorToDriverError(err))
		}
		return adapter.ReturnOkOptionObject(obj)
	}
}
