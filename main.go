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

func RegisterDriver(driver Driver) {
	exports.Exports.SetHandle = func(handle uint32) {
		hostHeadle = handle
	}

	exports.Exports.GetProperties = func() (result exports.DriverProps) {
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

	exports.Exports.GetFormMeta = func() (result cm.List[exports.FormField]) {
		return cm.ToList(driver.GetFormMeta())
	}

	exports.Exports.Init = func(pctx cm.Rep) (result adapter.Result) {
		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		if err := driver.Init(ctx); err != nil {
			return cm.Err[adapter.Result](adapter.ErrorToDriverError(err))
		}
		return cm.OK[adapter.Result](struct{}{})
	}

	exports.Exports.Drop = func(pctx cm.Rep) (result adapter.Result) {
		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		if err := driver.Drop(ctx); err != nil {
			return cm.Err[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](adapter.ErrorToDriverError(err))
		}
		return cm.OK[cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]](struct{}{})
	}

	exports.Exports.GetFile = func(pctx cm.Rep, path string) (result adapter.ResultObject) {
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

	exports.Exports.GetRoot = func(pctx cm.Rep) adapter.ResultObject {
		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()

		root, err := driver.GetRoot(ctx)
		if err != nil {
			return cm.Err[adapter.ResultObject](adapter.ErrorToDriverError(err))
		}
		return cm.OK[adapter.ResultObject](*root)
	}

	exports.Exports.ListFiles = func(pctx cm.Rep, dir exports.Object) (result adapter.ResultObjects) {
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

	exports.Exports.LinkFile = func(pctx cm.Rep, file exports.Object, args exports.LinkArgs) (result cm.Result[exports.LinkResultShape, exports.LinkResult, exports.DriverErrors]) {
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

	exports.Exports.LinkRange = func(pctx cm.Rep, file exports.Object, args exports.LinkArgs, range_ exports.RangeSpec) (result cm.Result[exports.DriverErrors, struct{}, exports.DriverErrors]) {
		ctx, cancel := adapter.WarpCancellable(pctx)
		defer cancel()
		driver, ok := driver.(StreamReader)
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

	exports.Exports.MakeDir = func(pctx cm.Rep, dir exports.Object, name string) (result adapter.ResultOptionObject) {
		driver, ok := driver.(Mkdir)
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

	exports.Exports.RenameFile = func(pctx cm.Rep, file exports.Object, newName string) (result adapter.ResultOptionObject) {
		driver, ok := driver.(Rename)
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

	exports.Exports.MoveFile = func(pctx cm.Rep, file, toDir exports.Object) (result adapter.ResultOptionObject) {
		driver, ok := driver.(Move)
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

	exports.Exports.RemoveFile = func(pctx cm.Rep, file exports.Object) (result adapter.Result) {
		driver, ok := driver.(Remove)
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

	exports.Exports.CopyFile = func(pctx cm.Rep, file, toDir exports.Object) (result adapter.ResultOptionObject) {
		driver, ok := driver.(Copy)
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

	exports.Exports.UploadFile = func(pctx cm.Rep, dir exports.Object, req exports.UploadRequest) (result adapter.ResultOptionObject) {
		driver, ok := driver.(Put)
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
