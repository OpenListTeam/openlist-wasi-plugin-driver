package adapter

import (
	"context"
	"time"

	driverexports "github.com/OpenListTeam/openlist-wasi-plugin-driver/binding/openlist/plugin-driver/exports"

	"go.bytecodealliance.org/cm"
)

func WarpCancellable(pctx cm.Rep) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	poll := cm.Reinterpret[driverexports.Cancellable]((uint32)(pctx)).Subscribe()
	cancelDrop := func() {
		cancel()
		poll.ResourceDrop()
	}
	go func() {
		// NOTE: 使用Block会导致后续调度卡死，可能是tinygo的缺陷
		// poll.Block()
		for !poll.Ready() && ctx.Err() == nil {
			time.Sleep(time.Millisecond * 200)
		}
		cancelDrop()
	}()
	return ctx, cancelDrop
}
