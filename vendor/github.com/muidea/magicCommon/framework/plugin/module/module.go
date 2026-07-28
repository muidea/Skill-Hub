package module

import (
	"context"
	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"

	"github.com/muidea/magicCommon/framework/plugin/common"
)

var moduleMgr = common.NewPluginMgr("module")

func Register(module any) {
	if err := RegisterE(module); err != nil {
		slog.Error("register module failed", "error", err)
	}
}

func RegisterE(module any) error {
	return moduleMgr.Register(module)
}

func MustRegister(module any) {
	if err := RegisterE(module); err != nil {
		panic(err)
	}
}

func Setup(ctx context.Context, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) *cd.Error {
	return moduleMgr.Setup(ctx, eventHub, backgroundRoutine)
}

func Run(ctx context.Context) *cd.Error {
	return moduleMgr.Run(ctx)
}

func Teardown(ctx context.Context) {
	moduleMgr.Teardown(ctx)
}
