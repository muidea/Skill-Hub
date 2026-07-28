package initiator

import (
	"context"
	"fmt"
	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"

	"github.com/muidea/magicCommon/framework/plugin/common"
)

var initiatorMgr = common.NewPluginMgr("initiator")

func Register(initiator any) {
	if err := RegisterE(initiator); err != nil {
		slog.Error("register initiator failed", "error", err)
	}
}

func RegisterE(initiator any) error {
	return initiatorMgr.Register(initiator)
}

func MustRegister(initiator any) {
	if err := RegisterE(initiator); err != nil {
		panic(err)
	}
}

func GetEntity[T any](id string, maskType T) (ret T, err *cd.Error) {
	entityVal, entityErr := initiatorMgr.GetEntity(id)
	if entityErr != nil {
		err = entityErr
		return
	}

	eVal, eOK := entityVal.(T)
	if !eOK {
		err = cd.NewError(cd.Unexpected, fmt.Sprintf("initiator:%s type not match", id))
		return
	}

	ret = eVal
	return
}

func Setup(ctx context.Context, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) *cd.Error {
	return initiatorMgr.Setup(ctx, eventHub, backgroundRoutine)
}

func Run(ctx context.Context) *cd.Error {
	return initiatorMgr.Run(ctx)
}

func Teardown(ctx context.Context) {
	initiatorMgr.Teardown(ctx)
}
